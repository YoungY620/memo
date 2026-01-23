package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Moonshot AgentConfig `yaml:"moonshot"`
	Watch    WatchConfig    `yaml:"watch"`
}

type AgentConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

type WatchConfig struct {
	IgnorePatterns []string `yaml:"ignore_patterns"`
	DebounceMs     int      `yaml:"debounce_ms"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	// defaults
	if cfg.Watch.DebounceMs == 0 {
		cfg.Watch.DebounceMs = 1000
	}
	if len(cfg.Watch.IgnorePatterns) == 0 {
		cfg.Watch.IgnorePatterns = []string{".git", "node_modules", ".baecon", "*.log"}
	}
	return &cfg, nil
}
