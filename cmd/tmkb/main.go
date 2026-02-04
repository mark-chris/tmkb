package main

import (
	"os"

	"github.com/mark-chris/tmkb/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
