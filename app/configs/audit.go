package config

import (
	"encoding/json"
	"os"
)

// DefaultConfig returns a normalized copy of the built-in default config.
func DefaultConfig() Config {
	cfg := defaultConfig()
	applyDefaults(&cfg)
	return cfg
}

// NormalizeConfig applies defaults and sanitization to a config copy.
func NormalizeConfig(cfg Config) Config {
	normalized := cfg
	applyDefaults(&normalized)
	return normalized
}

// LoadConfigFile reads and normalizes a config file without mutating it on disk.
func LoadConfigFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	applyDefaults(&cfg)
	return cfg, nil
}
