package cli

import (
	"testing"

	"github.com/mark-chris/tmkb/internal/cli/testutil"
	"github.com/mark-chris/tmkb/internal/knowledge"
)

// resetListFlags resets list command flags and global variables
func resetListFlags() {
	verbose = false
}

// TestListCommand_AllPatterns tests listing all patterns
func TestListCommand_AllPatterns(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Configure for test
	resetListFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Execute list command
	output := captureOutput(func() {
		err = runList(listCmd, []string{})
	})

	if err != nil {
		t.Fatalf("List command failed: %v", err)
	}

	// Validate output contains all test patterns
	if !contains(output, "Found 3 pattern(s)") {
		t.Error("Expected output to show 3 patterns found")
	}

	// Check all pattern IDs are present
	expectedIDs := []string{"TMKB-TEST-001", "TMKB-TEST-002", "TMKB-TEST-003"}
	for _, id := range expectedIDs {
		if !contains(output, id) {
			t.Errorf("Expected output to contain pattern ID: %s", id)
		}
	}

	// Check pattern names are present (non-verbose mode still shows names)
	expectedNames := []string{"Test Pattern 001", "Test Pattern 002", "Test Pattern 003"}
	for _, name := range expectedNames {
		if !contains(output, name) {
			t.Errorf("Expected output to contain pattern name: %s", name)
		}
	}
}

// TestListCommand_VerboseMode tests verbose mode shows detailed information
func TestListCommand_VerboseMode(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Configure for test
	resetListFlags()
	patternsDir = fixture.Dir
	verbose = true
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Execute list command in verbose mode
	output := captureOutput(func() {
		err = runList(listCmd, []string{})
	})

	if err != nil {
		t.Fatalf("List command in verbose mode failed: %v", err)
	}

	// Validate verbose output includes detailed fields
	if !contains(output, "Name:") {
		t.Error("Expected verbose output to contain 'Name:' field")
	}

	if !contains(output, "Category:") {
		t.Error("Expected verbose output to contain 'Category:' field")
	}

	if !contains(output, "Severity:") {
		t.Error("Expected verbose output to contain 'Severity:' field")
	}

	if !contains(output, "Likelihood:") {
		t.Error("Expected verbose output to contain 'Likelihood:' field")
	}

	if !contains(output, "Language:") {
		t.Error("Expected verbose output to contain 'Language:' field")
	}

	if !contains(output, "Framework:") {
		t.Error("Expected verbose output to contain 'Framework:' field")
	}

	// Check that tier information is displayed
	if !contains(output, "[B]") {
		t.Error("Expected verbose output to contain tier information [B]")
	}

	// Verify all patterns are shown with details
	if !contains(output, "Python") || !contains(output, "Flask") {
		t.Error("Expected verbose output to show TMKB-TEST-001 language/framework")
	}

	if !contains(output, "Go") {
		t.Error("Expected verbose output to show TMKB-TEST-002 language")
	}

	if !contains(output, "JavaScript") || !contains(output, "Express") {
		t.Error("Expected verbose output to show TMKB-TEST-003 language/framework")
	}
}

// TestListCommand_EmptyDirectory tests graceful handling of empty patterns directory
func TestListCommand_EmptyDirectory(t *testing.T) {
	// Create empty temporary directory
	tmpDir := t.TempDir()

	// Configure for test
	resetListFlags()
	patternsDir = tmpDir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()

	// Load patterns from empty directory (should return empty list)
	patterns, _ := loader.LoadAll()
	index.Build(patterns)

	// Execute list command
	output := captureOutput(func() {
		err := runList(listCmd, []string{})
		if err != nil {
			t.Errorf("List command should not error on empty directory: %v", err)
		}
	})

	// Validate graceful message
	if !contains(output, "No patterns found") {
		t.Error("Expected 'No patterns found' message for empty directory")
	}

	// Ensure no panic or unexpected output
	if contains(output, "panic") || contains(output, "fatal") {
		t.Error("List command should handle empty directory gracefully")
	}
}
