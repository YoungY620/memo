package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/user/kimi-sdk-agent-indexer/core/watcher"
)

// IndexConfig describes index specific options.
type IndexConfig struct {
	Path string `yaml:"path"`
}

// Config holds runtime configuration for the indexer.
type Config struct {
	Watcher   watcher.Config `yaml:"watcher"`
	Index     IndexConfig    `yaml:"index"`
	SchemaDir string         `yaml:"schemaDir"`

	source string
}

// Default returns a baseline configuration.
func Default() Config {
	return Config{
		Watcher: watcher.Config{
			Root:        ".",
			IgnoreGlobs: []string{".git", "node_modules", ".kimi-indexer", ".kimi-indexer/**"},
			Extensions:  nil,
		},
		Index: IndexConfig{
			Path: ".kimi-indexer",
		},
		SchemaDir: "schemas",
	}
}

// Load reads configuration from a YAML file. Missing files fall back to defaults.
func Load(path string) (*Config, error) {
	cfg := Default()

	if path == "" {
		return &cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg.source = path
			return &cfg, nil
		}
		return nil, fmt.Errorf("config: read %q: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %q: %w", path, err)
	}
	cfg.source = path
	return &cfg, nil
}

// ApplyOverrides mutates the configuration with values supplied via CLI flags.
func (c *Config) ApplyOverrides(root, index, schema string) {
	if root != "" {
		c.Watcher.Root = root
	}
	if index != "" {
		c.Index.Path = index
	}
	if schema != "" {
		c.SchemaDir = schema
	}
}

// Normalize resolves relative paths to absolute ones.
func (c *Config) Normalize() error {
	base := filepath.Dir(c.source)
	if base == "." || base == "" {
		if wd, err := os.Getwd(); err == nil {
			base = wd
		}
	}

	var err error
	if c.Watcher.Root == "" {
		c.Watcher.Root = "."
	}
	if !filepath.IsAbs(c.Watcher.Root) {
		if base == "" {
			c.Watcher.Root, err = filepath.Abs(c.Watcher.Root)
		} else {
			c.Watcher.Root, err = filepath.Abs(filepath.Join(base, c.Watcher.Root))
		}
		if err != nil {
			return fmt.Errorf("config: resolve watcher.root: %w", err)
		}
	}

	if c.Index.Path == "" {
		c.Index.Path = ".kimi-indexer"
	}
	if !filepath.IsAbs(c.Index.Path) {
		c.Index.Path = filepath.Join(c.Watcher.Root, c.Index.Path)
	}

	if c.SchemaDir == "" {
		c.SchemaDir = "schemas"
	}
	if !filepath.IsAbs(c.SchemaDir) {
		if base == "" {
			c.SchemaDir, err = filepath.Abs(c.SchemaDir)
		} else {
			c.SchemaDir, err = filepath.Abs(filepath.Join(base, c.SchemaDir))
		}
		if err != nil {
			return fmt.Errorf("config: resolve schemaDir: %w", err)
		}
	}

	return nil
}

// Validate performs simple sanity checks on the configuration.
func (c *Config) Validate() error {
	if c.Watcher.Root == "" {
		return errors.New("config: watcher.root required")
	}
	if c.Index.Path == "" {
		return errors.New("config: index.path required")
	}
	if c.SchemaDir == "" {
		return errors.New("config: schemaDir required")
	}
	return nil
}

// PrettyYAML renders the configuration as YAML for diagnostics.
func (c Config) PrettyYAML() string {
	out, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("%+v", c)
	}
	return string(out)
}
