package verifier

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/alexandrmotologa/cascadepilot/internal/provider"
)

var (
	jsonBlockRegex = regexp.MustCompile("(?s)```(?:json)?\\s*([\\{\\[].*?[\\}\\]])\\s*```")
)

// SchemaGate validates JSON syntax and schema compliance.
type SchemaGate struct{}

// NewSchemaGate creates a new validator.
func NewSchemaGate() *SchemaGate {
	return &SchemaGate{}
}

// EvaluateSchema checks if the completion satisfies requested JSON structures.
func (s *SchemaGate) EvaluateSchema(content string, req *provider.ChatCompletionRequest) (adherence float64, reason string) {
	isJSONExpected := false

	if req.ResponseFormat != nil && (req.ResponseFormat.Type == "json_object" || req.ResponseFormat.Type == "json_schema") {
		isJSONExpected = true
	} else {
		// Infer from prompt messages
		for _, m := range req.Messages {
			lower := strings.ToLower(m.ContentString())
			if strings.Contains(lower, "format as json") || strings.Contains(lower, "valid json") || strings.Contains(lower, "return json") {
				isJSONExpected = true
				break
			}
		}
	}

	if !isJSONExpected {
		return 1.0, ""
	}

	// Try direct JSON unmarshal
	trimmed := strings.TrimSpace(content)
	var raw interface{}
	if err := json.Unmarshal([]byte(trimmed), &raw); err == nil {
		return 1.0, ""
	}

	// Try extracting from markdown block
	matches := jsonBlockRegex.FindStringSubmatch(trimmed)
	if len(matches) > 1 {
		candidate := strings.TrimSpace(matches[1])
		if err := json.Unmarshal([]byte(candidate), &raw); err == nil {
			return 1.0, ""
		}
	}

	return 0.0, "invalid_json_format"
}
