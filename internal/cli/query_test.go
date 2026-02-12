package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/mark-chris/tmkb/internal/cli/testutil"
	"github.com/mark-chris/tmkb/internal/knowledge"
)

// resetQueryFlags resets query command flags and global variables
func resetQueryFlags() {
	queryContext = ""
	queryLanguage = ""
	queryFramework = ""
	queryCategory = ""
	queryLimit = 0
	verbose = false
	outputFormat = "json"
}

// captureOutput captures stdout for testing
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// TestQueryCommand_WithContext tests querying with context returns results
func TestQueryCommand_WithContext(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Configure for test
	resetQueryFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Set query parameters
	queryContext = "background job authorization"

	// Capture output
	output := captureOutput(func() {
		err = runQuery(queryCmd, []string{})
	})

	if err != nil {
		t.Errorf("Query command failed: %v", err)
	}

	// Validate JSON output contains expected pattern
	var result knowledge.QueryResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if result.PatternCount == 0 {
		t.Error("Expected at least one pattern in results")
	}

	// Verify TMKB-TEST-001 is returned (matches keywords: background, job, authorization)
	found := false
	for _, p := range result.Patterns {
		if p.ID == "TMKB-TEST-001" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected TMKB-TEST-001 in results")
	}
}

// TestQueryCommand_LanguageFilter tests filtering by language
func TestQueryCommand_LanguageFilter(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Configure for test
	resetQueryFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Set query parameters - filter for Python patterns
	queryContext = "background job"
	queryLanguage = "python"

	// Capture output
	output := captureOutput(func() {
		err = runQuery(queryCmd, []string{})
	})

	if err != nil {
		t.Errorf("Query command with language filter failed: %v", err)
	}

	// Validate JSON output
	var result knowledge.QueryResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Test should only return TMKB-TEST-001 (Python/Flask)
	if result.PatternsIncluded != 1 {
		t.Errorf("Expected 1 pattern, got %d", result.PatternsIncluded)
	}

	if len(result.Patterns) > 0 && result.Patterns[0].ID != "TMKB-TEST-001" {
		t.Errorf("Expected TMKB-TEST-001, got %s", result.Patterns[0].ID)
	}
}

// TestQueryCommand_FrameworkFilter tests filtering by framework
func TestQueryCommand_FrameworkFilter(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Configure for test
	resetQueryFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Set query parameters - filter for Flask patterns
	queryContext = "background job"
	queryFramework = "flask"

	// Execute query
	err = runQuery(queryCmd, []string{})
	if err != nil {
		t.Errorf("Query command with framework filter failed: %v", err)
	}

	// Test should only return TMKB-TEST-001 (Python/Flask)
	// Output validation can be added later
}

// TestQueryCommand_CombinedFilters tests combining language and framework filters
func TestQueryCommand_CombinedFilters(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Configure for test
	resetQueryFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Set query parameters - combine language and framework filters
	queryContext = "background job"
	queryLanguage = "python"
	queryFramework = "flask"

	// Execute query
	err = runQuery(queryCmd, []string{})
	if err != nil {
		t.Errorf("Query command with combined filters failed: %v", err)
	}

	// Test should only return TMKB-TEST-001 (Python/Flask)
	// Output validation can be added later
}

// TestQueryCommand_LimitFlag tests the --limit flag caps results
func TestQueryCommand_LimitFlag(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Configure for test
	resetQueryFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Set query parameters with limit
	queryContext = "job authorization token" // Matches multiple patterns
	queryLimit = 1

	// Capture output
	output := captureOutput(func() {
		err = runQuery(queryCmd, []string{})
	})

	if err != nil {
		t.Errorf("Query command with limit flag failed: %v", err)
	}

	// Validate JSON output
	var result knowledge.QueryResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Test should return at most 1 pattern
	if result.PatternsIncluded > 1 {
		t.Errorf("Expected at most 1 pattern with limit=1, got %d", result.PatternsIncluded)
	}

	if len(result.Patterns) > 1 {
		t.Errorf("Expected at most 1 pattern in output, got %d", len(result.Patterns))
	}
}

// TestQueryCommand_VerboseMode tests --verbose produces human-readable output
func TestQueryCommand_VerboseMode(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Configure for test
	resetQueryFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Set query parameters with verbose mode
	queryContext = "background job"
	verbose = true

	// Capture output
	output := captureOutput(func() {
		err = runQuery(queryCmd, []string{})
	})

	if err != nil {
		t.Errorf("Query command in verbose mode failed: %v", err)
	}

	// Test should produce human-readable output with specific markers
	if !strings.Contains(output, "Found") {
		t.Error("Expected verbose output to contain 'Found' message")
	}

	if !strings.Contains(output, "threat pattern") {
		t.Error("Expected verbose output to contain 'threat pattern' text")
	}

	// Verbose output should not be JSON
	if strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Error("Expected human-readable output, got JSON")
	}
}

// TestQueryCommand_InvalidPatternsDir tests handling of missing patterns directory
func TestQueryCommand_InvalidPatternsDir(t *testing.T) {
	// Configure for test with non-existent directory
	resetQueryFlags()
	patternsDir = "/nonexistent/patterns/directory"
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()

	// Attempt to load patterns - this should fail
	patterns, err := loader.LoadAll()
	if err == nil {
		t.Error("Expected error when loading from invalid directory, got none")
	}

	// Even with error, build index with empty patterns
	index.Build(patterns)

	// Set query parameters
	queryContext = "background job"

	// Execute query - should handle gracefully (no patterns to return)
	err = runQuery(queryCmd, []string{})
	if err != nil {
		t.Errorf("Query command should handle missing patterns gracefully: %v", err)
	}

	// Test validates error handling and graceful degradation
}
