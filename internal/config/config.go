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
		},
		Tiers: TiersConfig{
			Tier1: TierConfig{
				Model:               "gemini-1.5-flash",
				Provider:            "mock",
				Endpoint:            "https://generativelanguage.googleapis.com/v1beta/openai",
				APIKeyEnv:           "GEMINI_API_KEY",
				CostPerMillionInput: 0.075,
				CostPerMillionOutput: 0.30,
				TimeoutMs:           8000,
			},
			Tier2: TierConfig{
				Model:               "claude-3-5-sonnet",
				Provider:            "mock",
				Endpoint:            "https://api.anthropic.com/v1",
				APIKeyEnv:           "ANTHROPIC_API_KEY",
				CostPerMillionInput: 3.00,
				CostPerMillionOutput: 15.00,
				TimeoutMs:           35000,
			},
		},
		Storage: StorageConfig{
			DBPath: "data/cascadepilot.db",
		},
		Simulation: SimulationConfig{
			MockEnabled:                true,
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

	return cfg, nil
}
