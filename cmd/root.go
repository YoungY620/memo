package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	// Version is set by main.go from build flags
	Version = "dev"

	// Global flags
	pathFlag   string
	logLevel   string
	configFlag string
)

var rootCmd = &cobra.Command{
	Use:   "memo",
	Short: "AI-powered codebase memory",
	Long: `Memo maintains AI-readable documentation (.memo/index) for your codebase.

Commands:
  watch   Watch mode - monitors file changes and updates index continuously (default)
  scan    Scan mode  - analyzes all files once, updates index, then exits
  mcp     Query mode - starts MCP server for AI agents to query the index`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&pathFlag, "path", "p", "", "target directory (default: current dir)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "log level: error/notice/info/debug")
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// SetVersion sets the version for the root command
func SetVersion(v string) {
	Version = v
	rootCmd.Version = v
}

// resolveWorkDir resolves the working directory from the path flag
func resolveWorkDir() (string, error) {
	workDir := pathFlag
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
	}
	return filepath.Abs(workDir)
}
