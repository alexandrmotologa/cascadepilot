package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/alexandrmotologa/cascadepilot/internal/finops"
	"github.com/alexandrmotologa/cascadepilot/internal/provider"
	"github.com/alexandrmotologa/cascadepilot/internal/router"
)

// APIHandler handles FinOps reporting and administrative API routes.
type APIHandler struct {
	store  *finops.Store
	engine *router.Engine
}

// NewAPIHandler creates a new API controller.
func NewAPIHandler(store *finops.Store, engine *router.Engine) *APIHandler {
	return &APIHandler{
		store:  store,
		engine: engine,
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
		payload.Count = 5
	}
	if payload.Count > 50 {
		payload.Count = 50
	}

	samplePrompts := []string{
		"Classify the sentiment of this review as positive, neutral, or negative: 'Great product, arrived quickly.'",
		"Extract the vendor name and total invoice amount: Acme Corp, Invoice #4812, Total: $1,250.00.",
		"What is our corporate policy on travel per diem for domestic flights?",
		"Step by step, prove whether a two-phase commit protocol is vulnerable to coordinator crash during commit phase.",
		"Format this list of users into valid JSON with fields id, email, and active status.",
	}

	ctx := r.Context()
	executed := 0
	for i := 0; i < payload.Count; i++ {
		prompt := samplePrompts[i%len(samplePrompts)]
		req := &provider.ChatCompletionRequest{
			Model: "cascadepilot-auto",
			Messages: []provider.Message{
				{Role: "user", Content: prompt},
			},
		}
		_, _, err := a.engine.Execute(ctx, req)
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
