package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Test with non-existent config file (should use defaults)
	cfg, err := LoadConfig("nonexistent.yaml")
	require.NoError(t, err)

	// Verify default values
	assert.Equal(t, 30000, cfg.Watch.DebounceMs, "Default debounce should be 30000ms")
	assert.Equal(t, 300000, cfg.Watch.MaxWaitMs, "Default max wait should be 300000ms")
	assert.Contains(t, cfg.Watch.IgnorePatterns, ".git", "Default ignore should include .git")
	assert.Contains(t, cfg.Watch.IgnorePatterns, "node_modules", "Default ignore should include node_modules")
	assert.Contains(t, cfg.Watch.IgnorePatterns, ".memo", "Default ignore should include .memo")
}

func TestLoadConfig_FileNotExist(t *testing.T) {
	cfg, err := LoadConfig("/path/to/nonexistent/config.yaml")
	require.NoError(t, err, "Should not error for non-existent config file")
	assert.NotNil(t, cfg)
	// Should have defaults
	assert.Equal(t, 30000, cfg.Watch.DebounceMs)
}

func TestLoadConfig_ParseYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `
agent:
  api_key: test-api-key
  model: test-model
watch:
  ignore_patterns:
    - .git
    - custom_ignore
  debounce_ms: 5000
  max_wait_ms: 60000
log_level: debug
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	assert.Equal(t, "test-api-key", cfg.Agent.APIKey)
	assert.Equal(t, "test-model", cfg.Agent.Model)
	assert.Equal(t, 5000, cfg.Watch.DebounceMs)
	assert.Equal(t, 60000, cfg.Watch.MaxWaitMs)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Contains(t, cfg.Watch.IgnorePatterns, "custom_ignore")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Invalid YAML content
	err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0644)
	require.NoError(t, err)

	_, err = LoadConfig(configPath)
	assert.Error(t, err, "Should error on invalid YAML")
}

func TestLoadConfig_PartialConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Only set some values, others should use defaults
	content := `
watch:
  debounce_ms: 10000
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	assert.Equal(t, 10000, cfg.Watch.DebounceMs, "Should use config value")
	assert.Equal(t, 300000, cfg.Watch.MaxWaitMs, "Should use default for unset value")
}

func TestLoadGitignore_Patterns(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "basic patterns",
			content: `node_modules
dist
*.log`,
			expected: []string{"node_modules", "dist", "*.log"},
		},
		{
			name: "with comments",
			content: `# This is a comment
node_modules
# Another comment
dist`,
			expected: []string{"node_modules", "dist"},
		},
		{
			name: "with empty lines",
			content: `node_modules

dist

`,
			expected: []string{"node_modules", "dist"},
		},
		{
			name: "with negation (ignored)",
			content: `node_modules
!keep_this
dist`,
			expected: []string{"node_modules", "dist"},
		},
		{
			name: "with leading slash",
			content: `/root_only
/another_root`,
			expected: []string{"root_only", "another_root"},
		},
		{
			name: "with trailing slash",
			content: `node_modules/
build/`,
			expected: []string{"node_modules", "build"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subDir := filepath.Join(tmpDir, tt.name)
			require.NoError(t, os.MkdirAll(subDir, 0755))

			gitignorePath := filepath.Join(subDir, ".gitignore")
			require.NoError(t, os.WriteFile(gitignorePath, []byte(tt.content), 0644))

			patterns, err := LoadGitignore(subDir)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, patterns)
		})
	}
}

func TestLoadGitignore_NotExist(t *testing.T) {
	tmpDir := t.TempDir()
	patterns, err := LoadGitignore(tmpDir)
	require.NoError(t, err, "Should not error for non-existent .gitignore")
	assert.Nil(t, patterns)
}

func TestNormalizeGitignorePattern(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"node_modules", "node_modules"},
		{"/root_only", "root_only"},
		{"trailing/", "trailing"},
		{"/both/", "both"},
		{"", ""},
		{".", "."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeGitignorePattern(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeGitignore_Dedup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .gitignore with some patterns
	gitignoreContent := `node_modules
dist
.git
custom_pattern`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignoreContent), 0644))

	// Create config with overlapping patterns
	cfg := &Config{
		Watch: WatchConfig{
			IgnorePatterns: []string{".git", "node_modules", ".memo"},
		},
	}

	err := cfg.MergeGitignore(tmpDir)
	require.NoError(t, err)

	// Count occurrences
	counts := make(map[string]int)
	for _, p := range cfg.Watch.IgnorePatterns {
		counts[p]++
	}

	// Should not have duplicates
	for pattern, count := range counts {
		assert.Equal(t, 1, count, "Pattern %s should appear only once", pattern)
	}

	// Should have all patterns
	assert.Contains(t, cfg.Watch.IgnorePatterns, ".git")
	assert.Contains(t, cfg.Watch.IgnorePatterns, "node_modules")
	assert.Contains(t, cfg.Watch.IgnorePatterns, ".memo")
	assert.Contains(t, cfg.Watch.IgnorePatterns, "dist")
	assert.Contains(t, cfg.Watch.IgnorePatterns, "custom_pattern")
}

func TestMergeGitignore_NoGitignore(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		Watch: WatchConfig{
			IgnorePatterns: []string{".git", "node_modules"},
		},
	}
	originalLen := len(cfg.Watch.IgnorePatterns)

	err := cfg.MergeGitignore(tmpDir)
	require.NoError(t, err)

	// Should not change config
	assert.Equal(t, originalLen, len(cfg.Watch.IgnorePatterns))
}
