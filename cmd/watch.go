package cmd

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/YoungY620/memo/analyzer"
	"github.com/YoungY620/memo/internal"
	"github.com/spf13/cobra"
)

var (
	skipScan bool
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch mode - monitors file changes and updates index continuously",
	Long:  `Continuously monitors file changes and updates .memo/index. This is the default command.`,
	RunE:  runWatch,
}

func init() {
	watchCmd.Flags().StringVarP(&configFlag, "config", "c", "config.yaml", "config file path")
	watchCmd.Flags().BoolVar(&skipScan, "skip-scan", false, "skip initial full scan")
	rootCmd.AddCommand(watchCmd)

	// Set watch as the default command when no subcommand is provided
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runWatch(cmd, args)
	}
}

func runWatch(cmd *cobra.Command, args []string) error {
	workDir, err := resolveWorkDir()
	if err != nil {
		return err
	}

	cfg, err := loadConfigAndSetup(workDir)
	if err != nil {
		return err
	}

	// Initialize .memo/index directory
	indexDir := filepath.Join(workDir, ".memo", "index")
	if err := initIndex(indexDir); err != nil {
		return err
	}
	internal.LogDebug("Initialized .memo/index directory: %s", indexDir)

	// Acquire single instance lock
	memoDir := filepath.Join(workDir, ".memo")
	lockFile, err := analyzer.TryLock(memoDir)
	if err != nil {
		return err
	}
	defer analyzer.Unlock(lockFile)

	// Initialize history logger
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
		return err
	}
	defer watcher.Close()

	// Start async update check
	updateCh := internal.CheckUpdateAsync(Version)

	// Print banner
	var updateInfo *analyzer.UpdateInfo
	select {
	case result := <-updateCh:
		if result != nil {
			updateInfo = &analyzer.UpdateInfo{
				LatestVersion: result.LatestVersion,
				UpdateCommand: result.UpdateCommand,
			}
		}
	default:
		// Update check not ready yet, continue without it
	}

	analyzer.PrintBanner(analyzer.BannerOptions{
		WorkDir:    workDir,
		Version:    Version,
		UpdateInfo: updateInfo,
	})

	// Initial scan (unless --skip-scan is set)
	internal.LogInfo("Watcher started, workDir=%s", workDir)
	if !skipScan {
		watcher.ScanAll()
		internal.LogDebug("Initial scan completed")
	} else {
		internal.LogInfo("Skipping initial scan (--skip-scan)")
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
	return nil
}
