package cache

import (
	"testing"
	"time"

	"github.com/alexandrmotologa/cascadepilot/internal/provider"
)

func TestPromptCache_HitAndMiss(t *testing.T) {
	c := NewPromptCache(100, 1*time.Hour)

	req := &provider.ChatCompletionRequest{
		Messages: []provider.Message{
			{Role: "user", Content: "What is the capital of France?"},
		},
	}

	// Initial miss
	if _, found := c.Get(req); found {
		t.Errorf("expected miss on empty cache")
	}

	resp := &provider.ChatCompletionResponse{
		ID:    "resp-1",
		Model: "gemini-1.5-flash",
		Choices: []provider.Choice{
			{
				Index:   0,
				Message: provider.Message{Role: "assistant", Content: "Paris"},
			},
		},
		Usage: provider.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	// Set cache
	c.Set(req, resp)

	// Hit
	cached, found := c.Get(req)
	if !found {
		t.Fatalf("expected cache hit after set")
	}
	if cached.Choices[0].Message.ContentString() != "Paris" {
		t.Errorf("expected cached content 'Paris', got %s", cached.Choices[0].Message.ContentString())
	}

	stats := c.Stats()
	if stats.Hits != 1 || stats.Misses != 1 {
		t.Errorf("expected 1 hit and 1 miss, got %d hits and %d misses", stats.Hits, stats.Misses)
	}
	if stats.HitRatio != 50.0 {
		t.Errorf("expected 50%% hit ratio, got %.1f%%", stats.HitRatio)
	}
}

func TestPromptCache_Expiration(t *testing.T) {
	// Cache with 10 millisecond TTL
	c := NewPromptCache(100, 10*time.Millisecond)

	req := &provider.ChatCompletionRequest{
		Messages: []provider.Message{
			{Role: "user", Content: "Short lived prompt"},
		},
	}

	resp := &provider.ChatCompletionResponse{
		Choices: []provider.Choice{
			{Index: 0, Message: provider.Message{Role: "assistant", Content: "Ephemeral answer"}},
		},
	}

	c.Set(req, resp)

	// Immediate hit
	if _, found := c.Get(req); !found {
		t.Fatalf("expected immediate hit")
	}

	// Wait for TTL expiration
	time.Sleep(25 * time.Millisecond)

	if _, found := c.Get(req); found {
		t.Errorf("expected item to expire after TTL")
	}
}
