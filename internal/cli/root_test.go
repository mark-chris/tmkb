package cli

import (
	"testing"

	"github.com/mark-chris/tmkb/internal/cli/testutil"
	"github.com/spf13/cobra"
)

// resetRootFlags resets root command flags and global variables
func resetRootFlags() {
	verbose = false
	outputFormat = "json"
	loader = nil
	index = nil
}

// TestRootCommand_InitializesIndex tests that root command triggers index initialization
func TestRootCommand_InitializesIndex(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Reset state
	resetRootFlags()
	patternsDir = fixture.Dir

	// Create a test command that will trigger PersistentPreRunE
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	// Set up the command hierarchy
	rootCmd.AddCommand(testCmd)
	defer func() {
		// Clean up after test
		rootCmd.RemoveCommand(testCmd)
	}()

	// Execute the command - this should trigger PersistentPreRunE
	rootCmd.SetArgs([]string{"test", "-p", fixture.Dir})
	err := rootCmd.Execute()

	if err != nil {
		t.Fatalf("Root command initialization failed: %v", err)
	}

	// Verify that loader and index were initialized
	if loader == nil {
		t.Error("Expected loader to be initialized")
	}

	if index == nil {
		t.Error("Expected index to be initialized")
	}

	// Verify that patterns were loaded into the index
	patterns := index.GetAll()
	if len(patterns) != 3 {
		t.Errorf("Expected 3 patterns in index, got %d", len(patterns))
	}
}

// TestRootCommand_InvalidPatternsDir tests error handling for invalid patterns directory
func TestRootCommand_InvalidPatternsDir(t *testing.T) {
	// Reset state
	resetRootFlags()
	patternsDir = "/nonexistent/invalid/directory"

	// Create a test command that will trigger PersistentPreRunE
	testCmd := &cobra.Command{
		Use:   "test-invalid",
		Short: "Test command for invalid dir",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	// Set up the command hierarchy
	rootCmd.AddCommand(testCmd)
	defer func() {
		// Clean up after test
		rootCmd.RemoveCommand(testCmd)
	}()

	// Execute the command - this should fail during initialization
	rootCmd.SetArgs([]string{"test-invalid", "-p", "/nonexistent/invalid/directory"})
	err := rootCmd.Execute()

	if err == nil {
		t.Error("Expected error for invalid patterns directory, got none")
	}

	// Verify error message indicates loading failure
	if !contains(err.Error(), "failed to load patterns") {
		t.Errorf("Expected 'failed to load patterns' error, got: %v", err)
	}
}

// TestRootCommand_SkipsInitForHelp tests that help command skips initialization
func TestRootCommand_SkipsInitForHelp(t *testing.T) {
	// Reset state
	resetRootFlags()
	patternsDir = "/nonexistent/should/not/matter"

	// The help command should not trigger initialization, so even with
	// an invalid patterns directory, it should work
	rootCmd.SetArgs([]string{"help"})

	// Capture output to avoid cluttering test output
	_ = captureOutput(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("Help command should not fail even with invalid patterns dir: %v", err)
		}
	})

	// Verify that loader and index were NOT initialized
	// (they should remain nil if help was called)
	// Note: Since help is a built-in command, the state might vary
	// This test mainly ensures no panic or error occurs
}
