package cli

import (
	"fmt"
	"os"

	"github.com/mark-chris/tmkb/internal/knowledge"
	"github.com/spf13/cobra"
)

var (
	validateAll bool
)

var validateCmd = &cobra.Command{
	Use:   "validate [pattern-id]",
	Short: "Validate threat patterns",
	Long: `Validate threat patterns against the schema requirements.

Checks for required fields, proper formatting, and Tier A/B specific requirements.

Examples:
  # Validate all patterns
  tmkb validate --all

  # Validate a specific pattern
  tmkb validate TMKB-AUTHZ-001`,
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().BoolVar(&validateAll, "all", false,
		"Validate all patterns in the patterns directory")
}

func runValidate(cmd *cobra.Command, args []string) error {
	patterns := index.GetAll()

	if len(patterns) == 0 {
		fmt.Println("No patterns found to validate")
		return nil
	}

	// Filter to specific pattern if provided
	if len(args) > 0 && !validateAll {
		patternID := args[0]
		pattern := index.GetByID(patternID)
		if pattern == nil {
			return fmt.Errorf("pattern not found: %s", patternID)
		}
		patterns = []knowledge.ThreatPattern{*pattern}
	}

	// Validate
	results := knowledge.ValidateAll(patterns)

	// Output results
	hasErrors := false
	totalErrors := 0
	totalWarnings := 0

	for _, result := range results {
		totalErrors += len(result.Errors)
		totalWarnings += len(result.Warnings)

		if !result.IsValid {
			hasErrors = true
		}

		// Print results for each pattern
		if len(result.Errors) > 0 || len(result.Warnings) > 0 || verbose {
			status := "✓"
			if !result.IsValid {
				status = "✗"
			}
			fmt.Printf("%s %s\n", status, result.PatternID)

			for _, err := range result.Errors {
				fmt.Printf("  ERROR: %s - %s\n", err.Field, err.Message)
			}
			for _, warn := range result.Warnings {
				fmt.Printf("  WARN:  %s - %s\n", warn.Field, warn.Message)
			}
			if len(result.Errors) > 0 || len(result.Warnings) > 0 {
				fmt.Println()
			}
		}
	}

	// Summary
	fmt.Printf("\nValidated %d pattern(s): %d error(s), %d warning(s)\n",
		len(results), totalErrors, totalWarnings)

	if hasErrors {
		os.Exit(1)
	}

	return nil
}
