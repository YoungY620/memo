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
	configFile = flag.String("config", "", "config file path (default .kimi-indexer.yaml)")
	rootPath   = flag.String("root", "", "watch root directory (overrides config)")
	indexPath  = flag.String("index", "", "index output directory (overrides config)")
	init_      = flag.Bool("init", false, "initialize index directory")
	once       = flag.Bool("once", false, "scan once and exit (no continuous watch)")
	verbose    = flag.Bool("v", false, "verbose output")
)

func main() {
	flag.Parse()

	// Load config
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// CLI args override
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
			log.Fatalf("failed to initialize index: %v", err)
		}
		fmt.Printf("index directory initialized: %s\n", cfg.Index.Path)
		return
	}

	// Single scan mode
	if *once {
		if err := runOnce(cfg, ana); err != nil {
			log.Fatalf("scan failed: %v", err)
		}
		return
	}

	// Continuous watch mode
	if err := runWatch(cfg, ana); err != nil {
		log.Fatalf("watch failed: %v", err)
	}
}

// runOnce 单次扫描所有文件并生成索引
func runOnce(cfg *config.Config, ana *analyzer.Analyzer) error {
	fmt.Println("scanning files...")

	// Collect all files as changes
	buf := buffer.New()
	err := collectAllFiles(cfg, buf)
	if err != nil {
		return err
	}

	if buf.IsEmpty() {
		fmt.Println("no files found for indexing")
		return nil
	}

	changes := buf.Flush()
	fmt.Printf("found %d files, starting analysis...\n", len(changes))

	ctx := context.Background()
	if err := ana.Analyze(ctx, changes); err != nil {
		return err
	}

	fmt.Printf("index updated: %s\n", cfg.Index.Path)
	return nil
}

// runWatch 持续监控模式
func runWatch(cfg *config.Config, ana *analyzer.Analyzer) error {
	fmt.Printf("starting watch: %s\n", cfg.Watcher.Root)
	fmt.Printf("index directory: %s\n", cfg.Index.Path)
	fmt.Println("press Ctrl+C to exit")

	// Create change buffer
	buf := buffer.New()

	// Create trigger manager
	triggerFn := func(changes []buffer.Change) {
		if *verbose {
			fmt.Printf("\ntriggering analysis, changed files count: %d\n", len(changes))
		}
		ctx := context.Background()
		if err := ana.Analyze(ctx, changes); err != nil {
			log.Printf("analysis failed: %v", err)
		} else if *verbose {
			fmt.Println("index updated")
		}
	}
	tm := trigger.New(&cfg.Trigger, buf, triggerFn)

	// Create file watcher
	w, err := watcher.New(&cfg.Watcher)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Start watching
	if err := w.Start(); err != nil {
		return fmt.Errorf("启动watch failed: %w", err)
	}
	defer w.Stop()

	// Start trigger manager
	tm.Start()
	defer tm.Stop()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Event loop
	for {
		select {
		case event, ok := <-w.Events():
			if !ok {
				return nil
			}
			if *verbose {
				fmt.Printf("file changed: %s [%s]\n", event.Path, event.Type)
			}
			buf.Add(event)
			tm.NotifyChange()
		case <-sigCh:
			fmt.Println("\nexiting...")
			return nil
		}
	}
}

// collectAllFiles 收集目录下所有文件作为变更
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
