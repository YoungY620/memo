package main

import (
	"context"
	"flag"
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/YoungY620/memo/analyzer"
	"github.com/YoungY620/memo/internal"
	"github.com/YoungY620/memo/mcp"
)

var Version = "dev"

func main() {
	var (
		pathFlag     = flag.String("path", "", "Path to watch (default: current directory)")
		configFlag   = flag.String("config", "config.yaml", "Path to config file")
		versionFlag  = flag.Bool("version", false, "Print version and exit")
		onceFlag     = flag.Bool("once", false, "Run once and exit (no watch mode)")
		mcpFlag      = flag.Bool("mcp", false, "Run as MCP server (stdio)")
		logLevelFlag = flag.String("log-level", "", "Log level: error, notice, info, debug")
	)
	flag.Parse()

	if *versionFlag {
		fmt.Printf("memo %s\n", Version)
		os.Exit(0)
	}

	// Determine work directory
	workDir := *pathFlag
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			stdlog.Fatalf("[ERROR] Failed to get current directory: %v", err)
		}
	}
	workDir, _ = filepath.Abs(workDir)

	// MCP server mode
	if *mcpFlag {
		indexDir := filepath.Join(workDir, ".memo", "index")
		if _, err := os.Stat(indexDir); os.IsNotExist(err) {
			stdlog.Fatalf("[ERROR] Index directory not found: %s\nRun 'memo' first to initialize the index.", indexDir)
		}
		if err := mcp.Serve(workDir); err != nil {
			stdlog.Fatalf("[ERROR] MCP server error: %v", err)
		}
		return
	}

	// Load config
	cfg, err := LoadConfig(*configFlag)
	if err != nil {
		stdlog.Fatalf("[ERROR] Failed to load config: %v", err)
	}
	// Set log level: flag takes precedence over config
	if *logLevelFlag != "" {
		internal.SetLogLevel(*logLevelFlag)
	} else {
		internal.SetLogLevel(cfg.LogLevel)
	}
	internal.LogDebug("Config loaded: logLevel=%s, debounce=%dms, maxWait=%dms", cfg.LogLevel, cfg.Watch.DebounceMs, cfg.Watch.MaxWaitMs)

	// Merge .gitignore patterns if found
	if err := cfg.MergeGitignore(workDir); err != nil {
		internal.LogError("Failed to load .gitignore: %v", err)
	}
	internal.LogDebug("Total ignore patterns: %d", len(cfg.Watch.IgnorePatterns))

	// Initialize .memo/index directory
	indexDir := filepath.Join(workDir, ".memo", "index")
	if err := initIndex(indexDir); err != nil {
		stdlog.Fatalf("[ERROR] Failed to initialize .memo/index: %v", err)
	}
	internal.LogDebug("Initialized .memo/index directory: %s", indexDir)

	// Acquire single instance lock (watcher mode only)
	memoDir := filepath.Join(workDir, ".memo")
	lockFile, err := analyzer.TryLock(memoDir)
	if err != nil {
		stdlog.Fatalf("[ERROR] %v", err)
	}
	defer analyzer.Unlock(lockFile)

	// Initialize history logger for watcher
	internal.InitHistoryLogger(memoDir, "watcher")
	defer internal.CloseHistoryLogger()

	// Ensure status is idle on startup and exit
	if err := analyzer.SetStatus(memoDir, "idle"); err != nil {
		internal.LogError("Failed to set initial status: %v", err)
	}
	defer func() {
		if err := analyzer.SetStatus(memoDir, "idle"); err != nil {
			internal.LogError("Failed to reset status on exit: %v", err)
		}
	}()

	// Create analyser
	agentCfg := analyzer.AgentConfig{
		APIKey: cfg.Agent.APIKey,
		Model:  cfg.Agent.Model,
	}
	ana := analyzer.NewAnalyser(agentCfg, workDir)

	// Create watcher
	watcher, err := analyzer.NewWatcher(workDir, cfg.Watch.IgnorePatterns, cfg.Watch.DebounceMs, cfg.Watch.MaxWaitMs, func(files []string) {
		internal.LogInfo("Triggered with %d changed files", len(files))
		internal.LogDebug("Changed files: %v", files)
		ctx := context.Background()
		if err := ana.Analyse(ctx, files); err != nil {
			internal.LogError("Analysis failed: %v", err)
		}
	})
	if err != nil {
		stdlog.Fatalf("[ERROR] Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Print banner (watcher and once mode, not MCP mode)
	analyzer.PrintBanner(analyzer.BannerOptions{
		WorkDir: workDir,
		Version: Version,
	})

	// Initial scan of all files
	internal.LogInfo("Watcher started, workDir=%s", workDir)
	watcher.ScanAll()
	internal.LogDebug("Initial scan completed")

	// Once mode: flush and exit
	if *onceFlag {
		watcher.Flush()
		internal.LogInfo("Once mode completed")
		return
	}

	// Watch mode
	internal.LogInfo("Memo watching: %s", workDir)

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := watcher.Run(); err != nil {
			internal.LogError("Watcher error: %v", err)
		}
	}()

	<-sigChan
	internal.LogInfo("Shutting down...")
}

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
