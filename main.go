package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/YoungY620/memo/mcp"
)

var Version = "dev"

func main() {
	var (
		pathFlag           = flag.String("path", "", "Path to watch (default: current directory)")
		configFlag         = flag.String("config", "config.yaml", "Path to config file")
		versionFlag        = flag.Bool("version", false, "Print version and exit")
		onceFlag           = flag.Bool("once", false, "Run once and exit (no watch mode)")
		mcpFlag            = flag.Bool("mcp", false, "Run as MCP server (stdio)")
		mcpWithWatcherFlag = flag.Bool("mcp-with-watcher", false, "Run MCP server with watcher subprocess")
		logLevelFlag       = flag.String("log-level", "", "Log level: error, notice, info, debug")
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
			log.Fatalf("[ERROR] Failed to get current directory: %v", err)
		}
	}
	workDir, _ = filepath.Abs(workDir)

	// MCP with Watcher mode
	if *mcpWithWatcherFlag {
		// Spawn watcher subprocess
		cmd := exec.Command(os.Args[0], "--path", workDir)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err != nil {
			log.Fatalf("[ERROR] Failed to start watcher subprocess: %v", err)
		}

		// Handle signals to kill subprocess
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			cmd.Process.Kill()
			os.Exit(0)
		}()

		// Run MCP server
		if err := mcp.Serve(workDir); err != nil {
			cmd.Process.Kill()
			log.Fatalf("[ERROR] MCP server error: %v", err)
		}
		cmd.Process.Kill()
		return
	}

	// MCP server only mode
	if *mcpFlag {
		indexDir := filepath.Join(workDir, ".memo", "index")
		if _, err := os.Stat(indexDir); os.IsNotExist(err) {
			log.Fatalf("[ERROR] Index directory not found: %s\nRun 'memo' first to initialize the index.", indexDir)
		}
		if err := mcp.Serve(workDir); err != nil {
			log.Fatalf("[ERROR] MCP server error: %v", err)
		}
		return
	}

	// Load config
	cfg, err := LoadConfig(*configFlag)
	if err != nil {
		log.Fatalf("[ERROR] Failed to load config: %v", err)
	}
	// Set log level: flag takes precedence over config
	if *logLevelFlag != "" {
		SetLogLevel(*logLevelFlag)
	} else {
		SetLogLevel(cfg.LogLevel)
	}
	logDebug("Config loaded: logLevel=%s, debounce=%dms, maxWait=%dms", cfg.LogLevel, cfg.Watch.DebounceMs, cfg.Watch.MaxWaitMs)

	// Merge .gitignore patterns if found
	if err := cfg.MergeGitignore(workDir); err != nil {
		logError("Failed to load .gitignore: %v", err)
	}
	logDebug("Total ignore patterns: %d", len(cfg.Watch.IgnorePatterns))

	// Initialize .memo/index directory
	indexDir := filepath.Join(workDir, ".memo", "index")
	if err := initIndex(indexDir); err != nil {
		log.Fatalf("[ERROR] Failed to initialize .memo/index: %v", err)
	}
	logDebug("Initialized .memo/index directory: %s", indexDir)

	// Create analyser
	analyser := NewAnalyser(cfg, workDir)

	// Create watcher
	watcher, err := NewWatcher(workDir, cfg.Watch.IgnorePatterns, cfg.Watch.DebounceMs, cfg.Watch.MaxWaitMs, func(files []string) {
		logInfo("Triggered with %d changed files", len(files))
		logDebug("Changed files: %v", files)
		ctx := context.Background()
		if err := analyser.Analyse(ctx, files); err != nil {
			logError("Analysis failed: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("[ERROR] Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Initial scan of all files
	logInfo("Starting initial scan...")
	watcher.ScanAll()
	logDebug("Initial scan completed")

	// Once mode: flush and exit
	if *onceFlag {
		watcher.Flush()
		logInfo("Once mode completed")
		return
	}

	// Watch mode
	logInfo("Memo watching: %s", workDir)

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := watcher.Run(); err != nil {
			logError("Watcher error: %v", err)
		}
	}()

	<-sigChan
	logInfo("Shutting down...")
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
			logDebug("Creating %s", path)
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return err
			}
		}
	}

	return nil
}
