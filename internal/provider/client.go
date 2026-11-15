package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alexandrmotologa/cascadepilot/internal/config"
)

// ProviderClient defines the contract for communicating with model tiers.
type ProviderClient interface {
	Complete(ctx context.Context, req *ChatCompletionRequest, tier config.TierConfig, tierName string, complexityScore float64) (*ChatCompletionResponse, error)
	CompleteStream(ctx context.Context, req *ChatCompletionRequest, tier config.TierConfig, tierName string, complexityScore float64, chunkChan chan<- StreamChunk) error
}

// UnifiedClient manages calls to OpenAI, Anthropic, Gemini, Ollama, and Mock providers.
type UnifiedClient struct {
	httpClient   *http.Client
	mockProvider *MockProvider
	mockEnabled  bool
}

// NewUnifiedClient creates a multi-provider upstream client.
func NewUnifiedClient(simCfg config.SimulationConfig) *UnifiedClient {
	return &UnifiedClient{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		mockProvider: NewMockProvider(simCfg.SimulatedNetworkLatencyMs),
		mockEnabled:  simCfg.MockEnabled,
	}
}

// Complete executes a non-streaming chat completion request.
func (c *UnifiedClient) Complete(ctx context.Context, req *ChatCompletionRequest, tier config.TierConfig, tierName string, complexityScore float64) (*ChatCompletionResponse, error) {
	apiKey := getAPIKey(tier.APIKeyEnv)

	// Fallback to mock if configured or if no API key is available
	if tier.Provider == "mock" || (apiKey == "" && c.mockEnabled) {
		return c.mockProvider.Complete(ctx, req, tierName, complexityScore)
	}

	switch strings.ToLower(tier.Provider) {
	case "anthropic":
		return c.completeAnthropic(ctx, req, tier, apiKey)
	case "openai", "google", "gemini", "ollama":
		return c.completeOpenAICompatible(ctx, req, tier, apiKey)
	default:
		// Default to OpenAI-compatible endpoint
		return c.completeOpenAICompatible(ctx, req, tier, apiKey)
	}
}

// CompleteStream executes a streaming chat completion request.
func (c *UnifiedClient) CompleteStream(ctx context.Context, req *ChatCompletionRequest, tier config.TierConfig, tierName string, complexityScore float64, chunkChan chan<- StreamChunk) error {
	apiKey := getAPIKey(tier.APIKeyEnv)

	if tier.Provider == "mock" || (apiKey == "" && c.mockEnabled) {
		return c.mockProvider.CompleteStream(ctx, req, tierName, complexityScore, chunkChan)
	}

	switch strings.ToLower(tier.Provider) {
	case "anthropic":
		return c.streamAnthropic(ctx, req, tier, apiKey, chunkChan)
	default:
		return c.streamOpenAICompatible(ctx, req, tier, apiKey, chunkChan)
	}
}

func (c *UnifiedClient) completeOpenAICompatible(ctx context.Context, req *ChatCompletionRequest, tier config.TierConfig, apiKey string) (*ChatCompletionResponse, error) {
	timeout := time.Duration(tier.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	endpoint := strings.TrimRight(tier.Endpoint, "/")
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}

	// Prepare payload with target model name
	payloadReq := *req
	payloadReq.Model = tier.Model
	payloadReq.Stream = false

	bodyBytes, err := json.Marshal(payloadReq)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctxTimeout, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("upstream request error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read upstream response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("upstream error (%d): %s", resp.StatusCode, string(respBytes))
	}

	var completion ChatCompletionResponse
	if err := json.Unmarshal(respBytes, &completion); err != nil {
		return nil, fmt.Errorf("failed to parse upstream json: %w", err)
	}

	return &completion, nil
}

func (c *UnifiedClient) streamOpenAICompatible(ctx context.Context, req *ChatCompletionRequest, tier config.TierConfig, apiKey string, chunkChan chan<- StreamChunk) error {
	defer close(chunkChan)

	endpoint := strings.TrimRight(tier.Endpoint, "/")
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}

	payloadReq := *req
	payloadReq.Model = tier.Model
	payloadReq.Stream = true

	bodyBytes, err := json.Marshal(payloadReq)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("upstream streaming error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upstream stream error (%d): %s", resp.StatusCode, string(respBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if strings.TrimSpace(data) == "[DONE]" {
			break
		}

		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err == nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case chunkChan <- chunk:
			}
		}
	}

	return scanner.Err()
}

// Anthropic translation helpers
func (c *UnifiedClient) completeAnthropic(ctx context.Context, req *ChatCompletionRequest, tier config.TierConfig, apiKey string) (*ChatCompletionResponse, error) {
	endpoint := strings.TrimRight(tier.Endpoint, "/")
	if !strings.HasSuffix(endpoint, "/messages") {
		endpoint += "/messages"
	}

	anthropicReq := map[string]interface{}{
		"model":      tier.Model,
		"max_tokens": 4096,
		"messages":   translateMessagesToAnthropic(req.Messages),
	}
	if req.Temperature != nil {
		anthropicReq["temperature"] = *req.Temperature
	}

	bodyBytes, _ := json.Marshal(anthropicReq)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("anthropic error (%d): %s", resp.StatusCode, string(respBytes))
	}

	var anthropicResp struct {
		ID      string `json:"id"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		StopReason string `json:"stop_reason"`
	}

	if err := json.Unmarshal(respBytes, &anthropicResp); err != nil {
		return nil, err
	}

	var sb strings.Builder
	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}

	return &ChatCompletionResponse{
		ID:      anthropicResp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   tier.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: sb.String(),
				},
				FinishReason: anthropicResp.StopReason,
			},
		},
		Usage: Usage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}, nil
}

func (c *UnifiedClient) streamAnthropic(ctx context.Context, req *ChatCompletionRequest, tier config.TierConfig, apiKey string, chunkChan chan<- StreamChunk) error {
	// For simplicity and resilience, stream fallback can translate completed Anthropic response
	resp, err := c.completeAnthropic(ctx, req, tier, apiKey)
	if err != nil {
		defer close(chunkChan)
		return err
	}

	defer close(chunkChan)
	if len(resp.Choices) > 0 {
		chunkChan <- StreamChunk{
			ID:      resp.ID,
			Object:  "chat.completion.chunk",
			Created: resp.Created,
			Model:   resp.Model,
			Choices: []StreamChoice{
				{
					Index: 0,
					Delta: StreamDelta{
						Role:    "assistant",
						Content: resp.Choices[0].Message.ContentString(),
					},
					FinishReason: &resp.Choices[0].FinishReason,
				},
			},
			Usage: &resp.Usage,
		}
	}
	return nil
}

func translateMessagesToAnthropic(messages []Message) []map[string]string {
	result := make([]map[string]string, 0, len(messages))
	for _, m := range messages {
		role := m.Role
		if role == "system" {
			role = "user"
		}
		result = append(result, map[string]string{
			"role":    role,
			"content": m.ContentString(),
		})
	}
	return result
}

func getAPIKey(envName string) string {
	if envName == "" {
		return ""
	}
	return os.Getenv(envName)
}
