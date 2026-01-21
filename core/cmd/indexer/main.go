// Package main is the kimi-indexer CLI entry point
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/user/kimi-sdk-agent-indexer/core/internal/analyzer"
	"github.com/user/kimi-sdk-agent-indexer/core/internal/buffer"
	"github.com/user/kimi-sdk-agent-indexer/core/internal/config"
	"github.com/user/kimi-sdk-agent-indexer/core/internal/trigger"
	"github.com/user/kimi-sdk-agent-indexer/core/internal/watcher"
)

var (
	configFile = flag.String("config", "", "Configuration file path (default .kimi-indexer.yaml)")
	rootPath   = flag.String("root", "", "Root directory to monitor (overrides config)")
	indexPath  = flag.String("index", "", "Index output directory (overrides config)")
	init_      = flag.Bool("init", false, "Initialize index directory")
	once       = flag.Bool("once", false, "Scan once and exit (no continuous monitoring)")
	verbose    = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Command line overrides
	if *rootPath != "" {
		cfg.Watcher.Root = *rootPath
	}
	if *indexPath != "" {
		cfg.Index.Path = *indexPath
	}

	// Initialize analyzer
	ana := analyzer.New(cfg)

	// Init mode
	if *init_ {
		ctx := context.Background()
		if err := ana.InitIndex(ctx); err != nil {
			log.Fatalf("Failed to initialize index: %v", err)
		}
		fmt.Printf("Index directory initialized: %s\n", cfg.Index.Path)
		return
	}

	// Single scan mode
	if *once {
		if err := runOnce(cfg, ana); err != nil {
			log.Fatalf("Scan failed: %v", err)
		}
		return
	}

	// Continuous monitoring mode
	if err := runWatch(cfg, ana); err != nil {
		log.Fatalf("Watch failed: %v", err)
	}
}

// runOnce scans all files once and generates index
func runOnce(cfg *config.Config, ana *analyzer.Analyzer) error {
	fmt.Println("Scanning files...")

	// Collect all files as changes
	buf := buffer.New()
	err := collectAllFiles(cfg, buf)
	if err != nil {
		return err
	}

	if buf.IsEmpty() {
		fmt.Println("No files found to index")
		return nil
	}

	changes := buf.Flush()
	fmt.Printf("Found %d files, analyzing...\n", len(changes))

	ctx := context.Background()
	if err := ana.Analyze(ctx, changes); err != nil {
		return err
	}

	fmt.Printf("Index updated: %s\n", cfg.Index.Path)
	return nil
}

// runWatch continuous monitoring mode
func runWatch(cfg *config.Config, ana *analyzer.Analyzer) error {
	fmt.Printf("Starting watch: %s\n", cfg.Watcher.Root)
	fmt.Printf("Index directory: %s\n", cfg.Index.Path)
	fmt.Println("Press Ctrl+C to exit")

	// Create change buffer
	buf := buffer.New()

	// Create trigger manager
	triggerFn := func(changes []buffer.Change) {
		if *verbose {
			fmt.Printf("\nTriggering analysis, changed files: %d\n", len(changes))
		}
		ctx := context.Background()
		if err := ana.Analyze(ctx, changes); err != nil {
			log.Printf("Analysis failed: %v", err)
		} else if *verbose {
			fmt.Println("Index updated")
		}
	}
	tm := trigger.New(&cfg.Trigger, buf, triggerFn)

	// Create file watcher
	w, err := watcher.New(&cfg.Watcher)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Start monitoring
	if err := w.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}
	defer w.Stop()

	// Start trigger manager
	tm.Start()
	defer tm.Stop()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Event processing loop
	for {
		select {
		case event, ok := <-w.Events():
			if !ok {
				return nil
			}
			if *verbose {
				fmt.Printf("File change: %s [%s]\n", event.Path, event.Type)
			}
			buf.Add(event)
			tm.NotifyChange()
		case <-sigCh:
			fmt.Println("\nExiting...")
			return nil
		}
	}
}

// collectAllFiles collects all files in directory as changes
func collectAllFiles(cfg *config.Config, buf *buffer.Buffer) error {
	root := cfg.Watcher.Root

	// Build ignore and extension maps
	ignoreMap := make(map[string]bool)
	for _, pattern := range cfg.Watcher.Ignore {
		ignoreMap[pattern] = true
	}

	extMap := make(map[string]bool)
	for _, ext := range cfg.Watcher.Extensions {
		extMap[ext] = true
	}

	return walkDir(root, root, ignoreMap, extMap, buf)
}

func walkDir(root, dir string, ignoreMap, extMap map[string]bool, buf *buffer.Buffer) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // Ignore inaccessible directories
	}

	for _, entry := range entries {
		name := entry.Name()

		// Check ignore
		if ignoreMap[name] {
			continue
		}

		path := dir + "/" + name

		if entry.IsDir() {
			walkDir(root, path, ignoreMap, extMap, buf)
		} else {
			// Check extension
			if len(extMap) > 0 {
				ext := ""
				for i := len(name) - 1; i >= 0; i-- {
					if name[i] == '.' {
						ext = name[i:]
						break
					}
				}
				if !extMap[ext] {
					continue
				}
			}

			// Add as create event
			buf.Add(watcher.Event{
				Path: path,
				Type: watcher.EventCreate,
			})
		}
	}

	return nil
}
