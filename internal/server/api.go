package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/alexandrmotologa/cascadepilot/internal/cache"
	"github.com/alexandrmotologa/cascadepilot/internal/finops"
	"github.com/alexandrmotologa/cascadepilot/internal/provider"
	"github.com/alexandrmotologa/cascadepilot/internal/router"
)

// APIHandler handles FinOps reporting, multi-tenancy, cache, and comparison API routes.
type APIHandler struct {
	store       *finops.Store
	engine      *router.Engine
	promptCache *cache.PromptCache
}

// NewAPIHandler creates a new API controller.
func NewAPIHandler(store *finops.Store, engine *router.Engine, promptCache *cache.PromptCache) *APIHandler {
	return &APIHandler{
		store:       store,
		engine:      engine,
		promptCache: promptCache,
	}
}

// HandleSummary returns aggregated FinOps metrics.
func (a *APIHandler) HandleSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := a.store.GetSummary()
	if err != nil {
		http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summary)
}

// HandleWaterfall returns waterfall routing counts.
func (a *APIHandler) HandleWaterfall(w http.ResponseWriter, r *http.Request) {
	waterfall, err := a.store.GetWaterfall()
	if err != nil {
		http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(waterfall)
}

// HandleLatency returns latency statistics across tiers.
func (a *APIHandler) HandleLatency(w http.ResponseWriter, r *http.Request) {
	latency, err := a.store.GetLatencyMetrics()
	if err != nil {
		http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(latency)
}

// HandleRequests returns recent query transactions.
func (a *APIHandler) HandleRequests(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	requests, err := a.store.GetRecentRequests(limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(requests)
}

// HandleTenants handles multi-tenant listing and creation.
func (a *APIHandler) HandleTenants(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		tenants, err := a.store.ListTenantsWithSpend()
		if err != nil {
			http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(tenants)
		return
	}

	if r.Method == http.MethodPost {
		var payload struct {
			Name             string  `json:"name"`
			Department       string  `json:"department"`
			MonthlyBudgetUSD float64 `json:"monthly_budget_usd"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if payload.Name == "" || payload.Department == "" {
			http.Error(w, "name and department are required", http.StatusBadRequest)
			return
		}

		if payload.MonthlyBudgetUSD <= 0 {
			payload.MonthlyBudgetUSD = 500.0
		}

		// Generate random secure virtual key
		randBytes := make([]byte, 8)
		_, _ = rand.Read(randBytes)
		apiKey := fmt.Sprintf("cp-live-%s-%s", payload.Department[:3], hex.EncodeToString(randBytes))

		tenantID := fmt.Sprintf("tenant_%d", time.Now().UnixNano())
		newTenant := &finops.TenantRecord{
			ID:               tenantID,
			Name:             payload.Name,
			Department:       payload.Department,
			APIKey:           apiKey,
			MonthlyBudgetUSD: payload.MonthlyBudgetUSD,
			CreatedAt:        time.Now(),
			Active:           true,
		}

		if err := a.store.CreateTenant(newTenant); err != nil {
			http.Error(w, fmt.Sprintf("failed to create tenant: %v", err), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(newTenant)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// HandleCompare runs parallel model execution for side-by-side benchmarking.
func (a *APIHandler) HandleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req provider.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	if len(req.Messages) == 0 {
		http.Error(w, "messages cannot be empty", http.StatusBadRequest)
		return
	}

	result, err := a.engine.CompareModels(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("comparison failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// HandleCache inspects or clears Tier 0 cache.
func (a *APIHandler) HandleCache(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if a.promptCache == nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"enabled": false})
		return
	}

	if r.Method == http.MethodGet {
		stats := a.promptCache.Stats()
		_ = json.NewEncoder(w).Encode(stats)
		return
	}

	if r.Method == http.MethodPost {
		a.promptCache.Clear()
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "cleared"})
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// HandleGetThresholds returns the current active thresholds.
func (a *APIHandler) HandleGetThresholds(w http.ResponseWriter, r *http.Request) {
	theta, tau := a.engine.GetThresholds()
	resp := map[string]float64{
		"theta_bypass":   theta,
		"tau_confidence": tau,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleSetThresholds updates active dynamic thresholds.
func (a *APIHandler) HandleSetThresholds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		ThetaBypass   float64 `json:"theta_bypass"`
		TauConfidence float64 `json:"tau_confidence"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if payload.ThetaBypass <= 0 || payload.ThetaBypass > 1.0 || payload.TauConfidence <= 0 || payload.TauConfidence > 1.0 {
		http.Error(w, "thresholds must be between 0.0 and 1.0", http.StatusBadRequest)
		return
	}

	if err := a.engine.SetThresholds(payload.ThetaBypass, payload.TauConfidence); err != nil {
		http.Error(w, fmt.Sprintf("failed to save thresholds: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "updated",
		"theta_bypass":   payload.ThetaBypass,
		"tau_confidence": payload.TauConfidence,
	})
}

// HandleExportCSV streams a CSV file download.
func (a *APIHandler) HandleExportCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"cascadepilot_finops.csv\"")
	if err := a.store.ExportCSV(w); err != nil {
		http.Error(w, fmt.Sprintf("export error: %v", err), http.StatusInternalServerError)
	}
}

// HandleExportJSON streams a JSON file download.
func (a *APIHandler) HandleExportJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"cascadepilot_finops.json\"")
	if err := a.store.ExportJSON(w); err != nil {
		http.Error(w, fmt.Sprintf("export error: %v", err), http.StatusInternalServerError)
	}
}

// HandleSimulate executes a test query batch against the engine.
func (a *APIHandler) HandleSimulate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Count int `json:"count"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)
	if payload.Count <= 0 {
		payload.Count = 10
	}
	if payload.Count > 50 {
		payload.Count = 50
	}

	samplePrompts := []struct {
		Prompt string
		Dept   string
	}{
		{"Classify the sentiment of this review as positive, neutral, or negative: 'Great product, arrived quickly.'", "Customer Support"},
		{"Extract the vendor name and total invoice amount: Acme Corp, Invoice #4812, Total: $1,250.00.", "Finance"},
		{"What is our corporate policy on travel per diem for domestic flights?", "General"},
		{"Step by step, prove whether a two-phase commit protocol is vulnerable to coordinator crash during commit phase.", "Engineering"},
		{"Format this list of users into valid JSON with fields id, email, and active status.", "Engineering"},
		{"Draft a polite email asking for client feedback on our new analytics dashboard release.", "Marketing"},
	}

	ctx := r.Context()
	executed := 0
	for i := 0; i < payload.Count; i++ {
		item := samplePrompts[i%len(samplePrompts)]
		req := &provider.ChatCompletionRequest{
			Model: "cascadepilot-auto",
			Messages: []provider.Message{
				{Role: "user", Content: item.Prompt},
			},
		}
		_, _, err := a.engine.ExecuteWithTenant(ctx, req, fmt.Sprintf("tenant_%d", i%4), item.Dept)
		if err == nil {
			executed++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "simulated",
		"executed": executed,
	})
}
