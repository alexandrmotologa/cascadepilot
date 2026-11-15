package finops

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStore_RecordAndSummary(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cascadepilot-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Initial empty summary
	sum, err := store.GetSummary()
	if err != nil {
		t.Fatalf("failed to get empty summary: %v", err)
	}
	if sum.TotalRequests != 0 {
		t.Errorf("expected 0 requests, got %d", sum.TotalRequests)
	}

	// Insert Tier 1 record
	rec1 := &RequestRecord{
		ID:               "req-1",
		Timestamp:        time.Now(),
		PromptPreview:    "Classify review",
		ComplexityScore:  0.15,
		RoutedTier:       "tier1",
		Tier1Model:       "gemini-1.5-flash",
		Tier2Model:       "claude-3-5-sonnet",
		Tier1Confidence:  0.95,
		Escalated:        false,
		PromptTokens:     50,
		CompletionTokens: 20,
		LatencyMs:        15,
		BaselineCostUSD:  0.00045,
		ActualCostUSD:    0.00001,
		SavedCostUSD:     0.00044,
		Stream:           false,
	}
	if err := store.RecordRequest(rec1); err != nil {
		t.Fatalf("failed to record request: %v", err)
	}

	// Insert Escalated record
	rec2 := &RequestRecord{
		ID:               "req-2",
		Timestamp:        time.Now(),
		PromptPreview:    "Complex proof",
		ComplexityScore:  0.85,
		RoutedTier:       "tier2_escalated",
		Tier1Model:       "gemini-1.5-flash",
		Tier2Model:       "claude-3-5-sonnet",
		Tier1Confidence:  0.45,
		Escalated:        true,
		EscalationReason: "confidence_below_threshold",
		PromptTokens:     100,
		CompletionTokens: 150,
		LatencyMs:        45,
		BaselineCostUSD:  0.00255,
		ActualCostUSD:    0.00257,
		SavedCostUSD:     -0.00002,
		Stream:           false,
	}
	if err := store.RecordRequest(rec2); err != nil {
		t.Fatalf("failed to record second request: %v", err)
	}

	// Check summary
	sum, err = store.GetSummary()
	if err != nil {
		t.Fatalf("failed to get summary: %v", err)
	}
	if sum.TotalRequests != 2 {
		t.Errorf("expected 2 requests, got %d", sum.TotalRequests)
	}
	if sum.Tier1Accepted != 1 {
		t.Errorf("expected 1 tier1 accepted, got %d", sum.Tier1Accepted)
	}
	if sum.Tier3Escalated != 1 {
		t.Errorf("expected 1 tier3 escalated, got %d", sum.Tier3Escalated)
	}
	if sum.Tier1SuccessRate != 50.0 {
		t.Errorf("expected 50%% success rate, got %.1f", sum.Tier1SuccessRate)
	}

	// Check threshold dynamic persistence
	if err := store.SetThresholds(0.80, 0.65); err != nil {
		t.Fatalf("failed to set thresholds: %v", err)
	}
	th, tc, found, err := store.GetThresholds()
	if err != nil || !found {
		t.Fatalf("failed to get thresholds: %v", err)
	}
	if th != 0.80 || tc != 0.65 {
		t.Errorf("expected thresholds 0.80/0.65, got %.2f/%.2f", th, tc)
	}

	// Check CSV export
	var buf bytes.Buffer
	if err := store.ExportCSV(&buf); err != nil {
		t.Fatalf("failed to export csv: %v", err)
	}
	if buf.Len() == 0 {
		t.Errorf("expected non-empty csv output")
	}

	// Check tenants listing and creation
	tenants, err := store.ListTenantsWithSpend()
	if err != nil {
		t.Fatalf("failed to list tenants: %v", err)
	}
	if len(tenants) == 0 {
		t.Errorf("expected seeded tenants, got 0")
	}

	customTenant := &TenantRecord{
		ID:               "tenant_test_ai",
		Name:             "AI Research Group",
		Department:       "Data Science",
		APIKey:           "cp-live-ds-alpha1",
		MonthlyBudgetUSD: 2000.0,
	}
	if err := store.CreateTenant(customTenant); err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	retrieved, found, err := store.GetTenantByKey("cp-live-ds-alpha1")
	if err != nil || !found {
		t.Fatalf("failed to find tenant by key: %v", err)
	}
	if retrieved.Name != "AI Research Group" {
		t.Errorf("expected tenant name 'AI Research Group', got '%s'", retrieved.Name)
	}
}
