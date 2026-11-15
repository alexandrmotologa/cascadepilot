package provider

import (
	"encoding/json"
)

// ChatCompletionRequest conforms to the OpenAI chat completions API format.
type ChatCompletionRequest struct {
	Model            string                 `json:"model"`
	Messages         []Message              `json:"messages"`
	Temperature      *float64               `json:"temperature,omitempty"`
	TopP             *float64               `json:"top_p,omitempty"`
	MaxTokens        *int                   `json:"max_tokens,omitempty"`
	Stream           bool                   `json:"stream,omitempty"`
	ResponseFormat   *ResponseFormat        `json:"response_format,omitempty"`
	LogProbs         bool                   `json:"logprobs,omitempty"`
	TopLogProbs      *int                   `json:"top_logprobs,omitempty"`
	Stop             interface{}            `json:"stop,omitempty"`
	PresencePenalty  *float64               `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64               `json:"frequency_penalty,omitempty"`
	User             string                 `json:"user,omitempty"`
	Tools            []interface{}          `json:"tools,omitempty"`
	ExtraFields      map[string]interface{} `json:"-"`
}

// ResponseFormat specifies structured JSON or schema requests.
type ResponseFormat struct {
	Type   string                 `json:"type"` // "text", "json_object", "json_schema"
	Schema map[string]interface{} `json:"schema,omitempty"`
}

// Message represents a single chat conversation message.
type Message struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"` // Can be string or array of content parts
	Name       string      `json:"name,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// ContentString returns text representation of message content.
func (m Message) ContentString() string {
	switch v := m.Content.(type) {
	case string:
		return v
	case []interface{}:
		var result string
		for _, part := range v {
			if partMap, ok := part.(map[string]interface{}); ok {
				if text, ok := partMap["text"].(string); ok {
					result += text + "\n"
				}
			}
		}
		return result
	default:
		bytes, _ := json.Marshal(m.Content)
		return string(bytes)
	}
}

// ChatCompletionResponse represents the standard chat completion payload.
type ChatCompletionResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}

// Choice represents a completion candidate choice.
type Choice struct {
	Index        int           `json:"index"`
	Message      Message       `json:"message"`
	FinishReason string        `json:"finish_reason"`
	LogProbs     *ChoiceLogProbs `json:"logprobs,omitempty"`
}

// ChoiceLogProbs contains token log probability data.
type ChoiceLogProbs struct {
	Content []TokenLogProb `json:"content,omitempty"`
}

// TokenLogProb holds probability data for a single generated token.
type TokenLogProb struct {
	Token   string  `json:"token"`
	LogProb float64 `json:"logprob"`
	Bytes   []byte  `json:"bytes,omitempty"`
}

// Usage captures token counts for the transaction.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk represents a Server-Sent Events chunk for streaming responses.
type StreamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
	Usage   *Usage         `json:"usage,omitempty"`
}

// StreamChoice represents an incremental diff choice in a stream.
type StreamChoice struct {
	Index        int           `json:"index"`
	Delta        StreamDelta   `json:"delta"`
	FinishReason *string       `json:"finish_reason"`
	LogProbs     *ChoiceLogProbs `json:"logprobs,omitempty"`
}

// StreamDelta holds incremental content in a stream.
type StreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}
