package cli

import (
	"testing"

	"github.com/mark-chris/tmkb/internal/cli/testutil"
	"github.com/mark-chris/tmkb/internal/knowledge"
)

// resetValidateFlags resets validate command flags and global variables
func resetValidateFlags() {
	validateAll = false
	verbose = false
}

// TestValidateCommand_AllValid tests validation when all patterns are valid
func TestValidateCommand_AllValid(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Configure for test
	resetValidateFlags()
	validateAll = true
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Execute validate command for all patterns
	output := captureOutput(func() {
		err = runValidate(validateCmd, []string{})
	})

	if err != nil {
		t.Fatalf("Validate command failed: %v", err)
	}

	// Validate output shows success
	if !contains(output, "Validated 3 pattern(s)") {
		t.Error("Expected output to show 3 patterns validated")
	}

	if !contains(output, "0 error(s)") {
		t.Error("Expected output to show 0 errors for valid patterns")
	}

	// Should not show individual pattern results unless there are errors or verbose
	// (based on the implementation logic in validate.go)
}

// TestValidateCommand_SinglePattern tests validating a specific pattern
func TestValidateCommand_SinglePattern(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Configure for test
	resetValidateFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Execute validate command for single pattern
	output := captureOutput(func() {
		err = runValidate(validateCmd, []string{"TMKB-TEST-001"})
	})

	if err != nil {
		t.Fatalf("Validate command for single pattern failed: %v", err)
	}

	// Validate output shows single pattern validation
	if !contains(output, "Validated 1 pattern(s)") {
		t.Error("Expected output to show 1 pattern validated")
	}

	if !contains(output, "0 error(s)") {
		t.Error("Expected output to show 0 errors for valid pattern")
	}
}

// TestValidateCommand_InvalidPattern tests error handling for non-existent pattern
func TestValidateCommand_InvalidPattern(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Configure for test
	resetValidateFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Execute validate command for non-existent pattern
	err = runValidate(validateCmd, []string{"TMKB-INVALID-999"})

	if err == nil {
		t.Fatal("Expected error for non-existent pattern, got none")
	}

	if !contains(err.Error(), "pattern not found") {
		t.Errorf("Expected 'pattern not found' error, got: %v", err)
	}
}

// TestValidateCommand_EmptyDirectory tests graceful handling of empty patterns directory
func TestValidateCommand_EmptyDirectory(t *testing.T) {
	// Create empty temporary directory
	tmpDir := t.TempDir()

	// Configure for test
	resetValidateFlags()
	validateAll = true
	patternsDir = tmpDir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()

	// Load patterns from empty directory (should return empty list)
	patterns, _ := loader.LoadAll()
	index.Build(patterns)

	// Execute validate command
	output := captureOutput(func() {
		err := runValidate(validateCmd, []string{})
		if err != nil {
			t.Errorf("Validate command should not error on empty directory: %v", err)
		}
	})

	// Validate graceful message
	if !contains(output, "No patterns found") {
		t.Error("Expected 'No patterns found' message for empty directory")
	}

	// Ensure no panic or unexpected output
	if contains(output, "panic") || contains(output, "fatal") {
		t.Error("Validate command should handle empty directory gracefully")
	}
}
