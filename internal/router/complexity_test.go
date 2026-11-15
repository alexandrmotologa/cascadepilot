package router

import (
	"testing"

	"github.com/alexandrmotologa/cascadepilot/internal/provider"
)

func TestComplexityScorer_SimpleQueries(t *testing.T) {
	scorer := NewComplexityScorer()

	simpleReq := &provider.ChatCompletionRequest{
		Messages: []provider.Message{
			{Role: "user", Content: "Classify this sentiment as positive: Great food and fast delivery."},
		},
	}

	result := scorer.Score(simpleReq, 0.75)
	if result.Score > 0.40 {
		t.Errorf("expected low complexity score for simple query, got %.2f", result.Score)
	}
	if result.BypassDirectToTier2 {
		t.Errorf("expected bypass to be false for simple query")
	}
}

func TestComplexityScorer_ComplexQueries(t *testing.T) {
	scorer := NewComplexityScorer()

	complexReq := &provider.ChatCompletionRequest{
		Messages: []provider.Message{
			{Role: "user", Content: "Step by step, derive the formal mathematical proof for consensus and split-brain in a distributed lock under clock drift and race condition. Analyze tradeoffs and closed-form limits."},
		},
	}

	result := scorer.Score(complexReq, 0.50)
	if result.Score < 0.50 {
		t.Errorf("expected high complexity score for complex query, got %.2f", result.Score)
	}
}

func TestComplexityScorer_ExplicitBypass(t *testing.T) {
	scorer := NewComplexityScorer()

	req := &provider.ChatCompletionRequest{
		Messages: []provider.Message{
			{Role: "user", Content: "Hello [bypass-cascade] please handle directly."},
		},
	}

	result := scorer.Score(req, 0.75)
	if !result.BypassDirectToTier2 {
		t.Errorf("expected explicit bypass to trigger direct tier 2")
	}
}
