package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available patterns",
	Long: `List all threat patterns in the knowledge base.

Examples:
  # List all patterns
  tmkb list

  # List with verbose output
  tmkb list --verbose`,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	patterns := index.GetAll()

	if len(patterns) == 0 {
		fmt.Println("No patterns found")
		return nil
	}

	fmt.Printf("Found %d pattern(s):\n\n", len(patterns))

	for _, p := range patterns {
		if verbose {
			fmt.Printf("[%s] %s\n", p.Tier, p.ID)
			fmt.Printf("  Name:     %s\n", p.Name)
			fmt.Printf("  Category: %s > %s\n", p.Category, p.Subcategory)
			fmt.Printf("  Severity: %s | Likelihood: %s\n", p.Severity, p.Likelihood)
			fmt.Printf("  Language: %s | Framework: %s\n", p.Language, p.Framework)
			fmt.Println()
		} else {
			fmt.Printf("%s  %-20s  %s\n", p.ID, fmt.Sprintf("[%s/%s]", p.Severity, p.Tier), p.Name)
		}
	}

	return nil
}
