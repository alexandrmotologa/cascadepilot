package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/cascadepilot/internal/config"
	"github.com/alexandrmotologa/cascadepilot/internal/finops"
	"github.com/alexandrmotologa/cascadepilot/internal/provider"
	"github.com/alexandrmotologa/cascadepilot/internal/router"
)

var startTime = time.Now()

// ProxyHandler serves OpenAI and Anthropic compatible gateway endpoints.
type ProxyHandler struct {
	cfg    *config.Config
	engine *router.Engine
	store  *finops.Store
}

// NewProxyHandler creates a new reverse proxy handler.
func NewProxyHandler(cfg *config.Config, engine *router.Engine, store *finops.Store) *ProxyHandler {
	return &ProxyHandler{
		cfg:    cfg,
		engine: engine,
		store:  store,
	}
}

// HandleChatCompletions handles OpenAI /v1/chat/completions.
func (p *ProxyHandler) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req provider.ChatCompletionRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, fmt.Sprintf("invalid json payload: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.Messages) == 0 {
		http.Error(w, "messages array cannot be empty", http.StatusBadRequest)
		return
	}

	tenantID, department := p.extractTenant(r)
	ctx := r.Context()

	// Non-streaming completion
	if !req.Stream {
		resp, record, err := p.engine.ExecuteWithTenant(ctx, &req, tenantID, department)
		if err != nil {
			http.Error(w, fmt.Sprintf("execution error: %v", err), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cascadepilot-Routed-Tier", record.RoutedTier)
		w.Header().Set("X-Cascadepilot-Escalated", fmt.Sprintf("%t", record.Escalated))
		w.Header().Set("X-Cascadepilot-Saved-USD", fmt.Sprintf("%.6f", record.SavedCostUSD))
		w.Header().Set("X-Cascadepilot-Confidence", fmt.Sprintf("%.3f", record.Tier1Confidence))
		w.Header().Set("X-Cascadepilot-Department", department)
		w.Header().Set("X-Cascadepilot-Cache-Hit", fmt.Sprintf("%t", record.CacheHit))

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Streaming completion
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	chunkChan := make(chan provider.StreamChunk, 100)

	go func() {
		_, _ = p.engine.ExecuteStream(ctx, &req, chunkChan)
	}()

	for chunk := range chunkChan {
		chunkBytes, err := json.Marshal(chunk)
		if err != nil {
			continue
		}
		fmt.Fprintf(w, "data: %s\n\n", string(chunkBytes))
		flusher.Flush()
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func (p *ProxyHandler) extractTenant(r *http.Request) (string, string) {
	tenantID := "tenant_default"
	department := "General"

	if deptHeader := r.Header.Get("X-Department"); deptHeader != "" {
		department = deptHeader
	}

	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if strings.HasPrefix(token, "cp-") && p.store != nil {
			if tenant, found, _ := p.store.GetTenantByKey(token); found {
				tenantID = tenant.ID
				department = tenant.Department
			}
		}
	}

	return tenantID, department
}

// HandleAnthropicMessages translates and handles Anthropic /v1/messages requests.
func (p *ProxyHandler) HandleAnthropicMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var anthropicReq struct {
		Model       string `json:"model"`
		MaxTokens   int    `json:"max_tokens"`
		Temperature *float64 `json:"temperature"`
		Messages    []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(bodyBytes, &anthropicReq); err != nil {
		http.Error(w, "invalid anthropic payload", http.StatusBadRequest)
		return
	}

	messages := make([]provider.Message, 0, len(anthropicReq.Messages))
	for _, m := range anthropicReq.Messages {
		messages = append(messages, provider.Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	openAIReq := &provider.ChatCompletionRequest{
		Model:       anthropicReq.Model,
		Messages:    messages,
		Temperature: anthropicReq.Temperature,
		MaxTokens:   &anthropicReq.MaxTokens,
		Stream:      false,
	}

	tenantID, department := p.extractTenant(r)
	resp, _, err := p.engine.ExecuteWithTenant(r.Context(), openAIReq, tenantID, department)
	if err != nil {
		http.Error(w, fmt.Sprintf("execution error: %v", err), http.StatusBadGateway)
		return
	}

	responseText := ""
	if len(resp.Choices) > 0 {
		responseText = resp.Choices[0].Message.ContentString()
	}

	anthropicResp := map[string]interface{}{
		"id":   resp.ID,
		"type": "message",
		"role": "assistant",
		"content": []map[string]string{
			{
				"type": "text",
				"text": responseText,
			},
		},
		"model": resp.Model,
		"usage": map[string]int{
			"input_tokens":  resp.Usage.PromptTokens,
			"output_tokens": resp.Usage.CompletionTokens,
		},
		"stop_reason": "end_turn",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(anthropicResp)
}

// HandleModels returns available virtual and configured models.
func (p *ProxyHandler) HandleModels(w http.ResponseWriter, r *http.Request) {
	frontierModel := p.cfg.Tiers.Tier3.Model
	if frontierModel == "" {
		frontierModel = p.cfg.Tiers.Tier2.Model
	}

	models := map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{
				"id":          "cascadepilot-auto",
				"object":      "model",
				"created":     1715000000,
				"owned_by":    "cascadepilot",
				"description": "Default speculative cascading model router",
			},
			{
				"id":          p.cfg.Tiers.Tier1.Model,
				"object":      "model",
				"created":     1715000000,
				"owned_by":    p.cfg.Tiers.Tier1.Provider,
				"description": "Tier 1 low-cost SLM",
			},
			{
				"id":          p.cfg.Tiers.Tier2.Model,
				"object":      "model",
				"created":     1715000000,
				"owned_by":    p.cfg.Tiers.Tier2.Provider,
				"description": "Tier 2 mid-tier reasoning model",
			},
			{
				"id":          frontierModel,
				"object":      "model",
				"created":     1715000000,
				"owned_by":    p.cfg.Tiers.Tier3.Provider,
				"description": "Tier 3 flagship frontier model",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(models)
}

// HandleHealth provides a health probe endpoint.
func (p *ProxyHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	theta, tau := p.engine.GetThresholds()
	frontierModel := p.cfg.Tiers.Tier3.Model
	if frontierModel == "" {
		frontierModel = p.cfg.Tiers.Tier2.Model
	}

	health := map[string]interface{}{
		"status":         "healthy",
		"service":        "cascadepilot",
		"uptime_sec":     int(time.Since(startTime).Seconds()),
		"theta_bypass":   theta,
		"tau_confidence": tau,
		"tier1_model":    p.cfg.Tiers.Tier1.Model,
		"tier2_model":    p.cfg.Tiers.Tier2.Model,
		"tier3_model":    frontierModel,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(health)
}

// HandleMetrics provides expanded Prometheus metrics.
func (p *ProxyHandler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	var sb strings.Builder

	summary, _ := p.store.GetSummary()

	sb.WriteString("# HELP cascadepilot_uptime_seconds Total seconds since process start\n")
	sb.WriteString("# TYPE cascadepilot_uptime_seconds counter\n")
	fmt.Fprintf(&sb, "cascadepilot_uptime_seconds %d\n\n", int(time.Since(startTime).Seconds()))

	sb.WriteString("# HELP cascadepilot_requests_total Total number of routed requests\n")
	sb.WriteString("# TYPE cascadepilot_requests_total counter\n")
	if summary != nil {
		fmt.Fprintf(&sb, "cascadepilot_requests_total{tier=\"tier0_cache\"} %d\n", summary.Tier0CacheHits)
		fmt.Fprintf(&sb, "cascadepilot_requests_total{tier=\"tier1\"} %d\n", summary.Tier1Accepted)
		fmt.Fprintf(&sb, "cascadepilot_requests_total{tier=\"tier2_mid\"} %d\n", summary.Tier2MidtierCount)
		fmt.Fprintf(&sb, "cascadepilot_requests_total{tier=\"tier3_direct\"} %d\n", summary.Tier3Direct)
		fmt.Fprintf(&sb, "cascadepilot_requests_total{tier=\"tier3_escalated\"} %d\n", summary.Tier3Escalated)
		fmt.Fprintf(&sb, "cascadepilot_requests_total{status=\"all\"} %d\n\n", summary.TotalRequests)

		sb.WriteString("# HELP cascadepilot_cost_actual_usd_total Total actual USD expenditure\n")
		sb.WriteString("# TYPE cascadepilot_cost_actual_usd_total gauge\n")
		fmt.Fprintf(&sb, "cascadepilot_cost_actual_usd_total %.6f\n\n", summary.TotalActualCostUSD)

		sb.WriteString("# HELP cascadepilot_cost_saved_usd_total Total USD savings realized\n")
		sb.WriteString("# TYPE cascadepilot_cost_saved_usd_total gauge\n")
		fmt.Fprintf(&sb, "cascadepilot_cost_saved_usd_total %.6f\n\n", summary.TotalSavedUSD)

		sb.WriteString("# HELP cascadepilot_quality_parity_percent Retained quality percentage\n")
		sb.WriteString("# TYPE cascadepilot_quality_parity_percent gauge\n")
		fmt.Fprintf(&sb, "cascadepilot_quality_parity_percent %.2f\n\n", summary.QualityParityScore)
	}

	_, _ = w.Write([]byte(sb.String()))
}
