package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark-chris/tmkb/internal/cli/testutil"
	"github.com/mark-chris/tmkb/internal/knowledge"
)

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// resetGetFlags resets get command flags and global variables
func resetGetFlags() {
	verbose = false
	outputFormat = "json"
}

// TestGetCommand_ValidID tests retrieving a pattern by valid ID
func TestGetCommand_ValidID(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Configure for test
	resetGetFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Execute get command for valid ID
	output := captureOutput(func() {
		err = runGet(getCmd, []string{"TMKB-TEST-001"})
	})

	if err != nil {
		t.Fatalf("Get command failed: %v", err)
	}

	// Validate JSON output contains the pattern
	var pattern knowledge.ThreatPattern
	if err := json.Unmarshal([]byte(output), &pattern); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if pattern.ID != "TMKB-TEST-001" {
		t.Errorf("Expected pattern ID TMKB-TEST-001, got %s", pattern.ID)
	}

	if pattern.Name != "Test Pattern 001" {
		t.Errorf("Expected pattern name 'Test Pattern 001', got %s", pattern.Name)
	}

	if pattern.Language != "Python" {
		t.Errorf("Expected language Python, got %s", pattern.Language)
	}
}

// TestGetCommand_InvalidID tests handling of non-existent pattern ID
func TestGetCommand_InvalidID(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Configure for test
	resetGetFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Execute get command for invalid ID
	err = runGet(getCmd, []string{"TMKB-INVALID-999"})

	if err == nil {
		t.Fatal("Expected error for invalid pattern ID, got none")
	}

	if !contains(err.Error(), "pattern not found") {
		t.Errorf("Expected 'pattern not found' error, got: %v", err)
	}
}

// TestGetCommand_VerboseOutput tests verbose mode produces human-readable output
func TestGetCommand_VerboseOutput(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Configure for test
	resetGetFlags()
	patternsDir = fixture.Dir
	verbose = true
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Execute get command with verbose mode
	output := captureOutput(func() {
		err = runGet(getCmd, []string{"TMKB-TEST-002"})
	})

	if err != nil {
		t.Fatalf("Get command in verbose mode failed: %v", err)
	}

	// Validate human-readable output
	if !contains(output, "TMKB-TEST-002") {
		t.Error("Expected output to contain pattern ID")
	}

	if !contains(output, "Test Pattern 002") {
		t.Error("Expected output to contain pattern name")
	}

	if !contains(output, "AGENT SUMMARY") {
		t.Error("Expected verbose output to contain 'AGENT SUMMARY' section")
	}

	if !contains(output, "Tier:") {
		t.Error("Expected verbose output to contain 'Tier:' field")
	}

	// Verbose output should not be JSON
	trimmed := strings.TrimSpace(output)
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		t.Error("Expected human-readable output, got JSON")
	}
}

// TestGetCommand_JSONOutput tests JSON output format
func TestGetCommand_JSONOutput(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Configure for test
	resetGetFlags()
	patternsDir = fixture.Dir
	outputFormat = "json"
	verbose = false
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Execute get command with JSON format
	output := captureOutput(func() {
		err = runGet(getCmd, []string{"TMKB-TEST-003"})
	})

	if err != nil {
		t.Fatalf("Get command with JSON format failed: %v", err)
	}

	// Validate JSON structure
	var pattern knowledge.ThreatPattern
	if err := json.Unmarshal([]byte(output), &pattern); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify all required fields are present
	if pattern.ID != "TMKB-TEST-003" {
		t.Errorf("Expected pattern ID TMKB-TEST-003, got %s", pattern.ID)
	}

	if pattern.Language != "JavaScript" {
		t.Errorf("Expected language JavaScript, got %s", pattern.Language)
	}

	if pattern.Framework != "Express" {
		t.Errorf("Expected framework Express, got %s", pattern.Framework)
	}

	if pattern.AgentSummary.Threat == "" {
		t.Error("Expected agent summary threat to be populated")
	}

	if pattern.Triggers.Keywords == nil {
		t.Error("Expected triggers keywords to be present")
	}
}
