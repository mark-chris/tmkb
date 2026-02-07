package cli

import (
	"fmt"
	"log"
	"os"

	"github.com/mark-chris/tmkb/internal/mcp"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server for AI agent integration",
	Long: `Start a Model Context Protocol (MCP) server that AI agents can query.

The MCP server communicates via stdin/stdout using the JSON-RPC 2.0 protocol.
It is designed to be invoked by MCP clients like Claude Code.

Examples:
  # Start MCP server (typically invoked by Claude Code)
  tmkb serve`,
	RunE: runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	// Create MCP server with the loaded pattern index
	server := mcp.NewServer(index)

	// Log to stderr (stdout is reserved for protocol communication)
	log.SetOutput(os.Stderr)
	log.Printf("Starting MCP server with %d patterns loaded", index.Count())
	log.Println("Server ready for MCP protocol communication via stdio")

	// Run server - blocks until stdin closes (EOF)
	if err := server.ServeStdio(os.Stdin, os.Stdout); err != nil {
		return fmt.Errorf("MCP server error: %w", err)
	}

	log.Println("MCP server shutdown")
	return nil
}
