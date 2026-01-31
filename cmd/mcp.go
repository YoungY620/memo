package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/YoungY620/memo/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Query mode - starts MCP server for AI agents to query the index",
	Long:  `Starts an MCP server for AI agents to query the .memo/index. Requires an existing index (run 'memo' or 'memo scan' first).`,
	RunE:  runMcp,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

func runMcp(cmd *cobra.Command, args []string) error {
	workDir, err := resolveWorkDir()
	if err != nil {
		return err
	}

	// Verify index exists
	indexDir := filepath.Join(workDir, ".memo", "index")
	if _, err := os.Stat(indexDir); os.IsNotExist(err) {
		return fmt.Errorf("index directory not found: %s\nRun 'memo' or 'memo scan' first to initialize the index", indexDir)
	}

	return mcp.Serve(workDir)
}
