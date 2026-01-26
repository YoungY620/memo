package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

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
		cfg.Watch.DebounceMs = 5000 // 5 seconds quiet period
	}
	if cfg.Watch.MaxWaitMs == 0 {
		cfg.Watch.MaxWaitMs = 300000 // 5 minutes max wait
	}
	if len(cfg.Watch.IgnorePatterns) == 0 {
		cfg.Watch.IgnorePatterns = []string{".git", "node_modules", ".memo", "*.log"}
	}
	return cfg, nil
}

// LoadGitignore parses a .gitignore file and returns the patterns.
// It handles comments, empty lines, and basic gitignore syntax.
func LoadGitignore(workDir string) ([]string, error) {
	gitignorePath := filepath.Join(workDir, ".gitignore")

	file, err := os.Open(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No .gitignore, return empty
		}
		return nil, err
	}
	defer file.Close()

	var patterns []string
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Skip negation patterns (we only care about ignoring)
		if strings.HasPrefix(line, "!") {
			continue
		}

		// Normalize the pattern
		pattern := normalizeGitignorePattern(line)
		if pattern != "" && !seen[pattern] {
			seen[pattern] = true
			patterns = append(patterns, pattern)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return patterns, nil
}

// normalizeGitignorePattern converts gitignore pattern to our ignore pattern format
func normalizeGitignorePattern(pattern string) string {
	// Remove leading slash (we treat all patterns as relative)
	pattern = strings.TrimPrefix(pattern, "/")

	// Remove trailing slash (directory indicator)
	pattern = strings.TrimSuffix(pattern, "/")

	// Skip empty after normalization
	if pattern == "" {
		return ""
	}

	return pattern
}

// MergeGitignore loads .gitignore from workDir and merges patterns into config.
// Patterns from .gitignore are added if not already present.
func (c *Config) MergeGitignore(workDir string) error {
	patterns, err := LoadGitignore(workDir)
	if err != nil {
		return err
	}

	if len(patterns) == 0 {
		return nil
	}

	// Build set of existing patterns for deduplication
	existing := make(map[string]bool)
	for _, p := range c.Watch.IgnorePatterns {
		existing[p] = true
	}

	// Add new patterns from .gitignore
	added := 0
	for _, p := range patterns {
		if !existing[p] {
			c.Watch.IgnorePatterns = append(c.Watch.IgnorePatterns, p)
			existing[p] = true
			added++
		}
	}

	if added > 0 {
		logDebug("Merged %d patterns from .gitignore", added)
	}

	return nil
}
