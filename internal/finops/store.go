package finops

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// RequestRecord captures a single completed query transaction.
type RequestRecord struct {
	ID               string    `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	PromptPreview    string    `json:"prompt_preview"`
	ComplexityScore  float64   `json:"complexity_score"`
	RoutedTier       string    `json:"routed_tier"` // "tier0_cache", "tier1", "tier2_mid", "tier3_direct", "tier3_escalated"
	Tier1Model       string    `json:"tier1_model"`
	Tier2Model       string    `json:"tier2_model"`
	Tier1Confidence  float64   `json:"tier1_confidence"`
	Escalated        bool      `json:"escalated"`
	EscalationReason string    `json:"escalation_reason,omitempty"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	LatencyMs        int64     `json:"latency_ms"`
	BaselineCostUSD  float64   `json:"baseline_cost_usd"`
	ActualCostUSD    float64   `json:"actual_cost_usd"`
	SavedCostUSD     float64   `json:"saved_cost_usd"`
	Stream           bool      `json:"stream"`
	TenantID         string    `json:"tenant_id"`
	Department       string    `json:"department"`
	CacheHit         bool      `json:"cache_hit"`
}

// TenantRecord represents a multi-tenant client organization or department.
type TenantRecord struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Department       string    `json:"department"`
	APIKey           string    `json:"api_key"`
	MonthlyBudgetUSD float64   `json:"monthly_budget_usd"`
	CreatedAt        time.Time `json:"created_at"`
	Active           bool      `json:"active"`
}

// TenantSummary contains department-level FinOps chargeback metrics.
type TenantSummary struct {
	TenantRecord
	TotalRequests      int64   `json:"total_requests"`
	TotalTokens        int64   `json:"total_tokens"`
	TotalActualCostUSD float64 `json:"total_actual_cost_usd"`
	TotalSavedCostUSD  float64 `json:"total_saved_cost_usd"`
	BudgetUtilization  float64 `json:"budget_utilization_pct"`
}

// Store manages SQLite persistence for FinOps transactions.
type Store struct {
	db   *sql.DB
	lock sync.RWMutex
}

// NewStore initializes the SQLite database with WAL mode and schema tables.
func NewStore(dbPath string) (*Store, error) {
	if dbPath == "" {
		dbPath = "data/cascadepilot.db"
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return s, nil
}

func (s *Store) initSchema() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	query := `
	CREATE TABLE IF NOT EXISTS requests (
		id TEXT PRIMARY KEY,
		timestamp DATETIME NOT NULL,
		prompt_preview TEXT,
		complexity_score REAL,
		routed_tier TEXT,
		tier1_model TEXT,
		tier2_model TEXT,
		tier1_confidence REAL,
		escalated INTEGER,
		escalation_reason TEXT,
		prompt_tokens INTEGER,
		completion_tokens INTEGER,
		latency_ms INTEGER,
		baseline_cost_usd REAL,
		actual_cost_usd REAL,
		saved_cost_usd REAL,
		stream INTEGER,
		tenant_id TEXT DEFAULT 'tenant_default',
		department TEXT DEFAULT 'General',
		cache_hit INTEGER DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_requests_timestamp ON requests(timestamp);
	CREATE INDEX IF NOT EXISTS idx_requests_routed_tier ON requests(routed_tier);

	CREATE TABLE IF NOT EXISTS config_thresholds (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		theta_bypass REAL NOT NULL,
		tau_confidence REAL NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS tenants (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		department TEXT NOT NULL,
		api_key TEXT UNIQUE NOT NULL,
		monthly_budget_usd REAL NOT NULL,
		created_at DATETIME NOT NULL,
		active INTEGER NOT NULL
	);
	`

	if _, err := s.db.Exec(query); err != nil {
		return err
	}

	// Schema migrations for backward compatibility with existing databases
	_, _ = s.db.Exec("ALTER TABLE requests ADD COLUMN tenant_id TEXT DEFAULT 'tenant_default'")
	_, _ = s.db.Exec("ALTER TABLE requests ADD COLUMN department TEXT DEFAULT 'General'")
	_, _ = s.db.Exec("ALTER TABLE requests ADD COLUMN cache_hit INTEGER DEFAULT 0")

	// Create index on department after column is guaranteed to exist
	_, _ = s.db.Exec("CREATE INDEX IF NOT EXISTS idx_requests_department ON requests(department)")

	// Seed default tenants if empty
	var count int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM tenants").Scan(&count)
	if count == 0 {
		nowStr := time.Now().Format(time.RFC3339Nano)
		seedQuery := `
		INSERT INTO tenants (id, name, department, api_key, monthly_budget_usd, created_at, active) VALUES
		('tenant_eng', 'Core Engineering', 'Engineering', 'cp-live-eng-9f821a', 1500.0, ?, 1),
		('tenant_support', 'Customer Care & Helpdesk', 'Customer Support', 'cp-live-sup-3b714c', 500.0, ?, 1),
		('tenant_mkt', 'Growth & Marketing Ops', 'Marketing', 'cp-live-mkt-5c920e', 300.0, ?, 1),
		('tenant_default', 'General Enterprise Workloads', 'General', 'cp-live-gen-default', 1000.0, ?, 1);
		`
		_, _ = s.db.Exec(seedQuery, nowStr, nowStr, nowStr, nowStr)
	}

	return nil
}

// Close closes the underlying database handle.
func (s *Store) Close() error {
	return s.db.Close()
}
