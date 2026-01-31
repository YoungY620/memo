package cmd

import (
	"os"
	"path/filepath"

	"github.com/YoungY620/memo/internal"
)

// initIndex initializes the .memo/index directory with default files
func initIndex(indexDir string) error {
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return err
	}

	files := map[string]string{
		"arch.json":      `{"modules": [], "relationships": ""}`,
		"interface.json": `{"external": [], "internal": []}`,
		"stories.json":   `{"stories": []}`,
		"issues.json":    `{"issues": []}`,
	}

	for name, content := range files {
		path := filepath.Join(indexDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			internal.LogDebug("Creating %s", path)
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return err
			}
		}
	}

	// Create local MCP config file for watcher sessions
	// This prevents loading ~/.kimi/mcp.json (which may contain memo itself)
	// Users can customize this file to add MCP servers for watcher sessions
	memoDir := filepath.Dir(indexDir)
	mcpFile := filepath.Join(memoDir, "mcp.json")
	if _, err := os.Stat(mcpFile); os.IsNotExist(err) {
		internal.LogDebug("Creating %s", mcpFile)
		if err := os.WriteFile(mcpFile, []byte("{}"), 0644); err != nil {
			return err
		}
	}

	// Create .gitignore to exclude runtime files from version control
	gitignoreFile := filepath.Join(memoDir, ".gitignore")
	if _, err := os.Stat(gitignoreFile); os.IsNotExist(err) {
		gitignoreContent := `# Runtime files - do not commit
watcher.lock
status.json
.history
`
		internal.LogDebug("Creating %s", gitignoreFile)
		if err := os.WriteFile(gitignoreFile, []byte(gitignoreContent), 0644); err != nil {
			return err
		}
	}

	return nil
}

// loadConfigAndSetup loads config and sets up logging
func loadConfigAndSetup(workDir string) (*Config, error) {
	cfg, err := LoadConfig(configFlag)
	if err != nil {
		return nil, err
	}

	// Set log level: flag takes precedence over config
	if logLevel != "" {
		internal.SetLogLevel(logLevel)
	} else {
		internal.SetLogLevel(cfg.LogLevel)
	}
	internal.LogDebug("Config loaded: logLevel=%s, debounce=%dms, maxWait=%dms",
		cfg.LogLevel, cfg.Watch.DebounceMs, cfg.Watch.MaxWaitMs)

	// Merge .gitignore patterns if found
	if err := cfg.MergeGitignore(workDir); err != nil {
		internal.LogError("Failed to load .gitignore: %v", err)
	}
	internal.LogDebug("Total ignore patterns: %d", len(cfg.Watch.IgnorePatterns))

	return cfg, nil
}
