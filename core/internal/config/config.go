// Package config provides configuration loading functionality
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config 是项目的主配置结构
type Config struct {
	Watcher WatcherConfig `yaml:"watcher"`
	Trigger TriggerConfig `yaml:"trigger"`
	Index   IndexConfig   `yaml:"index"`
}

// WatcherConfig 文件监控配置
type WatcherConfig struct {
	Root       string   `yaml:"root"`       // 监控根目录
	Ignore     []string `yaml:"ignore"`     // 忽略的目录/文件 (glob 模式)
	Extensions []string `yaml:"extensions"` // 监控的文件扩展名
}

// TriggerConfig 触发管理配置
type TriggerConfig struct {
	MinFiles int `yaml:"minFiles"` // 变更文件数阈值
	IdleMs   int `yaml:"idleMs"`   // 空闲超时 (毫秒)
}

// IndexConfig 索引配置
type IndexConfig struct {
	Path     string `yaml:"path"`     // 索引输出目录
	MaxNotes int    `yaml:"maxNotes"` // flash-notes 最大条数
	MaxTags  int    `yaml:"maxTags"`  // tag 最大个数
	MaxTypes int    `yaml:"maxTypes"` // 每个 _activities.json 的 type 最大个数
}

// DefaultConfig 返回默认配置
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

// Load 从指定路径加载配置文件，如果文件不存在则使用默认配置
func Load(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	// 如果没有指定配置文件路径，使用默认路径
	if configPath == "" {
		configPath = ".kimi-indexer.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，使用默认配置
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// 处理相对路径
	if !filepath.IsAbs(cfg.Watcher.Root) {
		cfg.Watcher.Root, _ = filepath.Abs(cfg.Watcher.Root)
	}
	if !filepath.IsAbs(cfg.Index.Path) {
		cfg.Index.Path = filepath.Join(cfg.Watcher.Root, cfg.Index.Path)
	}

	return cfg, nil
}

// Save 将配置保存到指定路径
func Save(cfg *Config, configPath string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}
