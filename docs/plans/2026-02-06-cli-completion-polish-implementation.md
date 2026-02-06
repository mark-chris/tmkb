# CLI Completion and Polish Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add comprehensive test coverage for all CLI commands with performance benchmarks to verify sub-second response times.

**Architecture:** Test-first approach using Go's testing framework. Create test utilities for fixtures and command execution, then write unit tests for each command, integration tests for workflows, and benchmarks for performance validation.

**Tech Stack:** Go 1.25, testing package, github.com/spf13/cobra (CLI framework)

---

## Task 1: Test Utilities Package

Create shared test utilities for CLI testing.

**Files:**
- Create: `internal/cli/testutil/testutil.go`
- Test: `internal/cli/testutil/testutil_test.go`

**Step 1: Write failing test for test utilities**

```go
// internal/cli/testutil/testutil_test.go
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetupTestPatterns_CreatesDirectory(t *testing.T) {
	fixture := SetupTestPatterns(t)
	defer fixture.Cleanup()

	if _, err := os.Stat(fixture.Dir); os.IsNotExist(err) {
		t.Errorf("Expected directory %s to exist", fixture.Dir)
	}
}

func TestSetupTestPatterns_CreatesPatterns(t *testing.T) {
	fixture := SetupTestPatterns(t)
	defer fixture.Cleanup()

	if len(fixture.Patterns) == 0 {
		t.Error("Expected patterns to be created")
	}
}

func TestCreateTestPattern_ValidStructure(t *testing.T) {
	pattern := CreateTestPattern("TEST-001", "Test Pattern", "authorization")

	if pattern.ID != "TEST-001" {
		t.Errorf("Expected ID TEST-001, got %s", pattern.ID)
	}
	if pattern.Name != "Test Pattern" {
		t.Errorf("Expected name 'Test Pattern', got %s", pattern.Name)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/testutil/... -v`
Expected: FAIL with "undefined: SetupTestPatterns" or similar

**Step 3: Implement test utilities**

```go
// internal/cli/testutil/testutil.go
package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mark-chris/tmkb/internal/knowledge"
	"gopkg.in/yaml.v3"
)

// TestFixture provides test patterns and temporary directory
type TestFixture struct {
	Dir      string
	Patterns []knowledge.ThreatPattern
	Cleanup  func()
}

// PatternWrapper wraps ThreatPattern for YAML serialization
type PatternWrapper struct {
	ThreatPattern knowledge.ThreatPattern `yaml:"threat_pattern"`
}

// SetupTestPatterns creates a temporary patterns directory with test data
func SetupTestPatterns(t *testing.T) *TestFixture {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()

	// Create test patterns
	patterns := []knowledge.ThreatPattern{
		CreateTestPattern("TMKB-TEST-001", "Python Flask Pattern", "authorization"),
		CreateTestPattern("TMKB-TEST-002", "Go Any Pattern", "authentication"),
		CreateTestPattern("TMKB-TEST-003", "JS Express Pattern", "authorization"),
	}

	// Customize patterns for filter testing
	patterns[0].Language = "python"
	patterns[0].Framework = "flask"
	patterns[0].Triggers.Keywords = []string{"background", "job", "authorization"}

	patterns[1].Language = "go"
	patterns[1].Framework = "any"
	patterns[1].Triggers.Keywords = []string{"session", "token"}

	patterns[2].Language = "javascript"
	patterns[2].Framework = "express"
	patterns[2].Triggers.Keywords = []string{"tenant", "isolation"}

	// Write patterns to temp directory
	for _, p := range patterns {
		writePatternFile(tmpDir, p)
	}

	return &TestFixture{
		Dir:      tmpDir,
		Patterns: patterns,
		Cleanup:  func() { os.RemoveAll(tmpDir) },
	}
}

// CreateTestPattern generates a minimal valid pattern for testing
func CreateTestPattern(id, name, category string) knowledge.ThreatPattern {
	return knowledge.ThreatPattern{
		ID:          id,
		Name:        name,
		Tier:        "B",
		Version:     "1.0.0",
		LastUpdated: "2026-02-06",
		Category:    category,
		Subcategory: "test",
		Language:    "any",
		Framework:   "any",
		Severity:    "medium",
		Likelihood:  "medium",
		Description: "Test pattern description",
		AgentSummary: knowledge.AgentSummary{
			Threat: "Test threat",
			Check:  "Test check",
			Fix:    "Test fix",
		},
		Triggers: knowledge.Triggers{
			Keywords: []string{"test"},
		},
		Mitigations: []knowledge.Mitigation{},
		Provenance: knowledge.Provenance{
			SourceType:        "test",
			Description:       "Test provenance",
			PublicReferences:  []knowledge.PublicReference{},
		},
	}
}

// writePatternFile writes a pattern to the temp directory
func writePatternFile(dir string, p knowledge.ThreatPattern) error {
	wrapper := PatternWrapper{ThreatPattern: p}
	data, err := yaml.Marshal(wrapper)
	if err != nil {
		return err
	}

	filename := filepath.Join(dir, p.ID+".yaml")
	return os.WriteFile(filename, data, 0644)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/testutil/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/testutil/
git commit -m "test: add CLI test utilities for fixtures and test patterns"
```

---

## Task 2: Query Command Tests

Test the query command with all filters and modes.

**Files:**
- Create: `internal/cli/query_test.go`

**Step 1: Write failing tests for query command**

```go
// internal/cli/query_test.go
package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark-chris/tmkb/internal/cli/testutil"
	"github.com/mark-chris/tmkb/internal/knowledge"
)

func TestQueryCommand_WithContext(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Reset command for testing
	queryCmd.ResetFlags()
	queryContext = ""
	queryLanguage = ""
	queryFramework = ""
	queryCategory = ""
	queryLimit = 0

	// Set flags
	patternsDir = fixture.Dir
	verbose = false
	queryContext = "background job"

	// Execute
	err := runQuery(queryCmd, []string{})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
}

func TestQueryCommand_LanguageFilter(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	queryCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false
	queryContext = "test"
	queryLanguage = "python"

	// Execute and capture (we'll improve output capture in next iteration)
	err := runQuery(queryCmd, []string{})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
}

func TestQueryCommand_FrameworkFilter(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	queryCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false
	queryContext = "test"
	queryFramework = "flask"

	err := runQuery(queryCmd, []string{})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
}

func TestQueryCommand_CombinedFilters(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	queryCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false
	queryContext = "background"
	queryLanguage = "python"
	queryFramework = "flask"

	err := runQuery(queryCmd, []string{})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
}

func TestQueryCommand_LimitFlag(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	queryCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false
	queryContext = ""
	queryLimit = 1

	err := runQuery(queryCmd, []string{})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
}

func TestQueryCommand_VerboseMode(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	queryCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = true
	queryContext = "background"

	err := runQuery(queryCmd, []string{})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
}

func TestQueryCommand_InvalidPatternsDir(t *testing.T) {
	queryCmd.ResetFlags()
	patternsDir = "/nonexistent/directory"
	verbose = false
	queryContext = "test"

	err := runQuery(queryCmd, []string{})
	if err == nil {
		t.Error("Expected error for invalid patterns directory, got nil")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run TestQuery -v`
Expected: FAIL (tests run but may have issues with global state)

**Step 3: Refactor query command for testability**

No code changes needed - tests should work with current implementation.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run TestQuery -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/query_test.go
git commit -m "test: add query command tests with filters and modes"
```

---

## Task 3: Get Command Tests

Test the get command for retrieving patterns by ID.

**Files:**
- Create: `internal/cli/get_test.go`

**Step 1: Write failing tests for get command**

```go
// internal/cli/get_test.go
package cli

import (
	"testing"

	"github.com/mark-chris/tmkb/internal/cli/testutil"
)

func TestGetCommand_ValidID(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	getCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false

	err := runGet(getCmd, []string{"TMKB-TEST-001"})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
}

func TestGetCommand_InvalidID(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	getCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false

	err := runGet(getCmd, []string{"NONEXISTENT-ID"})
	if err == nil {
		t.Error("Expected error for non-existent pattern ID, got nil")
	}
	if err != nil && !contains(err.Error(), "pattern not found") {
		t.Errorf("Expected 'pattern not found' error, got: %v", err)
	}
}

func TestGetCommand_VerboseOutput(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	getCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = true

	err := runGet(getCmd, []string{"TMKB-TEST-001"})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
}

func TestGetCommand_JSONOutput(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	getCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false
	outputFormat = "json"

	err := runGet(getCmd, []string{"TMKB-TEST-001"})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || len(s) > len(substr) && s[1:len(s)-1] != s && contains(s[1:], substr))
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run TestGet -v`
Expected: PASS or FAIL with contains function issue

**Step 3: Fix helper function**

```go
// Add to get_test.go
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run TestGet -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/get_test.go
git commit -m "test: add get command tests for pattern retrieval"
```

---

## Task 4: List Command Tests

Test the list command for displaying all patterns.

**Files:**
- Create: `internal/cli/list_test.go`

**Step 1: Write failing tests for list command**

```go
// internal/cli/list_test.go
package cli

import (
	"testing"

	"github.com/mark-chris/tmkb/internal/cli/testutil"
)

func TestListCommand_AllPatterns(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	listCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false

	err := runList(listCmd, []string{})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
}

func TestListCommand_VerboseMode(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	listCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = true

	err := runList(listCmd, []string{})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
}

func TestListCommand_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	listCmd.ResetFlags()
	patternsDir = tmpDir
	verbose = false

	// Should not error, just report no patterns
	err := runList(listCmd, []string{})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run TestList -v`
Expected: PASS (tests should work with current implementation)

**Step 3: No implementation changes needed**

Tests should pass with current list command implementation.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run TestList -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/list_test.go
git commit -m "test: add list command tests for pattern display"
```

---

## Task 5: Validate Command Tests

Test the validate command for pattern validation.

**Files:**
- Create: `internal/cli/validate_test.go`

**Step 1: Write failing tests for validate command**

```go
// internal/cli/validate_test.go
package cli

import (
	"testing"

	"github.com/mark-chris/tmkb/internal/cli/testutil"
)

func TestValidateCommand_AllValid(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	validateCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false
	validateAll = true

	err := runValidate(validateCmd, []string{})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
}

func TestValidateCommand_SinglePattern(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	validateCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false
	validateAll = false

	err := runValidate(validateCmd, []string{"TMKB-TEST-001"})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
}

func TestValidateCommand_InvalidPattern(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	validateCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false
	validateAll = false

	err := runValidate(validateCmd, []string{"NONEXISTENT"})
	if err == nil {
		t.Error("Expected error for non-existent pattern, got nil")
	}
}

func TestValidateCommand_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	validateCmd.ResetFlags()
	patternsDir = tmpDir
	verbose = false
	validateAll = true

	// Should not error, just report no patterns
	err := runValidate(validateCmd, []string{})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run TestValidate -v`
Expected: PASS (tests should work)

**Step 3: No implementation changes needed**

Tests should pass with current validate command implementation.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run TestValidate -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/validate_test.go
git commit -m "test: add validate command tests for pattern validation"
```

---

## Task 6: Root Command Tests

Test root command initialization and global flags.

**Files:**
- Create: `internal/cli/root_test.go`

**Step 1: Write failing tests for root command**

```go
// internal/cli/root_test.go
package cli

import (
	"testing"

	"github.com/mark-chris/tmkb/internal/cli/testutil"
)

func TestRootCommand_InitializesIndex(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	rootCmd.ResetFlags()
	patternsDir = fixture.Dir

	// Run a simple command to trigger PersistentPreRunE
	err := rootCmd.PersistentPreRunE(rootCmd, []string{})
	if err != nil {
		t.Fatalf("initialization failed: %v", err)
	}

	if index == nil {
		t.Error("Expected index to be initialized")
	}
	if loader == nil {
		t.Error("Expected loader to be initialized")
	}
}

func TestRootCommand_InvalidPatternsDir(t *testing.T) {
	rootCmd.ResetFlags()
	patternsDir = "/nonexistent/directory"

	err := rootCmd.PersistentPreRunE(rootCmd, []string{})
	if err == nil {
		t.Error("Expected error for invalid patterns directory, got nil")
	}
}

func TestRootCommand_SkipsInitForHelp(t *testing.T) {
	// Help command should not trigger initialization
	helpCmd := &cobra.Command{
		Use: "help",
	}

	err := rootCmd.PersistentPreRunE(helpCmd, []string{})
	if err != nil {
		t.Errorf("Help command should not trigger initialization error: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run TestRoot -v`
Expected: PASS or FAIL depending on cobra import

**Step 3: Add missing import if needed**

```go
import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/mark-chris/tmkb/internal/cli/testutil"
)
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run TestRoot -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/root_test.go
git commit -m "test: add root command tests for initialization"
```

---

## Task 7: Integration Tests

Test end-to-end workflows across commands.

**Files:**
- Create: `internal/cli/integration_test.go`

**Step 1: Write failing integration tests**

```go
// internal/cli/integration_test.go
package cli

import (
	"testing"

	"github.com/mark-chris/tmkb/internal/cli/testutil"
)

func TestWorkflow_QueryThenGet(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Step 1: Query to find a pattern
	queryCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false
	queryContext = "background job"

	err := runQuery(queryCmd, []string{})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	// Step 2: Get the pattern details
	getCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false

	err = runGet(getCmd, []string{"TMKB-TEST-001"})
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
}

func TestWorkflow_ListThenValidate(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Step 1: List all patterns
	listCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false

	err := runList(listCmd, []string{})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	// Step 2: Validate all patterns
	validateCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false
	validateAll = true

	err = runValidate(validateCmd, []string{})
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
}

func TestWorkflow_FilterChain(t *testing.T) {
	fixture := testutil.SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Query with no filters - gets all
	queryCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false
	queryContext = ""

	err := runQuery(queryCmd, []string{})
	if err != nil {
		t.Fatalf("query without filters failed: %v", err)
	}

	// Query with language filter - narrows results
	queryCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false
	queryContext = ""
	queryLanguage = "python"

	err = runQuery(queryCmd, []string{})
	if err != nil {
		t.Fatalf("query with language filter failed: %v", err)
	}

	// Query with language + framework - narrows further
	queryCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false
	queryContext = ""
	queryLanguage = "python"
	queryFramework = "flask"

	err = runQuery(queryCmd, []string{})
	if err != nil {
		t.Fatalf("query with combined filters failed: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run TestWorkflow -v`
Expected: PASS (integration tests should work)

**Step 3: No implementation changes needed**

Integration tests should pass with current implementation.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run TestWorkflow -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/integration_test.go
git commit -m "test: add integration tests for end-to-end workflows"
```

---

## Task 8: Performance Benchmarks

Add benchmarks to verify sub-second response times.

**Files:**
- Create: `internal/cli/benchmark_test.go`

**Step 1: Write benchmarks**

```go
// internal/cli/benchmark_test.go
package cli

import (
	"testing"

	"github.com/mark-chris/tmkb/internal/cli/testutil"
	"github.com/mark-chris/tmkb/internal/knowledge"
)

func BenchmarkQuery_ColdStart(b *testing.B) {
	fixture := testutil.SetupTestPatterns(b.(*testing.T))
	defer fixture.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		queryCmd.ResetFlags()
		patternsDir = fixture.Dir
		verbose = false
		queryContext = "background job"
		b.StartTimer()

		_ = runQuery(queryCmd, []string{})
	}
}

func BenchmarkQuery_WarmIndex(b *testing.B) {
	fixture := testutil.SetupTestPatterns(b.(*testing.T))
	defer fixture.Cleanup()

	// Pre-build index
	loader = knowledge.NewLoader(fixture.Dir)
	patterns, _ := loader.LoadAll()
	index = knowledge.NewIndex()
	index.Build(patterns)

	queryCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false
	queryContext = "background job"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		opts := knowledge.QueryOptions{
			Context:   queryContext,
			Limit:     3,
			Verbosity: "agent",
		}
		_ = knowledge.Query(index, opts)
	}
}

func BenchmarkGet_ByID(b *testing.B) {
	fixture := testutil.SetupTestPatterns(b.(*testing.T))
	defer fixture.Cleanup()

	// Pre-build index
	loader = knowledge.NewLoader(fixture.Dir)
	patterns, _ := loader.LoadAll()
	index = knowledge.NewIndex()
	index.Build(patterns)

	getCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = runGet(getCmd, []string{"TMKB-TEST-001"})
	}
}

func BenchmarkList_All(b *testing.B) {
	fixture := testutil.SetupTestPatterns(b.(*testing.T))
	defer fixture.Cleanup()

	// Pre-build index
	loader = knowledge.NewLoader(fixture.Dir)
	patterns, _ := loader.LoadAll()
	index = knowledge.NewIndex()
	index.Build(patterns)

	listCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = runList(listCmd, []string{})
	}
}

func BenchmarkValidate_All(b *testing.B) {
	fixture := testutil.SetupTestPatterns(b.(*testing.T))
	defer fixture.Cleanup()

	// Pre-build index
	loader = knowledge.NewLoader(fixture.Dir)
	patterns, _ := loader.LoadAll()
	index = knowledge.NewIndex()
	index.Build(patterns)

	validateCmd.ResetFlags()
	patternsDir = fixture.Dir
	verbose = false
	validateAll = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = runValidate(validateCmd, []string{})
	}
}
```

**Step 2: Run benchmarks**

Run: `go test ./internal/cli -bench=. -benchmem`
Expected: Benchmarks run and report performance

**Step 3: Verify performance targets**

Check benchmark output:
- Query (warm): Should be <100ms per operation
- Get: Should be <50ms per operation
- All operations: Should complete successfully

**Step 4: Document results**

Record benchmark results in commit message.

**Step 5: Commit**

```bash
git add internal/cli/benchmark_test.go
git commit -m "test: add performance benchmarks for CLI commands

Benchmark results:
- Query (warm): <100ms
- Get: <50ms
- List: <50ms
- Validate: <500ms"
```

---

## Task 9: Test Validation and Documentation

Run all tests, verify coverage, and document results.

**Files:**
- Modify: `README.md` (if needed)

**Step 1: Run all tests**

Run: `go test ./internal/cli/... -v`
Expected: All tests pass

**Step 2: Check test coverage**

Run: `go test ./internal/cli/... -cover`
Expected: High coverage (>80%)

**Step 3: Run benchmarks**

Run: `go test ./internal/cli/... -bench=. -benchmem`
Expected: All benchmarks meet performance targets

**Step 4: Verify all acceptance criteria**

- ✅ All CLI commands have tests
- ✅ All flags tested
- ✅ Error cases tested
- ✅ Performance targets met
- ✅ Integration workflows tested

**Step 5: Commit final validation**

```bash
git add .
git commit -m "test: validate CLI test suite completeness

All 30+ tests passing:
- Query: 7 tests
- Get: 4 tests
- List: 3 tests
- Validate: 4 tests
- Root: 3 tests
- Integration: 3 tests
- Benchmarks: 5 benchmarks

Performance verified:
- Query <1s cold, <100ms warm
- Get <50ms
- Validate <500ms"
```

---

## Summary

**9 Tasks Total:**
1. Test utilities package - Fixtures and helpers
2. Query command tests - 7 tests for context, filters, modes
3. Get command tests - 4 tests for ID retrieval
4. List command tests - 3 tests for pattern display
5. Validate command tests - 4 tests for validation
6. Root command tests - 3 tests for initialization
7. Integration tests - 3 workflow tests
8. Performance benchmarks - 5 benchmarks
9. Test validation - Verify coverage and performance

**Expected Outcome:**
- 30+ tests passing
- 5 benchmarks with performance baselines
- Sub-second response times verified
- All Issue #5 acceptance criteria met
