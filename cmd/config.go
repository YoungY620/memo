package cmd

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Agent    AgentConfig `yaml:"agent"`
	Watch    WatchConfig `yaml:"watch"`
	LogLevel string      `yaml:"log_level"` // error, notice, info, debug
}

type AgentConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

type WatchConfig struct {
	IgnorePatterns []string `yaml:"ignore_patterns"`
	DebounceMs     int      `yaml:"debounce_ms"`
	MaxWaitMs      int      `yaml:"max_wait_ms"`
}

func LoadConfig(path string) (*Config, error) {
	cfg := &Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// Config file not found, use defaults
	} else {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// Apply defaults
	if cfg.Watch.DebounceMs == 0 {
		cfg.Watch.DebounceMs = 30000 // 30 seconds quiet period
	}
	if cfg.Watch.MaxWaitMs == 0 {
		cfg.Watch.MaxWaitMs = 300000 // 5 minutes max wait
	}
	if len(cfg.Watch.IgnorePatterns) == 0 {
		cfg.Watch.IgnorePatterns = []string{".git", "node_modules", ".memo", "*.log"}
	}
	return cfg, nil
}
