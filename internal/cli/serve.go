package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	servePort int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server for AI agent integration",
	Long: `Start a Model Context Protocol (MCP) server that AI agents can query.

The MCP server exposes the tmkb_query tool for agent integration with
Claude Code, Cursor, and other MCP-compatible AI coding assistants.

Examples:
  # Start server on default port
  tmkb serve

  # Start server on custom port
  tmkb serve --port 3000`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 3000,
		"Port to listen on")
}

func runServe(cmd *cobra.Command, args []string) error {
	fmt.Printf("Starting MCP server on port %d...\n", servePort)
	fmt.Println("Loaded", index.Count(), "patterns")
	fmt.Println()
	fmt.Println("MCP server implementation pending.")
	fmt.Println("For now, use the CLI directly or integrate via stdout.")
	
	// TODO: Implement MCP server
	// See internal/mcp/server.go for implementation
	
	return nil
}
