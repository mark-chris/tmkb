package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/mark-chris/tmkb/internal/knowledge"
)

var (
	// Global flags
	patternsDir string
	outputFormat string
	verbose bool

	// Shared resources
	loader *knowledge.Loader
	index  *knowledge.Index
)

// rootCmd is the base command
var rootCmd = &cobra.Command{
	Use:   "tmkb",
	Short: "Threat Model Knowledge Base CLI",
	Long: `TMKB - A security context source for AI-assisted development.

Query threat patterns for authorization enforcement in multi-tenant applications.
Designed for consumption by AI coding agents, with human-readable output available.

Examples:
  # Query patterns for a context
  tmkb query --context "background job processing"

  # Get a specific pattern by ID
  tmkb get TMKB-AUTHZ-001

  # Validate all patterns
  tmkb validate --all

  # Start MCP server
  tmkb serve --port 3000`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip initialization for help commands
		if cmd.Name() == "help" || cmd.Name() == "version" {
			return nil
		}

		// Initialize loader and index
		loader = knowledge.NewLoader(patternsDir)
		index = knowledge.NewIndex()

		// Load patterns
		patterns, err := loader.LoadAll()
		if err != nil {
			return fmt.Errorf("failed to load patterns: %w", err)
		}

		// Build index
		index.Build(patterns)

		return nil
	},
}

// Execute runs the CLI
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Find default patterns directory
	defaultPatternsDir := findPatternsDir()

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&patternsDir, "patterns", "p", defaultPatternsDir,
		"Path to patterns directory")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "json",
		"Output format: json or text")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"Human-readable verbose output")

	// Add subcommands
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)
}

// findPatternsDir locates the patterns directory
func findPatternsDir() string {
	// Check common locations
	candidates := []string{
		"patterns",
		"./patterns",
		filepath.Join(os.Getenv("HOME"), ".tmkb", "patterns"),
		"/usr/local/share/tmkb/patterns",
	}

	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}

	return "patterns"
}

// getFormat returns the output format based on flags
func getFormat() knowledge.OutputFormat {
	if outputFormat == "text" || verbose {
		return knowledge.FormatText
	}
	return knowledge.FormatJSON
}
