package cmd

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
