package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var Version = "dev"

func main() {
	var (
		pathFlag    = flag.String("path", "", "Path to watch (default: current directory)")
		configFlag  = flag.String("config", "config.yaml", "Path to config file")
		versionFlag = flag.Bool("version", false, "Print version and exit")
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

	// Load config
	cfg, err := LoadConfig(*configFlag)
	if err != nil {
		log.Fatalf("[ERROR] Failed to load config: %v", err)
	}
	SetLogLevel(cfg.LogLevel)
	logDebug("Config loaded: logLevel=%s, debounce=%dms, maxWait=%dms", cfg.LogLevel, cfg.Watch.DebounceMs, cfg.Watch.MaxWaitMs)

	// Initialize .memo directory
	memoDir := filepath.Join(workDir, ".memo")
	if err := initMemo(memoDir); err != nil {
		log.Fatalf("[ERROR] Failed to initialize .memo: %v", err)
	}
	logDebug("Initialized .memo directory: %s", memoDir)

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

	logInfo("Memo watching: %s", workDir)

	// Initial scan of all files
	logInfo("Starting initial scan...")
	watcher.ScanAll()
	logDebug("Initial scan completed")

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

func initMemo(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	files := map[string]string{
		"arch.json":      `{"modules": [], "relationships": ""}`,
		"interface.json": `{"external": [], "internal": []}`,
		"stories.json":   `{"stories": []}`,
		"issues.json":    `{"issues": []}`,
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			logDebug("Creating %s", path)
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return err
			}
		}
	}

	return nil
}
