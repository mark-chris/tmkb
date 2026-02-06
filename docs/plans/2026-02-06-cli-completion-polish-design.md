# CLI Completion and Polish Design

**Date**: 2026-02-06
**Issue**: #5
**Status**: Approved

## Overview

Complete CLI implementation with comprehensive test coverage and performance validation. All commands are implemented; this phase focuses on verification, testing, and ensuring sub-second response times.

## Goals

1. Add comprehensive CLI test coverage (currently 0%)
2. Verify all commands work correctly with filters and flags
3. Benchmark and ensure sub-second response time
4. Test error handling and edge cases
5. Validate output formats (JSON and text)

## Current State Assessment

### Implemented Commands

**Core Commands:**
- ✅ `query` - Query patterns by context with filters
- ✅ `get` - Retrieve pattern by ID
- ✅ `list` - List all available patterns
- ✅ `validate` - Validate patterns against schema
- ✅ `serve` - Start MCP server
- ✅ `version` - Print version information

**Global Flags:**
- ✅ `--verbose` - Human-readable verbose output
- ✅ `--format` - Output format (json/text)
- ✅ `--patterns` - Patterns directory path

**Query Filters:**
- ✅ `--context` - Context description for relevance matching
- ✅ `--language` - Programming language filter
- ✅ `--framework` - Framework filter
- ✅ `--category` - Category filter
- ✅ `--limit` - Maximum results

### What's Missing

- ❌ **CLI test coverage** - No tests exist
- ❌ **Performance benchmarks** - No validation of response times
- ❌ **Integration tests** - No end-to-end workflow tests
- ❌ **Error case validation** - No tests for error handling

## Design Decisions

### Test-First Approach

**Decision**: Write comprehensive tests before adding new features

**Rationale**:
- All features are implemented
- Testing validates existing functionality
- Finds bugs before adding complexity
- Establishes baseline performance metrics

**Alternative**: Add new features first (shell completion, better errors), then test. Rejected because it delays validation of existing code.

### Test Organization

**Decision**: One test file per command plus shared utilities

```
internal/cli/
├── testutil/
│   └── testutil.go          # Shared test utilities
├── query_test.go             # Query command tests
├── get_test.go               # Get command tests
├── list_test.go              # List command tests
├── validate_test.go          # Validate command tests
├── root_test.go              # Root command & flags tests
├── integration_test.go       # End-to-end workflows
└── benchmark_test.go         # Performance benchmarks
```

**Rationale**:
- Mirrors production code organization
- Easy to locate relevant tests
- Utilities prevent duplication
- Benchmarks separate from functional tests

### Test Utilities Design

**Decision**: Provide helper functions for common test operations

```go
// internal/cli/testutil/testutil.go
package testutil

type TestFixture struct {
    Dir      string                      // Temporary patterns directory
    Patterns []knowledge.ThreatPattern   // Test patterns
    Cleanup  func()                      // Cleanup function
}

func SetupTestPatterns(t *testing.T) *TestFixture
func ExecuteCommand(cmd *cobra.Command, args ...string) (output string, err error)
func AssertJSONValid(t *testing.T, output string)
func AssertContains(t *testing.T, output, expected string)
func CreateTestPattern(id, name, category string) knowledge.ThreatPattern
```

**Rationale**:
- Reduces boilerplate in tests
- Consistent test setup across files
- Makes tests more readable
- Easier to maintain

## Test Coverage

### Query Command Tests (query_test.go)

**Basic Queries:**
1. `TestQueryCommand_WithContext` - Valid context returns results
2. `TestQueryCommand_EmptyContext` - Returns all patterns sorted by severity
3. `TestQueryCommand_NoMatches` - Context with no keyword matches returns empty

**Filters:**
4. `TestQueryCommand_LanguageFilter` - Filters by language correctly
5. `TestQueryCommand_FrameworkFilter` - Filters by framework correctly
6. `TestQueryCommand_CategoryFilter` - Filters by category correctly
7. `TestQueryCommand_CombinedFilters` - Language + framework together

**Output Modes:**
8. `TestQueryCommand_JSONOutput` - Default JSON format valid
9. `TestQueryCommand_VerboseOutput` - Verbose flag produces human-readable text
10. `TestQueryCommand_TokenLimit` - Agent mode respects 500 token limit
11. `TestQueryCommand_LimitFlag` - --limit flag caps results

**Error Cases:**
12. `TestQueryCommand_InvalidPatternsDir` - Handles missing patterns directory
13. `TestQueryCommand_CorruptPatterns` - Graceful error on invalid YAML

### Get Command Tests (get_test.go)

1. `TestGetCommand_ValidID` - Retrieves pattern by ID
2. `TestGetCommand_InvalidID` - Returns error for non-existent ID
3. `TestGetCommand_JSONOutput` - JSON format complete and valid
4. `TestGetCommand_VerboseOutput` - Verbose mode human-readable

### List Command Tests (list_test.go)

1. `TestListCommand_AllPatterns` - Lists all patterns
2. `TestListCommand_VerboseMode` - Shows detailed info per pattern
3. `TestListCommand_CompactMode` - One-line per pattern
4. `TestListCommand_EmptyDirectory` - Handles no patterns gracefully

### Validate Command Tests (validate_test.go)

1. `TestValidateCommand_AllValid` - All patterns pass validation
2. `TestValidateCommand_SinglePattern` - Validates specific pattern by ID
3. `TestValidateCommand_InvalidPattern` - Detects and reports validation errors
4. `TestValidateCommand_ExitCode` - Returns exit code 1 on validation errors

### Root Command Tests (root_test.go)

1. `TestRootCommand_Help` - --help flag displays usage
2. `TestRootCommand_Version` - version command works
3. `TestRootCommand_GlobalFlags` - Global flags (--verbose, --format) work
4. `TestRootCommand_PatternsDirFlag` - --patterns flag changes directory

### Integration Tests (integration_test.go)

**End-to-End Workflows:**
1. `TestWorkflow_QueryThenGet` - Query finds pattern, get retrieves details
2. `TestWorkflow_ListThenValidate` - List patterns, validate each one
3. `TestWorkflow_FilterChain` - Multiple filters narrow results progressively

### Performance Benchmarks (benchmark_test.go)

**Benchmarks:**
1. `BenchmarkQuery_ColdStart` - First query with index building
2. `BenchmarkQuery_WarmIndex` - Query with pre-built index
3. `BenchmarkQuery_LargeResultSet` - Query returning max patterns
4. `BenchmarkGet_ByID` - Get single pattern performance
5. `BenchmarkValidate_All` - Validate all patterns performance

**Performance Targets:**
- Query (warm index): <100ms (p95)
- Query (cold start): <1s (p95)
- Get by ID: <50ms (p95)
- Validate all: <500ms (p95)

## Test Fixtures

### Test Patterns

**Minimal test patterns for filter testing:**

```yaml
# TMKB-TEST-001 - Python/Flask pattern
threat_pattern:
  id: TMKB-TEST-001
  name: Test Pattern Python Flask
  category: authorization
  language: python
  framework: flask
  severity: high
  likelihood: medium
  triggers:
    keywords: [background, job, authorization]
  # ... minimal required fields

# TMKB-TEST-002 - Go/any pattern
threat_pattern:
  id: TMKB-TEST-002
  name: Test Pattern Go
  category: authentication
  language: go
  framework: any
  severity: medium
  likelihood: high
  triggers:
    keywords: [session, token]
  # ... minimal required fields

# TMKB-TEST-003 - Authorization category
threat_pattern:
  id: TMKB-TEST-003
  name: Test Pattern Authorization
  category: authorization
  language: javascript
  framework: express
  severity: critical
  likelihood: high
  triggers:
    keywords: [tenant, isolation]
  # ... minimal required fields

# TMKB-TEST-INVALID - Invalid pattern for validation tests
threat_pattern:
  id: TMKB-TEST-INVALID
  # Missing required fields
  category: authorization
```

## Implementation Details

### Test Utility Implementation

**SetupTestPatterns:**
```go
func SetupTestPatterns(t *testing.T) *TestFixture {
    // Create temporary directory
    tmpDir := t.TempDir()

    // Create test patterns
    patterns := []knowledge.ThreatPattern{
        CreateTestPattern("TMKB-TEST-001", "Python Flask", "authorization"),
        CreateTestPattern("TMKB-TEST-002", "Go Any", "authentication"),
        CreateTestPattern("TMKB-TEST-003", "JS Express", "authorization"),
    }

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
```

**ExecuteCommand:**
```go
func ExecuteCommand(cmd *cobra.Command, args ...string) (string, error) {
    // Capture stdout
    old := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    // Execute command
    cmd.SetArgs(args)
    err := cmd.Execute()

    // Restore stdout and read output
    w.Close()
    os.Stdout = old
    output, _ := io.ReadAll(r)

    return string(output), err
}
```

### Example Test Structure

```go
func TestQueryCommand_WithContext(t *testing.T) {
    // Setup
    fixture := testutil.SetupTestPatterns(t)
    defer fixture.Cleanup()

    // Create command
    cmd := queryCmd
    cmd.SetArgs([]string{
        "--context", "background job",
        "--patterns", fixture.Dir,
    })

    // Execute
    output, err := testutil.ExecuteCommand(cmd)

    // Assert
    if err != nil {
        t.Fatalf("command failed: %v", err)
    }

    testutil.AssertJSONValid(t, output)
    testutil.AssertContains(t, output, "TMKB-TEST-001")

    // Parse and verify structure
    var result knowledge.QueryResult
    if err := json.Unmarshal([]byte(output), &result); err != nil {
        t.Fatalf("invalid JSON: %v", err)
    }

    if result.PatternCount == 0 {
        t.Error("expected patterns, got none")
    }
}
```

### Benchmark Structure

```go
func BenchmarkQuery_WarmIndex(b *testing.B) {
    // Setup once
    fixture := setupBenchmarkFixture(b)
    defer fixture.Cleanup()

    // Pre-build index
    loader := knowledge.NewLoader(fixture.Dir)
    patterns, _ := loader.LoadAll()
    index := knowledge.NewIndex()
    index.Build(patterns)

    // Benchmark query execution
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        opts := knowledge.QueryOptions{
            Context:   "background job authorization",
            Limit:     3,
            Verbosity: "agent",
        }
        _ = knowledge.Query(index, opts)
    }
}
```

## Error Handling

### Missing Patterns Directory

**Test**: `TestQueryCommand_InvalidPatternsDir`

```go
cmd.SetArgs([]string{
    "--context", "test",
    "--patterns", "/nonexistent/directory",
})
output, err := testutil.ExecuteCommand(cmd)

// Expected: Error message, exit code 1
assert.Error(t, err)
assert.Contains(t, output, "failed to load patterns")
```

### Corrupt Pattern Files

**Test**: `TestQueryCommand_CorruptPatterns`

```go
// Create directory with invalid YAML
tmpDir := t.TempDir()
os.WriteFile(filepath.Join(tmpDir, "corrupt.yaml"), []byte("invalid: [yaml"), 0644)

cmd.SetArgs([]string{
    "--context", "test",
    "--patterns", tmpDir,
})
output, err := testutil.ExecuteCommand(cmd)

// Expected: Graceful error, not panic
assert.Error(t, err)
assert.Contains(t, output, "failed to parse YAML")
```

### Invalid Arguments

**Test**: `TestGetCommand_InvalidID`

```go
cmd.SetArgs([]string{
    "NONEXISTENT-ID",
    "--patterns", fixture.Dir,
})
output, err := testutil.ExecuteCommand(cmd)

// Expected: Clear error message
assert.Error(t, err)
assert.Contains(t, output, "pattern not found")
```

## Edge Cases

### Empty Patterns Directory

**Behavior**: Commands handle gracefully with informative messages

- `list` → "No patterns found"
- `query` → Returns empty result (pattern_count: 0)
- `validate` → "No patterns found to validate"

**Tests**:
- `TestListCommand_EmptyDirectory`
- `TestQueryCommand_EmptyDirectory`
- `TestValidateCommand_EmptyDirectory`

### Large Result Sets

**Behavior**: Respects limits and defaults

- Query with `--limit 0` → Uses default (3 for agent, 10 for verbose)
- Query with `--limit 100` → Returns max available patterns

**Tests**:
- `TestQueryCommand_DefaultLimit`
- `TestQueryCommand_ExplicitLimit`

### Token Limit Boundary

**Behavior**: Respects 500 token limit in agent mode

- Pattern at exactly 500 tokens → Included
- Pattern exceeding 500 tokens → Excluded (unless only pattern)

**Tests**:
- `TestQueryCommand_TokenLimit` (already covered by Issue #4 tests)

### Concurrent Access

**Behavior**: Thread-safe query execution

**Benchmark**: `BenchmarkQuery_Concurrent`
```go
func BenchmarkQuery_Concurrent(b *testing.B) {
    // Setup
    fixture := setupBenchmarkFixture(b)
    defer fixture.Cleanup()

    loader := knowledge.NewLoader(fixture.Dir)
    patterns, _ := loader.LoadAll()
    index := knowledge.NewIndex()
    index.Build(patterns)

    // Concurrent queries
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            opts := knowledge.QueryOptions{
                Context: "background job",
                Limit:   3,
            }
            _ = knowledge.Query(index, opts)
        }
    })
}
```

## Success Criteria

### Functionality
- ✅ All 30+ test cases pass
- ✅ No panics or crashes on invalid input
- ✅ Exit codes correct (0 = success, 1 = error)
- ✅ JSON output valid and parseable
- ✅ Verbose output human-readable
- ✅ All filters work correctly
- ✅ All flags work correctly

### Performance
- ✅ Query (warm index): <100ms (p95)
- ✅ Query (cold start): <1s (p95)
- ✅ Get by ID: <50ms (p95)
- ✅ Validate all: <500ms (p95)
- ✅ No memory leaks in benchmark runs
- ✅ Concurrent queries maintain performance

### Coverage
- ✅ All CLI commands have tests
- ✅ All flags tested (--verbose, --format, filters)
- ✅ Error paths tested
- ✅ Integration workflows tested
- ✅ Benchmark baselines established

### Test Execution

```bash
# Run all tests
go test ./internal/cli/... -v

# Run with coverage
go test ./internal/cli/... -cover -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# Run benchmarks
go test ./internal/cli/... -bench=. -benchmem

# Run benchmarks with profiling
go test ./internal/cli/... -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof
```

## Testing Strategy

### Phase 1: Test Utilities (Task 1)
- Create `internal/cli/testutil/testutil.go`
- Implement helper functions
- Create test pattern fixtures
- Unit test the utilities themselves

### Phase 2: Command Tests (Tasks 2-6)
- Implement query_test.go (13 tests)
- Implement get_test.go (4 tests)
- Implement list_test.go (4 tests)
- Implement validate_test.go (4 tests)
- Implement root_test.go (4 tests)

### Phase 3: Integration Tests (Task 7)
- Implement integration_test.go (3 workflows)
- Test end-to-end scenarios
- Verify cross-command interactions

### Phase 4: Benchmarks (Task 8)
- Implement benchmark_test.go (6 benchmarks)
- Establish baseline metrics
- Verify performance targets met

### Phase 5: Validation (Task 9)
- Run all tests and verify passing
- Run benchmarks and validate performance
- Check coverage meets targets
- Fix any issues discovered

## Trade-offs

### Test-First vs Feature-First

**Chosen**: Test-First

**Pro**: Validates existing code, finds bugs early, establishes baseline
**Con**: Delays new features like shell completion

**Decision**: Test-first ensures quality foundation before adding complexity

### Test Organization

**Chosen**: One file per command

**Pro**: Easy to navigate, matches production structure
**Con**: Some duplication of setup code

**Decision**: Utilities package mitigates duplication

### Fixture Complexity

**Chosen**: Minimal test patterns in temp directories

**Pro**: Fast, isolated, repeatable
**Con**: Not testing with real patterns

**Decision**: Integration tests with real patterns complement unit tests

## Future Enhancements

1. **Shell Completion** - Add bash/zsh completion scripts
2. **Better Error Messages** - Enhance error formatting and suggestions
3. **Progress Indicators** - Add progress for long operations (validate --all)
4. **Config File Support** - Support ~/.tmkb/config.yaml for defaults
5. **Query History** - Cache recent queries for faster repeat access

## References

- Issue #5: CLI Completion and Polish
- Issue #3: Query Engine Improvements (relevance scoring)
- Issue #4: Output Format Polish (token limits, verbose mode)
- Cobra Documentation: https://github.com/spf13/cobra
- Go Testing Best Practices: https://go.dev/doc/tutorial/add-a-test
