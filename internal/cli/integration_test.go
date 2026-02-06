package cli

import (
	"encoding/json"
	"testing"

	"github.com/mark-chris/tmkb/internal/cli/testutil"
	"github.com/mark-chris/tmkb/internal/knowledge"
)

// TestWorkflow_QueryThenGet tests the workflow of querying patterns then getting details
func TestWorkflow_QueryThenGet(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Initialize system
	resetQueryFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Step 1: Query for patterns matching a context
	queryContext = "background job authorization"
	queryOutput := captureOutput(func() {
		err = runQuery(queryCmd, []string{})
	})

	if err != nil {
		t.Fatalf("Query step failed: %v", err)
	}

	// Parse query results
	var queryResult knowledge.QueryResult
	if err := json.Unmarshal([]byte(queryOutput), &queryResult); err != nil {
		t.Fatalf("Failed to parse query output: %v", err)
	}

	if queryResult.PatternCount == 0 {
		t.Fatal("Query returned no patterns")
	}

	// Step 2: Get detailed information for the first pattern found
	firstPatternID := queryResult.Patterns[0].ID
	resetGetFlags()

	getOutput := captureOutput(func() {
		err = runGet(getCmd, []string{firstPatternID})
	})

	if err != nil {
		t.Fatalf("Get step failed: %v", err)
	}

	// Parse get results
	var pattern knowledge.ThreatPattern
	if err := json.Unmarshal([]byte(getOutput), &pattern); err != nil {
		t.Fatalf("Failed to parse get output: %v", err)
	}

	// Verify we got the same pattern
	if pattern.ID != firstPatternID {
		t.Errorf("Expected pattern ID %s, got %s", firstPatternID, pattern.ID)
	}

	// Verify get provides more detail than query
	if pattern.Description == "" {
		t.Error("Expected full pattern description in get output")
	}

	if len(pattern.Mitigations) == 0 {
		t.Error("Expected mitigations in get output")
	}
}

// TestWorkflow_ListThenValidate tests the workflow of listing patterns then validating them
func TestWorkflow_ListThenValidate(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Initialize system
	resetListFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Step 1: List all patterns
	listOutput := captureOutput(func() {
		err = runList(listCmd, []string{})
	})

	if err != nil {
		t.Fatalf("List step failed: %v", err)
	}

	// Verify list shows patterns
	if !contains(listOutput, "Found 3 pattern(s)") {
		t.Error("Expected list to show 3 patterns")
	}

	// Extract pattern IDs from list (basic check)
	patternIDs := []string{"TMKB-TEST-001", "TMKB-TEST-002", "TMKB-TEST-003"}
	for _, id := range patternIDs {
		if !contains(listOutput, id) {
			t.Errorf("Expected list to contain pattern: %s", id)
		}
	}

	// Step 2: Validate one of the patterns from the list
	resetValidateFlags()

	validateOutput := captureOutput(func() {
		err = runValidate(validateCmd, []string{"TMKB-TEST-001"})
	})

	if err != nil {
		t.Fatalf("Validate step failed: %v", err)
	}

	// Verify validation succeeded
	if !contains(validateOutput, "Validated 1 pattern(s)") {
		t.Error("Expected validation of 1 pattern")
	}

	if !contains(validateOutput, "0 error(s)") {
		t.Error("Expected validation to show 0 errors")
	}

	// Step 3: Validate all patterns
	resetValidateFlags()
	validateAll = true

	validateAllOutput := captureOutput(func() {
		err = runValidate(validateCmd, []string{})
	})

	if err != nil {
		t.Fatalf("Validate all step failed: %v", err)
	}

	// Verify all patterns validated
	if !contains(validateAllOutput, "Validated 3 pattern(s)") {
		t.Error("Expected validation of all 3 patterns")
	}
}

// TestWorkflow_FilterChain tests progressive filtering with query parameters
func TestWorkflow_FilterChain(t *testing.T) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Initialize system
	resetQueryFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Step 1: Query with broad context (should return multiple patterns)
	queryContext = "job token authorization"
	broadOutput := captureOutput(func() {
		err = runQuery(queryCmd, []string{})
	})

	if err != nil {
		t.Fatalf("Broad query failed: %v", err)
	}

	var broadResult knowledge.QueryResult
	if err := json.Unmarshal([]byte(broadOutput), &broadResult); err != nil {
		t.Fatalf("Failed to parse broad query output: %v", err)
	}

	broadCount := broadResult.PatternCount

	// Step 2: Add language filter to narrow results
	resetQueryFlags()
	queryContext = "job token authorization"
	queryLanguage = "python"

	filteredOutput := captureOutput(func() {
		err = runQuery(queryCmd, []string{})
	})

	if err != nil {
		t.Fatalf("Filtered query failed: %v", err)
	}

	var filteredResult knowledge.QueryResult
	if err := json.Unmarshal([]byte(filteredOutput), &filteredResult); err != nil {
		t.Fatalf("Failed to parse filtered query output: %v", err)
	}

	filteredCount := filteredResult.PatternCount

	// Verify that filtering reduced the result set (or kept it same if only Python matched)
	if filteredCount > broadCount {
		t.Errorf("Expected filtered query to return fewer or equal patterns, got %d (was %d)",
			filteredCount, broadCount)
	}

	// Step 3: Add framework filter for even more specific results
	resetQueryFlags()
	queryContext = "job token authorization"
	queryLanguage = "python"
	queryFramework = "flask"

	specificOutput := captureOutput(func() {
		err = runQuery(queryCmd, []string{})
	})

	if err != nil {
		t.Fatalf("Specific query failed: %v", err)
	}

	var specificResult knowledge.QueryResult
	if err := json.Unmarshal([]byte(specificOutput), &specificResult); err != nil {
		t.Fatalf("Failed to parse specific query output: %v", err)
	}

	// Verify progressive filtering
	if specificResult.PatternCount > filteredCount {
		t.Errorf("Expected framework filter to return fewer or equal patterns")
	}

	// Verify that the Python/Flask pattern is returned
	foundPythonFlask := false
	for _, p := range specificResult.Patterns {
		if p.ID == "TMKB-TEST-001" {
			foundPythonFlask = true
			break
		}
	}

	if specificResult.PatternCount > 0 && !foundPythonFlask {
		t.Error("Expected to find TMKB-TEST-001 (Python/Flask) in specific query results")
	}
}
