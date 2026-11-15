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
	RoutedTier       string    `json:"routed_tier"` // "tier1", "tier2_direct", "tier2_escalated"
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
		stream INTEGER
	);

	CREATE INDEX IF NOT EXISTS idx_requests_timestamp ON requests(timestamp);
	CREATE INDEX IF NOT EXISTS idx_requests_routed_tier ON requests(routed_tier);

	CREATE TABLE IF NOT EXISTS config_thresholds (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		theta_bypass REAL NOT NULL,
		tau_confidence REAL NOT NULL,
		updated_at DATETIME NOT NULL
	);
	`

	_, err := s.db.Exec(query)
	return err
}

// Close closes the underlying database handle.
func (s *Store) Close() error {
	return s.db.Close()
}
