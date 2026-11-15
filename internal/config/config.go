package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DefaultConfig returns safe production defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:           "0.0.0.0",
			Port:           8088,
			ReadTimeoutMs:  60000,
			WriteTimeoutMs: 60000,
		},
		Thresholds: ThresholdConfig{
			ThetaBypass:        0.75,
			TauConfidence:      0.70,
			StreamBufferTokens: 16,
			AutoTuneEnabled:    true,
		},
		Tiers: TiersConfig{
			Tier1: TierConfig{
				Model:                "gemini-1.5-flash",
				Provider:             "mock",
				Endpoint:             "https://generativelanguage.googleapis.com/v1beta/openai",
				APIKeyEnv:            "GEMINI_API_KEY",
				CostPerMillionInput:  0.075,
				CostPerMillionOutput: 0.30,
				TimeoutMs:            8000,
			},
			Tier2: TierConfig{
				Model:                "gpt-4o-mini",
				Provider:             "mock",
				Endpoint:             "https://api.openai.com/v1",
				APIKeyEnv:            "OPENAI_API_KEY",
				CostPerMillionInput:  0.150,
				CostPerMillionOutput: 0.60,
				TimeoutMs:            15000,
			},
			Tier3: TierConfig{
				Model:                "claude-3-5-sonnet",
				Provider:             "mock",
				Endpoint:             "https://api.anthropic.com/v1",
				APIKeyEnv:            "ANTHROPIC_API_KEY",
				CostPerMillionInput:  3.00,
				CostPerMillionOutput: 15.00,
				TimeoutMs:            35000,
			},
		},
		Cache: CacheConfig{
			Enabled:    true,
			MaxSize:    5000,
			TTLMinutes: 1440, // 24 hours
		},
		Budget: BudgetConfig{
			DailyCapUSD:          100.0,
			MonthlyCapUSD:        2500.0,
			CircuitBreakerAction: "downgrade_to_tier1",
		},
		Storage: StorageConfig{
			DBPath: "data/cascadepilot.db",
		},
		Simulation: SimulationConfig{
			MockEnabled:               true,
			SimulatedNetworkLatencyMs: 12,
		},
	}
}

// LoadFromFile reads and parses a YAML configuration file.
func LoadFromFile(filePath string) (*Config, error) {
	cfg := DefaultConfig()

	if filePath == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml config: %w", err)
	}

	// Validate thresholds
	if cfg.Thresholds.ThetaBypass <= 0 || cfg.Thresholds.ThetaBypass > 1.0 {
		cfg.Thresholds.ThetaBypass = 0.75
	}
	if cfg.Thresholds.TauConfidence <= 0 || cfg.Thresholds.TauConfidence > 1.0 {
		cfg.Thresholds.TauConfidence = 0.70
	}
	if cfg.Thresholds.StreamBufferTokens <= 0 {
		cfg.Thresholds.StreamBufferTokens = 16
	}

	// Fallback Tier3 if legacy config only had Tier2 as frontier
	if cfg.Tiers.Tier3.Model == "" && cfg.Tiers.Tier2.Model != "" {
		cfg.Tiers.Tier3 = cfg.Tiers.Tier2
		cfg.Tiers.Tier2 = TierConfig{
			Model:                "gpt-4o-mini",
			Provider:             "mock",
			Endpoint:             "https://api.openai.com/v1",
			APIKeyEnv:            "OPENAI_API_KEY",
			CostPerMillionInput:  0.150,
			CostPerMillionOutput: 0.60,
			TimeoutMs:            15000,
		}
	}

	return cfg, nil
}
