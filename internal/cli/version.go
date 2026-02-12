package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the application version, set at build time via -ldflags.
var Version = "0.1.0"

// GitCommit is the git commit hash, set at build time via -ldflags.
var GitCommit = "unknown"

// BuildDate is the build date, set at build time via -ldflags.
var BuildDate = "unknown"

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
