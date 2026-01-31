package cmd

import (
	"context"
	"path/filepath"

	"github.com/YoungY620/memo/analyzer"
	"github.com/YoungY620/memo/internal"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan mode - analyzes all files once, updates index, then exits",
	Long:  `Analyzes all files in the codebase once, updates .memo/index, then exits. Useful for CI or initial setup.`,
	RunE:  runScan,
}

func init() {
	scanCmd.Flags().StringVarP(&configFlag, "config", "c", "config.yaml", "config file path")
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
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

	// Create watcher (reuse for scanning logic)
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

	// Scan all files
	internal.LogInfo("Scanning all files, workDir=%s", workDir)
	watcher.ScanAll()
	internal.LogDebug("Scan completed")

	// Flush and exit
	watcher.Flush()
	internal.LogInfo("Scan mode completed")
	return nil
}
