package finops

import (
	"database/sql"
	"math"
	"time"
)

// FinOpsSummary contains aggregated metrics for executive dashboards.
type FinOpsSummary struct {
	TotalRequests        int64   `json:"total_requests"`
	Tier1Accepted        int64   `json:"tier1_accepted"`
	Tier2Direct          int64   `json:"tier2_direct"`
	Tier2Escalated       int64   `json:"tier2_escalated"`
	Tier1SuccessRate     float64 `json:"tier1_success_rate"`
	TotalTokensSaved     int64   `json:"total_tokens_saved"`
	TotalBaselineCostUSD float64 `json:"total_baseline_cost_usd"`
	TotalActualCostUSD   float64 `json:"total_actual_cost_usd"`
	TotalSavedUSD        float64 `json:"total_saved_usd"`
	SavingsPercentage    float64 `json:"savings_percentage"`
	AverageLatencyMs     float64 `json:"average_latency_ms"`
	QualityParityScore   float64 `json:"quality_parity_score"`
}

// WaterfallMetrics tracks the flow across routing decision points.
type WaterfallMetrics struct {
	TotalQueries     int64 `json:"total_queries"`
	DirectTier2Count int64 `json:"direct_tier2_count"`
	SpeculativeTier1 int64 `json:"speculative_tier1"`
	Tier1Passed      int64 `json:"tier1_passed"`
	Tier1Escalated   int64 `json:"tier1_escalated"`
}

// LatencyMetrics compares latency distributions.
type LatencyMetrics struct {
	AvgTier1Ms     float64 `json:"avg_tier1_ms"`
	AvgDirectT2Ms  float64 `json:"avg_direct_t2_ms"`
	AvgEscalatedMs float64 `json:"avg_escalated_ms"`
	AvgOverallMs   float64 `json:"avg_overall_ms"`
}

// RecordRequest inserts a new query transaction record.
func (s *Store) RecordRequest(r *RequestRecord) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	query := `
	INSERT INTO requests (
		id, timestamp, prompt_preview, complexity_score, routed_tier,
		tier1_model, tier2_model, tier1_confidence, escalated, escalation_reason,
		prompt_tokens, completion_tokens, latency_ms, baseline_cost_usd,
		actual_cost_usd, saved_cost_usd, stream
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	escalatedInt := 0
	if r.Escalated {
		escalatedInt = 1
	}

	streamInt := 0
	if r.Stream {
		streamInt = 1
	}

	_, err := s.db.Exec(query,
		r.ID, r.Timestamp.Format(time.RFC3339Nano), r.PromptPreview, r.ComplexityScore, r.RoutedTier,
		r.Tier1Model, r.Tier2Model, r.Tier1Confidence, escalatedInt, r.EscalationReason,
		r.PromptTokens, r.CompletionTokens, r.LatencyMs, r.BaselineCostUSD,
		r.ActualCostUSD, r.SavedCostUSD, streamInt,
	)

	return err
}

// GetSummary computes global FinOps totals and rates.
func (s *Store) GetSummary() (*FinOpsSummary, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	query := `
	SELECT 
		COUNT(*),
		COALESCE(SUM(CASE WHEN routed_tier = 'tier1' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN routed_tier = 'tier2_direct' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN routed_tier = 'tier2_escalated' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(baseline_cost_usd), 0.0),
		COALESCE(SUM(actual_cost_usd), 0.0),
		COALESCE(SUM(saved_cost_usd), 0.0),
		COALESCE(AVG(latency_ms), 0.0)
	FROM requests
	`

	var total, t1Accepted, t2Direct, t2Escalated int64
	var baselineCost, actualCost, savedCost, avgLatency float64

	row := s.db.QueryRow(query)
	if err := row.Scan(&total, &t1Accepted, &t2Direct, &t2Escalated, &baselineCost, &actualCost, &savedCost, &avgLatency); err != nil {
		return nil, err
	}

	var savingsPct float64
	if baselineCost > 0 {
		savingsPct = (savedCost / baselineCost) * 100.0
	}

	var t1SuccessRate float64
	attemptedT1 := t1Accepted + t2Escalated
	if attemptedT1 > 0 {
		t1SuccessRate = (float64(t1Accepted) / float64(attemptedT1)) * 100.0
	}

	// Quality parity approximation: Tier 1 accuracy (~0.988) + Tier 2 (1.0)
	var qualityParity float64
	if total > 0 {
		qualityParity = ((float64(t1Accepted) * 0.988) + float64(t2Direct+t2Escalated)*1.0) / float64(total) * 100.0
		qualityParity = math.Min(100.0, math.Max(0.0, qualityParity))
	} else {
		qualityParity = 100.0
	}

	return &FinOpsSummary{
		TotalRequests:        total,
		Tier1Accepted:        t1Accepted,
		Tier2Direct:          t2Direct,
		Tier2Escalated:       t2Escalated,
		Tier1SuccessRate:     math.Round(t1SuccessRate*10) / 10,
		TotalTokensSaved:     0, // In token pricing, savedCost reflects savings
		TotalBaselineCostUSD: math.Round(baselineCost*10000) / 10000,
		TotalActualCostUSD:   math.Round(actualCost*10000) / 10000,
		TotalSavedUSD:        math.Round(savedCost*10000) / 10000,
		SavingsPercentage:    math.Round(savingsPct*10) / 10,
		AverageLatencyMs:     math.Round(avgLatency*10) / 10,
		QualityParityScore:   math.Round(qualityParity*10) / 10,
	}, nil
}

// GetWaterfall returns counts across the cascade stages.
func (s *Store) GetWaterfall() (*WaterfallMetrics, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	query := `
	SELECT 
		COUNT(*),
		COALESCE(SUM(CASE WHEN routed_tier = 'tier2_direct' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN routed_tier != 'tier2_direct' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN routed_tier = 'tier1' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN routed_tier = 'tier2_escalated' THEN 1 ELSE 0 END), 0)
	FROM requests
	`

	var total, directT2, specT1, t1Passed, t1Escalated int64
	row := s.db.QueryRow(query)
	if err := row.Scan(&total, &directT2, &specT1, &t1Passed, &t1Escalated); err != nil {
		return nil, err
	}

	return &WaterfallMetrics{
		TotalQueries:     total,
		DirectTier2Count: directT2,
		SpeculativeTier1: specT1,
		Tier1Passed:      t1Passed,
		Tier1Escalated:   t1Escalated,
	}, nil
}

// GetLatencyMetrics computes latency comparisons across tiers.
func (s *Store) GetLatencyMetrics() (*LatencyMetrics, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	query := `
	SELECT 
		COALESCE(AVG(CASE WHEN routed_tier = 'tier1' THEN latency_ms END), 0.0),
		COALESCE(AVG(CASE WHEN routed_tier = 'tier2_direct' THEN latency_ms END), 0.0),
		COALESCE(AVG(CASE WHEN routed_tier = 'tier2_escalated' THEN latency_ms END), 0.0),
		COALESCE(AVG(latency_ms), 0.0)
	FROM requests
	`

	var t1, directT2, escalated, overall float64
	row := s.db.QueryRow(query)
	if err := row.Scan(&t1, &directT2, &escalated, &overall); err != nil {
		return nil, err
	}

	return &LatencyMetrics{
		AvgTier1Ms:     math.Round(t1*10) / 10,
		AvgDirectT2Ms:  math.Round(directT2*10) / 10,
		AvgEscalatedMs: math.Round(escalated*10) / 10,
		AvgOverallMs:   math.Round(overall*10) / 10,
	}, nil
}

// GetRecentRequests returns the latest N records.
func (s *Store) GetRecentRequests(limit int) ([]RequestRecord, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	query := `
	SELECT 
		id, timestamp, prompt_preview, complexity_score, routed_tier,
		tier1_model, tier2_model, tier1_confidence, escalated, COALESCE(escalation_reason, ''),
		prompt_tokens, completion_tokens, latency_ms, baseline_cost_usd,
		actual_cost_usd, saved_cost_usd, stream
	FROM requests
	ORDER BY timestamp DESC
	LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []RequestRecord
	for rows.Next() {
		var r RequestRecord
		var tsStr string
		var escalatedInt, streamInt int

		err := rows.Scan(
			&r.ID, &tsStr, &r.PromptPreview, &r.ComplexityScore, &r.RoutedTier,
			&r.Tier1Model, &r.Tier2Model, &r.Tier1Confidence, &escalatedInt, &r.EscalationReason,
			&r.PromptTokens, &r.CompletionTokens, &r.LatencyMs, &r.BaselineCostUSD,
			&r.ActualCostUSD, &r.SavedCostUSD, &streamInt,
		)
		if err != nil {
			continue
		}

		r.Timestamp, _ = time.Parse(time.RFC3339Nano, tsStr)
		r.Escalated = escalatedInt == 1
		r.Stream = streamInt == 1
		records = append(records, r)
	}

	return records, nil
}

// GetThresholds retrieves dynamically configured thresholds or returns false if unset.
func (s *Store) GetThresholds() (thetaBypass, tauConfidence float64, found bool, err error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	query := `SELECT theta_bypass, tau_confidence FROM config_thresholds WHERE id = 1`
	row := s.db.QueryRow(query)
	err = row.Scan(&thetaBypass, &tauConfidence)
	if err == sql.ErrNoRows {
		return 0, 0, false, nil
	}
	if err != nil {
		return 0, 0, false, err
	}
	return thetaBypass, tauConfidence, true, nil
}

// SetThresholds persists dynamic thresholds into the database.
func (s *Store) SetThresholds(thetaBypass, tauConfidence float64) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	query := `
	INSERT INTO config_thresholds (id, theta_bypass, tau_confidence, updated_at)
	VALUES (1, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		theta_bypass = excluded.theta_bypass,
		tau_confidence = excluded.tau_confidence,
		updated_at = excluded.updated_at
	`

	_, err := s.db.Exec(query, thetaBypass, tauConfidence, time.Now().Format(time.RFC3339Nano))
	return err
}
