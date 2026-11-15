package config

// Config represents the root configuration for cascadepilot.
type Config struct {
	Server      ServerConfig     `yaml:"server"`
	Thresholds  ThresholdConfig  `yaml:"thresholds"`
	Tiers       TiersConfig      `yaml:"tiers"`
	Cache       CacheConfig      `yaml:"cache"`
	Budget      BudgetConfig     `yaml:"budget"`
	Storage     StorageConfig    `yaml:"storage"`
	Simulation  SimulationConfig `yaml:"simulation"`
}

// ServerConfig defines HTTP server binding parameters.
type ServerConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	ReadTimeoutMs  int    `yaml:"read_timeout_ms"`
	WriteTimeoutMs int    `yaml:"write_timeout_ms"`
}

// ThresholdConfig controls routing decisions and verification strictness.
type ThresholdConfig struct {
	ThetaBypass        float64 `yaml:"theta_bypass"`
	TauConfidence      float64 `yaml:"tau_confidence"`
	StreamBufferTokens int     `yaml:"stream_buffer_tokens"`
	AutoTuneEnabled    bool    `yaml:"auto_tune_enabled"`
}

// TiersConfig contains Tier 1, Tier 2 (Mid-tier), and Tier 3 (Frontier) models.
type TiersConfig struct {
	Tier1 TierConfig `yaml:"tier1"`
	Tier2 TierConfig `yaml:"tier2"` // Mid-tier reasoning
	Tier3 TierConfig `yaml:"tier3"` // Flagship frontier
}

// TierConfig defines parameters for an upstream model tier.
type TierConfig struct {
	Model                string  `yaml:"model"`
	Provider             string  `yaml:"provider"`
	Endpoint             string  `yaml:"endpoint"`
	APIKeyEnv            string  `yaml:"api_key_env"`
	CostPerMillionInput  float64 `yaml:"cost_per_million_input"`
	CostPerMillionOutput float64 `yaml:"cost_per_million_output"`
	TimeoutMs            int     `yaml:"timeout_ms"`
}

// CacheConfig configures Tier 0 exact and normalized prompt caching.
type CacheConfig struct {
	Enabled    bool `yaml:"enabled"`
	MaxSize    int  `yaml:"max_size"`
	TTLMinutes int  `yaml:"ttl_minutes"`
}

// BudgetConfig defines spending caps and circuit breaker behavior.
type BudgetConfig struct {
	DailyCapUSD          float64 `yaml:"daily_cap_usd"`
	MonthlyCapUSD        float64 `yaml:"monthly_cap_usd"`
	CircuitBreakerAction string  `yaml:"circuit_breaker_action"` // "downgrade_to_tier1" or "block"
}

// StorageConfig configures persistence for telemetry records.
type StorageConfig struct {
	DBPath string `yaml:"db_path"`
}

// SimulationConfig controls local synthetic testing behavior.
type SimulationConfig struct {
	MockEnabled               bool `yaml:"mock_enabled"`
	SimulatedNetworkLatencyMs int  `yaml:"simulated_network_latency_ms"`
}
