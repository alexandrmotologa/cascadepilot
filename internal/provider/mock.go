package provider

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// MockProvider generates deterministic completions with simulated token metrics.
type MockProvider struct {
	latencyMs int
}

// NewMockProvider creates a new mock engine with the given base latency.
func NewMockProvider(latencyMs int) *MockProvider {
	if latencyMs <= 0 {
		latencyMs = 10
	}
	return &MockProvider{latencyMs: latencyMs}
}

// Complete handles synchronous mock generation.
func (m *MockProvider) Complete(ctx context.Context, req *ChatCompletionRequest, tierName string, complexityScore float64) (*ChatCompletionResponse, error) {
	// Simulate lightweight network transmission
	jitter := time.Duration(m.latencyMs+rand.Intn(10)) * time.Millisecond
	select {
	case <-time.After(jitter):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	promptText := extractFullPrompt(req.Messages)
	promptTokens := estimateTokens(promptText)

	content, avgLogprob, finishReason := generateMockContent(promptText, tierName, complexityScore)
	completionTokens := estimateTokens(content)

	// Build logprobs
	words := strings.Fields(content)
	logprobs := make([]TokenLogProb, 0, len(words))
	for _, w := range words {
		// LogProb around log(avgLogprob)
		lp := math.Log(math.Max(0.01, math.Min(0.99, avgLogprob+(rand.Float64()*0.1-0.05))))
		logprobs = append(logprobs, TokenLogProb{
			Token:   " " + w,
			LogProb: lp,
		})
	}

	modelName := req.Model
	if modelName == "" || modelName == "cascadepilot-auto" {
		switch tierName {
		case "tier1":
			modelName = "gemini-1.5-flash"
		case "tier2":
			modelName = "gpt-4o-mini"
		default:
			modelName = "claude-3-5-sonnet"
		}
	}

	resp := &ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   modelName,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: finishReason,
				LogProbs: &ChoiceLogProbs{
					Content: logprobs,
				},
			},
		},
		Usage: Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}

	return resp, nil
}

// CompleteStream handles streaming generation chunk by chunk.
func (m *MockProvider) CompleteStream(ctx context.Context, req *ChatCompletionRequest, tierName string, complexityScore float64, chunkChan chan<- StreamChunk) error {
	defer close(chunkChan)

	promptText := extractFullPrompt(req.Messages)
	content, avgLogprob, finishReason := generateMockContent(promptText, tierName, complexityScore)

	modelName := req.Model
	if modelName == "" || modelName == "cascadepilot-auto" {
		switch tierName {
		case "tier1":
			modelName = "gemini-1.5-flash"
		case "tier2":
			modelName = "gpt-4o-mini"
		default:
			modelName = "claude-3-5-sonnet"
		}
	}

	id := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	created := time.Now().Unix()

	// Initial role chunk
	chunkChan <- StreamChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   modelName,
		Choices: []StreamChoice{
			{
				Index: 0,
				Delta: StreamDelta{Role: "assistant"},
			},
		},
	}

	words := strings.Fields(content)
	for i, w := range words {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(4+rand.Intn(6)) * time.Millisecond):
		}

		tokenStr := w + " "
		lp := math.Log(math.Max(0.01, math.Min(0.99, avgLogprob+(rand.Float64()*0.1-0.05))))

		var fr *string
		if i == len(words)-1 {
			fr = &finishReason
		}

		chunkChan <- StreamChunk{
			ID:      id,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   modelName,
			Choices: []StreamChoice{
				{
					Index: 0,
					Delta: StreamDelta{Content: tokenStr},
					LogProbs: &ChoiceLogProbs{
						Content: []TokenLogProb{
							{Token: tokenStr, LogProb: lp},
						},
					},
					FinishReason: fr,
				},
			},
		}
	}

	return nil
}

func extractFullPrompt(messages []Message) string {
	var sb strings.Builder
	for _, m := range messages {
		sb.WriteString(m.ContentString())
		sb.WriteString(" ")
	}
	return sb.String()
}

func estimateTokens(text string) int {
	words := len(strings.Fields(text))
	chars := len(text)
	// Standard heuristic ~ 4 chars per token or 1.3 tokens per word
	byChars := int(math.Ceil(float64(chars) / 4.0))
	byWords := int(math.Ceil(float64(words) * 1.3))
	if byChars > byWords {
		return int(math.Max(1, float64(byChars)))
	}
	return int(math.Max(1, float64(byWords)))
}

func generateMockContent(prompt, tierName string, complexityScore float64) (string, float64, string) {
	lower := strings.ToLower(prompt)

	// JSON format request
	if strings.Contains(lower, "json") {
		if strings.Contains(lower, "sarah connor") || strings.Contains(lower, "contact") {
			return `{"name": "Sarah Connor", "role": "Security Lead", "email": "sarah@cyberdyne.org"}`, 0.96, "stop"
		}
		return `{"status": "success", "processed": true, "timestamp": 1715000000}`, 0.95, "stop"
	}

	// Sentiment classification
	if strings.Contains(lower, "sentiment") {
		return "Positive. The review explicitly mentions that the package arrived two days early and was intact.", 0.98, "stop"
	}

	// Entity extraction
	if strings.Contains(lower, "extract") && strings.Contains(lower, "acme") {
		return "Company Name: Acme Corp\nAcquisition Price: $45 million\nYear: 2023", 0.97, "stop"
	}

	// SQL query
	if strings.Contains(lower, "sql") {
		return "SELECT id, name, email, created_at FROM customers WHERE created_at > '2024-01-01' ORDER BY created_at DESC;", 0.95, "stop"
	}

	// Translation
	if strings.Contains(lower, "translate") {
		return "Merci de vous être inscrit. Votre période d'essai gratuit est maintenant active.", 0.96, "stop"
	}

	// High complexity: distributed lock, markov chain, contract analysis, memory leak
	isHighComplexity := complexityScore > 0.65 ||
		strings.Contains(lower, "distributed lock") ||
		strings.Contains(lower, "redlock") ||
		strings.Contains(lower, "markov") ||
		strings.Contains(lower, "indemnification") ||
		strings.Contains(lower, "memory leak") ||
		strings.Contains(lower, "vector clocks")

	if isHighComplexity {
		if tierName == "tier1" {
			// Tier 1 generates a low-confidence draft or shows uncertainty
			return "I am not completely certain about the clock drift edge cases under network partition. It might be possible that a split brain occurs if the clock drifts beyond the threshold, but you should verify with specialized distributed systems documentation.", 0.52, "stop"
		}
		// Tier 2 generates a rigorous, high-confidence response
		return "Detailed Analysis: Split-brain in Redlock occurs when node clock drift exceeds the validity window delta (T_valid = MIN_VALIDITY - DRIFT). Specifically, if clock jump dt > (TTL - processing_time), a client may believe it holds an exclusive lease while another client acquires a quorum on the same resource. To prevent this, implement monotonically increasing epoch fencing tokens on the storage tier and verify state with consensus leases.", 0.94, "stop"
	}

	// Medium routine queries
	if tierName == "tier1" {
		return "Based on standard procedures, our policy allows remote work equipment reimbursement up to $500 annually for monitors, ergonomic chairs, and peripherals with valid receipts submitted within 30 days.", 0.91, "stop"
	}
	return "Under standard enterprise procedures, employees may claim reimbursement for authorized home office hardware up to the scheduled annual allowance upon submitting itemized receipts to accounting.", 0.96, "stop"
}
