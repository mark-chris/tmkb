package cli

import (
	"fmt"

	"github.com/mark-chris/tmkb/internal/knowledge"
	"github.com/spf13/cobra"
)

var (
	queryContext   string
	queryLanguage  string
	queryFramework string
	queryCategory  string
	queryLimit     int
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query threat patterns by context",
	Long: `Query the threat model knowledge base for patterns relevant to your implementation.

Returns structured, actionable security context optimized for AI agent consumption.
Use --verbose for human-readable detailed output.

Examples:
  # Query by context
  tmkb query --context "multi-tenant API endpoint"

  # Query with language/framework filter
  tmkb query --context "background job" --language python --framework flask

  # Get verbose human-readable output
  tmkb query --context "file upload processing" --verbose

  # Limit results
  tmkb query --context "authorization" --limit 5`,
	RunE: runQuery,
}

func init() {
	queryCmd.Flags().StringVarP(&queryContext, "context", "c", "",
		"What you're implementing (e.g., 'multi-tenant API', 'background job')")
	queryCmd.Flags().StringVarP(&queryLanguage, "language", "l", "",
		"Programming language filter (e.g., python)")
	queryCmd.Flags().StringVar(&queryFramework, "framework", "",
		"Framework filter (e.g., flask)")
	queryCmd.Flags().StringVar(&queryCategory, "category", "",
		"Category filter (e.g., authorization)")
	queryCmd.Flags().IntVar(&queryLimit, "limit", 0,
		"Maximum number of patterns to return (default: 3 for agent, 10 for verbose)")
}

func runQuery(cmd *cobra.Command, args []string) error {
	// Build query options
	verbosity := "agent"
	if verbose {
		verbosity = "human"
	}

	opts := knowledge.QueryOptions{
		Context:   queryContext,
		Language:  queryLanguage,
		Framework: queryFramework,
		Category:  queryCategory,
		Limit:     queryLimit,
		Verbosity: verbosity,
	}

	// Execute query
	result := knowledge.Query(index, opts)

	// Format output
	output, err := knowledge.FormatOutput(result, getFormat(), verbose)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Println(output)
	return nil
}
