package router

import (
	"context"
	"fmt"

	"github.com/alexandrmotologa/cascadepilot/internal/config"
	"github.com/alexandrmotologa/cascadepilot/internal/provider"
)

// FallbackHandler guarantees graceful failover between upstream providers.
type FallbackHandler struct {
	client provider.ProviderClient
}

// NewFallbackHandler creates a new fallback handler.
func NewFallbackHandler(client provider.ProviderClient) *FallbackHandler {
	return &FallbackHandler{client: client}
}

// ExecuteWithFallback tries Tier 1; if an unrecoverable upstream error occurs, fails over to Tier 2.
func (f *FallbackHandler) ExecuteWithFallback(
	ctx context.Context,
	req *provider.ChatCompletionRequest,
	tier1, tier2 config.TierConfig,
	complexity float64,
) (*provider.ChatCompletionResponse, string, error) {
	resp, err := f.client.Complete(ctx, req, tier1, "tier1", complexity)
	if err == nil {
		return resp, "tier1", nil
	}

	// Failover directly to Tier 2
	resp2, err2 := f.client.Complete(ctx, req, tier2, "tier2", complexity)
	if err2 != nil {
		return nil, "", fmt.Errorf("both tier1 (%v) and tier2 (%v) failed", err, err2)
	}

	return resp2, "tier2_fallback", nil
}
