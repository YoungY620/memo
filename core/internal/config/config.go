// Package config provides configuration loading functionality
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the main configuration structure
type Config struct {
	Watcher WatcherConfig `yaml:"watcher"`
	Trigger TriggerConfig `yaml:"trigger"`
	Index   IndexConfig   `yaml:"index"`
}

// WatcherConfig file monitoring configuration
type WatcherConfig struct {
	Root       string   `yaml:"root"`       // Root directory to monitor
	Ignore     []string `yaml:"ignore"`     // Directories/files to ignore (glob patterns)
	Extensions []string `yaml:"extensions"` // File extensions to monitor
}

// TriggerConfig trigger management configuration
type TriggerConfig struct {
	MinFiles int `yaml:"minFiles"` // Minimum file change count threshold
	IdleMs   int `yaml:"idleMs"`   // Idle timeout in milliseconds
}

// IndexConfig index configuration
type IndexConfig struct {
	Path     string `yaml:"path"`     // Index output directory
	MaxNotes int    `yaml:"maxNotes"` // Maximum flash-notes count
	MaxTags  int    `yaml:"maxTags"`  // Maximum tag count
	MaxTypes int    `yaml:"maxTypes"` // Maximum types per _activities.json
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Watcher: WatcherConfig{
			Root: ".",
			Ignore: []string{
				"node_modules",
				".git",
				"dist",
				".kimi-index",
				"__pycache__",
				".venv",
				"venv",
				"target",
				".idea",
				".vscode",
			},
			Extensions: []string{
				".ts", ".js", ".tsx", ".jsx",
				".py",
				".go",
				".rs",
				".java",
				".c", ".cpp", ".h", ".hpp",
				".md",
				".yaml", ".yml",
				".json",
			},
		},
		Trigger: TriggerConfig{
			MinFiles: 5,
			IdleMs:   30000,
		},
		Index: IndexConfig{
			Path:     ".kimi-index",
			MaxNotes: 50,
			MaxTags:  100,
			MaxTypes: 100,
		},
	}
}

// Load loads configuration from specified path, uses default if file not found
func Load(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	// Use default path if not specified
	if configPath == "" {
		configPath = ".kimi-indexer.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file not found, use default
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Handle relative paths
	if !filepath.IsAbs(cfg.Watcher.Root) {
		cfg.Watcher.Root, _ = filepath.Abs(cfg.Watcher.Root)
	}
	if !filepath.IsAbs(cfg.Index.Path) {
		cfg.Index.Path = filepath.Join(cfg.Watcher.Root, cfg.Index.Path)
	}

	return cfg, nil
}

// Save saves configuration to specified path
func Save(cfg *Config, configPath string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}
