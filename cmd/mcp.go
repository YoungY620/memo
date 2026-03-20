package cmd

import (
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

	return mcp.Serve(workDir)
}
