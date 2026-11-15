package verifier

import (
	"math"
	"testing"

	"github.com/alexandrmotologa/cascadepilot/internal/provider"
)

func TestConfidenceAnalyzer(t *testing.T) {
	analyzer := NewConfidenceAnalyzer()

	// Empty logprobs fallback
	if conf := analyzer.ComputeLogProbConfidence(nil); conf != 0.85 {
		t.Errorf("expected 0.85 fallback for nil logprobs, got %.2f", conf)
	}

	// High confidence tokens
	highLogProbs := &provider.ChoiceLogProbs{
		Content: []provider.TokenLogProb{
			{Token: "test", LogProb: math.Log(0.95)},
			{Token: "token", LogProb: math.Log(0.92)},
		},
	}
	highConf := analyzer.ComputeLogProbConfidence(highLogProbs)
	if highConf < 0.90 {
		t.Errorf("expected confidence >= 0.90, got %.2f", highConf)
	}
}

func TestHeuristicChecker_Refusal(t *testing.T) {
	checker := NewHeuristicChecker()

	// Normal text
	penalty, _ := checker.CheckRefusalPenalty("Paris is the capital of France.")
	if penalty != 0.0 {
		t.Errorf("expected 0 penalty for normal text, got %.2f", penalty)
	}

	// Hedging/refusal text
	refusalText := "I am not completely sure about this topic, please consult an expert."
	penalty, reason := checker.CheckRefusalPenalty(refusalText)
	if penalty < 0.5 {
		t.Errorf("expected high penalty for refusal text, got %.2f (reason: %s)", penalty, reason)
	}
}

func TestSchemaGate(t *testing.T) {
	gate := NewSchemaGate()

	req := &provider.ChatCompletionRequest{
		ResponseFormat: &provider.ResponseFormat{Type: "json_object"},
	}

	// Valid JSON
	validJSON := `{"status": "ok", "count": 42}`
	adherence, _ := gate.EvaluateSchema(validJSON, req)
	if adherence != 1.0 {
		t.Errorf("expected adherence 1.0 for valid json, got %.2f", adherence)
	}

	// Valid JSON in markdown block
	mdJSON := "```json\n{\"status\": \"ok\"}\n```"
	adherence, _ = gate.EvaluateSchema(mdJSON, req)
	if adherence != 1.0 {
		t.Errorf("expected adherence 1.0 for markdown json block, got %.2f", adherence)
	}

	// Invalid JSON
	invalidJSON := "not a json object"
	adherence, reason := gate.EvaluateSchema(invalidJSON, req)
	if adherence != 0.0 {
		t.Errorf("expected adherence 0.0 for invalid json, got %.2f (reason: %s)", adherence, reason)
	}
}
