package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

func main() {
	var (
		pathFlag   = flag.String("path", "", "Path to watch (default: current directory)")
		configFlag = flag.String("config", "config.yaml", "Path to config file")
	)
	flag.Parse()

	// Determine work directory
	workDir := *pathFlag
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get current directory: %v", err)
		}
	}
	workDir, _ = filepath.Abs(workDir)

	// Load config
	cfg, err := LoadConfig(*configFlag)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize .baecon directory
	baeconDir := filepath.Join(workDir, ".baecon")
	if err := initBaecon(baeconDir); err != nil {
		log.Fatalf("Failed to initialize .baecon: %v", err)
	}

	// Create analyser
	analyser := NewAnalyser(cfg, workDir)

	// Create watcher
	watcher, err := NewWatcher(workDir, cfg.Watch.IgnorePatterns, cfg.Watch.DebounceMs, func(files []string) {
		log.Printf("Files changed: %v", files)
		ctx := context.Background()
		if err := analyser.Analyse(ctx, files); err != nil {
			log.Printf("Analysis failed: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	log.Printf("Lightkeeper watching: %s", workDir)

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := watcher.Run(); err != nil {
			log.Printf("Watcher error: %v", err)
		}
	}()

	<-sigChan
	log.Println("Shutting down...")
}

func initBaecon(dir string) error {
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
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return err
			}
		}
	}

	return nil
}
