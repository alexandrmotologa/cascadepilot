package finops

import (
	"database/sql"
	"math"
	"time"
)

// FinOpsSummary contains aggregated metrics for executive dashboards.
type FinOpsSummary struct {
	TotalRequests        int64   `json:"total_requests"`
	Tier0CacheHits       int64   `json:"tier0_cache_hits"`
	Tier1Accepted        int64   `json:"tier1_accepted"`
	Tier2MidtierCount    int64   `json:"tier2_midtier_count"`
	Tier3Direct          int64   `json:"tier3_direct"`
	Tier3Escalated       int64   `json:"tier3_escalated"`
	Tier1SuccessRate     float64 `json:"tier1_success_rate"`
	TotalBaselineCostUSD float64 `json:"total_baseline_cost_usd"`
	TotalActualCostUSD   float64 `json:"total_actual_cost_usd"`
	TotalSavedUSD        float64 `json:"total_saved_usd"`
	SavingsPercentage    float64 `json:"savings_percentage"`
	AverageLatencyMs     float64 `json:"average_latency_ms"`
	QualityParityScore   float64 `json:"quality_parity_score"`
	DailySpendUSD        float64 `json:"daily_spend_usd"`
	MonthlySpendUSD      float64 `json:"monthly_spend_usd"`
	CircuitBreakerActive bool    `json:"circuit_breaker_active"`
}

// WaterfallMetrics tracks the flow across routing decision points.
type WaterfallMetrics struct {
	TotalQueries     int64 `json:"total_queries"`
	Tier0CacheHits   int64 `json:"tier0_cache_hits"`
	DirectTier3Count int64 `json:"direct_tier3_count"`
	SpeculativeTier1 int64 `json:"speculative_tier1"`
	Tier1Passed      int64 `json:"tier1_passed"`
	Tier2MidPassed   int64 `json:"tier2_mid_passed"`
	Tier3Escalated   int64 `json:"tier3_escalated"`
}

// LatencyMetrics compares latency distributions.
type LatencyMetrics struct {
	AvgCacheMs     float64 `json:"avg_cache_ms"`
	AvgTier1Ms     float64 `json:"avg_tier1_ms"`
	AvgTier2MidMs  float64 `json:"avg_tier2_mid_ms"`
	AvgDirectT3Ms  float64 `json:"avg_direct_t3_ms"`
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
		actual_cost_usd, saved_cost_usd, stream, tenant_id, department, cache_hit
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	escalatedInt := 0
	if r.Escalated {
		escalatedInt = 1
	}

	streamInt := 0
	if r.Stream {
		streamInt = 1
	}

	cacheInt := 0
	if r.CacheHit {
		cacheInt = 1
	}

	tenantID := r.TenantID
	if tenantID == "" {
		tenantID = "tenant_default"
	}

	department := r.Department
	if department == "" {
		department = "General"
	}

	_, err := s.db.Exec(query,
		r.ID, r.Timestamp.Format(time.RFC3339Nano), r.PromptPreview, r.ComplexityScore, r.RoutedTier,
		r.Tier1Model, r.Tier2Model, r.Tier1Confidence, escalatedInt, r.EscalationReason,
		r.PromptTokens, r.CompletionTokens, r.LatencyMs, r.BaselineCostUSD,
		r.ActualCostUSD, r.SavedCostUSD, streamInt, tenantID, department, cacheInt,
	)

	return err
}

// GetSummary computes global FinOps totals, rates, and current daily/monthly spend.
func (s *Store) GetSummary() (*FinOpsSummary, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	query := `
	SELECT 
		COUNT(*),
		COALESCE(SUM(CASE WHEN routed_tier = 'tier0_cache' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN routed_tier = 'tier1' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN routed_tier = 'tier2_mid' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN routed_tier = 'tier2_direct' OR routed_tier = 'tier3_direct' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN routed_tier = 'tier2_escalated' OR routed_tier = 'tier3_escalated' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(baseline_cost_usd), 0.0),
		COALESCE(SUM(actual_cost_usd), 0.0),
		COALESCE(SUM(saved_cost_usd), 0.0),
		COALESCE(AVG(latency_ms), 0.0)
	FROM requests
	`

	var total, t0Cache, t1Accepted, t2Mid, t3Direct, t3Escalated int64
	var baselineCost, actualCost, savedCost, avgLatency float64

	row := s.db.QueryRow(query)
	if err := row.Scan(&total, &t0Cache, &t1Accepted, &t2Mid, &t3Direct, &t3Escalated, &baselineCost, &actualCost, &savedCost, &avgLatency); err != nil {
		return nil, err
	}

	var savingsPct float64
	if baselineCost > 0 {
		savingsPct = (savedCost / baselineCost) * 100.0
	}

	var t1SuccessRate float64
	attemptedT1 := t1Accepted + t2Mid + t3Escalated
	if attemptedT1 > 0 {
		t1SuccessRate = (float64(t1Accepted) / float64(attemptedT1)) * 100.0
	}

	// Quality parity approximation: Cache (1.0) + Tier 1 (0.988) + Tier 2 Mid (0.994) + Frontier (1.0)
	var qualityParity float64
	if total > 0 {
		qualityParity = ((float64(t0Cache)*1.0 + float64(t1Accepted)*0.988 + float64(t2Mid)*0.994 + float64(t3Direct+t3Escalated)*1.0) / float64(total)) * 100.0
		qualityParity = math.Min(100.0, math.Max(0.0, qualityParity))
	} else {
		qualityParity = 100.0
	}

	// Calculate today and month spend
	todayStart := time.Now().UTC().Truncate(24 * time.Hour).Format(time.RFC3339)
	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)

	var dailySpend, monthlySpend float64
	_ = s.db.QueryRow("SELECT COALESCE(SUM(actual_cost_usd), 0.0) FROM requests WHERE timestamp >= ?", todayStart).Scan(&dailySpend)
	_ = s.db.QueryRow("SELECT COALESCE(SUM(actual_cost_usd), 0.0) FROM requests WHERE timestamp >= ?", monthStart).Scan(&monthlySpend)

	return &FinOpsSummary{
		TotalRequests:        total,
		Tier0CacheHits:       t0Cache,
		Tier1Accepted:        t1Accepted,
		Tier2MidtierCount:    t2Mid,
		Tier3Direct:          t3Direct,
		Tier3Escalated:       t3Escalated,
		Tier1SuccessRate:     math.Round(t1SuccessRate*10) / 10,
		TotalBaselineCostUSD: math.Round(baselineCost*10000) / 10000,
		TotalActualCostUSD:   math.Round(actualCost*10000) / 10000,
		TotalSavedUSD:        math.Round(savedCost*10000) / 10000,
		SavingsPercentage:    math.Round(savingsPct*10) / 10,
		AverageLatencyMs:     math.Round(avgLatency*10) / 10,
		QualityParityScore:   math.Round(qualityParity*10) / 10,
		DailySpendUSD:        math.Round(dailySpend*10000) / 10000,
		MonthlySpendUSD:      math.Round(monthlySpend*10000) / 10000,
		CircuitBreakerActive: false,
	}, nil
}

// GetWaterfall returns counts across the cascade stages.
func (s *Store) GetWaterfall() (*WaterfallMetrics, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	query := `
	SELECT 
		COUNT(*),
		COALESCE(SUM(CASE WHEN routed_tier = 'tier0_cache' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN routed_tier = 'tier2_direct' OR routed_tier = 'tier3_direct' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN routed_tier NOT IN ('tier0_cache', 'tier2_direct', 'tier3_direct') THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN routed_tier = 'tier1' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN routed_tier = 'tier2_mid' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN routed_tier = 'tier2_escalated' OR routed_tier = 'tier3_escalated' THEN 1 ELSE 0 END), 0)
	FROM requests
	`

	var total, t0Cache, directT3, specT1, t1Passed, t2MidPassed, t3Escalated int64
	row := s.db.QueryRow(query)
	if err := row.Scan(&total, &t0Cache, &directT3, &specT1, &t1Passed, &t2MidPassed, &t3Escalated); err != nil {
		return nil, err
	}

	return &WaterfallMetrics{
		TotalQueries:     total,
		Tier0CacheHits:   t0Cache,
		DirectTier3Count: directT3,
		SpeculativeTier1: specT1,
		Tier1Passed:      t1Passed,
		Tier2MidPassed:   t2MidPassed,
		Tier3Escalated:   t3Escalated,
	}, nil
}

// GetLatencyMetrics computes latency comparisons across tiers.
func (s *Store) GetLatencyMetrics() (*LatencyMetrics, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	query := `
	SELECT 
		COALESCE(AVG(CASE WHEN routed_tier = 'tier0_cache' THEN latency_ms END), 0.5),
		COALESCE(AVG(CASE WHEN routed_tier = 'tier1' THEN latency_ms END), 0.0),
		COALESCE(AVG(CASE WHEN routed_tier = 'tier2_mid' THEN latency_ms END), 0.0),
		COALESCE(AVG(CASE WHEN routed_tier IN ('tier2_direct', 'tier3_direct') THEN latency_ms END), 0.0),
		COALESCE(AVG(CASE WHEN routed_tier IN ('tier2_escalated', 'tier3_escalated') THEN latency_ms END), 0.0),
		COALESCE(AVG(latency_ms), 0.0)
	FROM requests
	`

	var cache, t1, t2Mid, directT3, escalated, overall float64
	row := s.db.QueryRow(query)
	if err := row.Scan(&cache, &t1, &t2Mid, &directT3, &escalated, &overall); err != nil {
		return nil, err
	}

	return &LatencyMetrics{
		AvgCacheMs:     math.Round(cache*10) / 10,
		AvgTier1Ms:     math.Round(t1*10) / 10,
		AvgTier2MidMs:  math.Round(t2Mid*10) / 10,
		AvgDirectT3Ms:  math.Round(directT3*10) / 10,
		AvgEscalatedMs: math.Round(escalated*10) / 10,
		AvgOverallMs:   math.Round(overall*10) / 10,
	}, nil
}

// ListTenantsWithSpend computes spend and budget utilization across all tenants.
func (s *Store) ListTenantsWithSpend() ([]TenantSummary, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	query := `
	SELECT 
		t.id, t.name, t.department, t.api_key, t.monthly_budget_usd, t.created_at, t.active,
		COUNT(r.id),
		COALESCE(SUM(r.prompt_tokens + r.completion_tokens), 0),
		COALESCE(SUM(r.actual_cost_usd), 0.0),
		COALESCE(SUM(r.saved_cost_usd), 0.0)
	FROM tenants t
	LEFT JOIN requests r ON t.id = r.tenant_id
	GROUP BY t.id
	ORDER BY t.department ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []TenantSummary
	for rows.Next() {
		var ts TenantSummary
		var tsStr string
		var activeInt int

		err := rows.Scan(
			&ts.ID, &ts.Name, &ts.Department, &ts.APIKey, &ts.MonthlyBudgetUSD, &tsStr, &activeInt,
			&ts.TotalRequests, &ts.TotalTokens, &ts.TotalActualCostUSD, &ts.TotalSavedCostUSD,
		)
		if err != nil {
			continue
		}

		ts.CreatedAt, _ = time.Parse(time.RFC3339Nano, tsStr)
		ts.Active = activeInt == 1

		if ts.MonthlyBudgetUSD > 0 {
			ts.BudgetUtilization = math.Round((ts.TotalActualCostUSD/ts.MonthlyBudgetUSD)*1000) / 10
		}

		ts.TotalActualCostUSD = math.Round(ts.TotalActualCostUSD*10000) / 10000
		ts.TotalSavedCostUSD = math.Round(ts.TotalSavedCostUSD*10000) / 10000

		summaries = append(summaries, ts)
	}

	return summaries, nil
}

// CreateTenant inserts a new virtual tenant and API key.
func (s *Store) CreateTenant(t *TenantRecord) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	query := `
	INSERT INTO tenants (id, name, department, api_key, monthly_budget_usd, created_at, active)
	VALUES (?, ?, ?, ?, ?, ?, 1)
	`
	_, err := s.db.Exec(query, t.ID, t.Name, t.Department, t.APIKey, t.MonthlyBudgetUSD, time.Now().Format(time.RFC3339Nano))
	return err
}

// GetTenantByKey retrieves a tenant by its API key.
func (s *Store) GetTenantByKey(apiKey string) (*TenantRecord, bool, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	query := `SELECT id, name, department, api_key, monthly_budget_usd, created_at, active FROM tenants WHERE api_key = ?`
	row := s.db.QueryRow(query, apiKey)

	var t TenantRecord
	var tsStr string
	var activeInt int

	err := row.Scan(&t.ID, &t.Name, &t.Department, &t.APIKey, &t.MonthlyBudgetUSD, &tsStr, &activeInt)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	t.CreatedAt, _ = time.Parse(time.RFC3339Nano, tsStr)
	t.Active = activeInt == 1
	return &t, true, nil
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
		actual_cost_usd, saved_cost_usd, stream, COALESCE(tenant_id, 'tenant_default'),
		COALESCE(department, 'General'), COALESCE(cache_hit, 0)
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
		var escalatedInt, streamInt, cacheInt int

		err := rows.Scan(
			&r.ID, &tsStr, &r.PromptPreview, &r.ComplexityScore, &r.RoutedTier,
			&r.Tier1Model, &r.Tier2Model, &r.Tier1Confidence, &escalatedInt, &r.EscalationReason,
			&r.PromptTokens, &r.CompletionTokens, &r.LatencyMs, &r.BaselineCostUSD,
			&r.ActualCostUSD, &r.SavedCostUSD, &streamInt, &r.TenantID, &r.Department, &cacheInt,
		)
		if err != nil {
			continue
		}

		r.Timestamp, _ = time.Parse(time.RFC3339Nano, tsStr)
		r.Escalated = escalatedInt == 1
		r.Stream = streamInt == 1
		r.CacheHit = cacheInt == 1
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

// GetGlobalSpendToday returns total USD spent since UTC midnight.
func (s *Store) GetGlobalSpendToday() (float64, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	today := time.Now().UTC().Truncate(24 * time.Hour).Format(time.RFC3339)
	var spend float64
	err := s.db.QueryRow("SELECT COALESCE(SUM(actual_cost_usd), 0.0) FROM requests WHERE timestamp >= ?", today).Scan(&spend)
	return spend, err
}

// GetGlobalSpendThisMonth returns total USD spent since the 1st of the current month.
func (s *Store) GetGlobalSpendThisMonth() (float64, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	var spend float64
	err := s.db.QueryRow("SELECT COALESCE(SUM(actual_cost_usd), 0.0) FROM requests WHERE timestamp >= ?", monthStart).Scan(&spend)
	return spend, err
}
