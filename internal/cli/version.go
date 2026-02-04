package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version   = "0.1.0"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("tmkb version %s\n", Version)
		if verbose {
			fmt.Printf("  Git commit: %s\n", GitCommit)
			fmt.Printf("  Build date: %s\n", BuildDate)
		}
	},
}
