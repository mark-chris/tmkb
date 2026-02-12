package cli

import (
	"fmt"

	"github.com/mark-chris/tmkb/internal/knowledge"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <pattern-id>",
	Short: "Get a specific pattern by ID",
	Long: `Retrieve detailed information about a specific threat pattern.

Examples:
  # Get pattern details (JSON)
  tmkb get TMKB-AUTHZ-001

  # Get pattern details (human-readable)
  tmkb get TMKB-AUTHZ-001 --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: runGet,
}

func runGet(cmd *cobra.Command, args []string) error {
	patternID := args[0]

	// Look up pattern
	pattern := index.GetByID(patternID)
	if pattern == nil {
		return fmt.Errorf("pattern not found: %s", patternID)
	}

	// Format output
	output, err := knowledge.FormatPatternDetail(pattern, getFormat())
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Println(output)
	return nil
}
