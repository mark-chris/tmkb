package cli

import (
	"os"
	"testing"

	"github.com/mark-chris/tmkb/internal/cli/testutil"
	"github.com/mark-chris/tmkb/internal/knowledge"
)

// BenchmarkQuery_ColdStart measures first query performance with index building.
// This simulates the cold start scenario where the index needs to be built from scratch.
func BenchmarkQuery_ColdStart(b *testing.B) {
	// Setup test fixtures once
	fixture := testutil.SetupTestPatterns(&testing.T{})
	defer fixture.Cleanup()

	// Capture stdout to prevent output pollution
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatalf("Failed to open devnull: %v", err)
	}
	defer devNull.Close()
	oldStdout := os.Stdout
	os.Stdout = devNull
	defer func() { os.Stdout = oldStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Reset state to simulate cold start
		resetQueryFlags()
		patternsDir = fixture.Dir
		loader = knowledge.NewLoader(patternsDir)
		index = knowledge.NewIndex()
		queryContext = "background job authorization"
		b.StartTimer()

		// Load patterns and build index (cold start)
		patterns, err := loader.LoadAll()
		if err != nil {
			b.Fatalf("Failed to load patterns: %v", err)
		}
		index.Build(patterns)

		// Execute query
		err = runQuery(queryCmd, []string{})
		if err != nil {
			b.Fatalf("Query failed: %v", err)
		}
	}
}

// BenchmarkQuery_WarmIndex measures query performance with pre-built index.
// This represents the typical case where the index is already in memory.
// Target: <100ms per operation
func BenchmarkQuery_WarmIndex(b *testing.B) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(&testing.T{})
	defer fixture.Cleanup()

	// Capture stdout to prevent output pollution
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatalf("Failed to open devnull: %v", err)
	}
	defer devNull.Close()
	oldStdout := os.Stdout
	os.Stdout = devNull
	defer func() { os.Stdout = oldStdout }()

	// Pre-build index once
	resetQueryFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		b.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Set query parameters
	queryContext = "background job authorization"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = runQuery(queryCmd, []string{})
		if err != nil {
			b.Fatalf("Query failed: %v", err)
		}
	}
}

// BenchmarkGet_ByID measures pattern retrieval by ID performance.
// This tests the performance of looking up a single pattern.
// Target: <50ms per operation
func BenchmarkGet_ByID(b *testing.B) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(&testing.T{})
	defer fixture.Cleanup()

	// Capture stdout to prevent output pollution
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatalf("Failed to open devnull: %v", err)
	}
	defer devNull.Close()
	oldStdout := os.Stdout
	os.Stdout = devNull
	defer func() { os.Stdout = oldStdout }()

	// Pre-build index once
	resetGetFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		b.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = runGet(getCmd, []string{"TMKB-TEST-001"})
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}

// BenchmarkList_All measures performance of listing all patterns.
// This tests the performance of retrieving all patterns from the index.
// Target: <50ms per operation
func BenchmarkList_All(b *testing.B) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(&testing.T{})
	defer fixture.Cleanup()

	// Capture stdout to prevent output pollution
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatalf("Failed to open devnull: %v", err)
	}
	defer devNull.Close()
	oldStdout := os.Stdout
	os.Stdout = devNull
	defer func() { os.Stdout = oldStdout }()

	// Pre-build index once
	resetQueryFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		b.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = runList(listCmd, []string{})
		if err != nil {
			b.Fatalf("List failed: %v", err)
		}
	}
}

// BenchmarkValidate_All measures performance of validating all patterns.
// This tests the performance of running validation checks on all patterns.
// Target: <500ms per operation
func BenchmarkValidate_All(b *testing.B) {
	// Setup test fixtures
	fixture := testutil.SetupTestPatterns(&testing.T{})
	defer fixture.Cleanup()

	// Capture stdout to prevent output pollution
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatalf("Failed to open devnull: %v", err)
	}
	defer devNull.Close()
	oldStdout := os.Stdout
	os.Stdout = devNull
	defer func() { os.Stdout = oldStdout }()

	// Pre-build index once
	resetQueryFlags()
	patternsDir = fixture.Dir
	loader = knowledge.NewLoader(patternsDir)
	index = knowledge.NewIndex()
	patterns, err := loader.LoadAll()
	if err != nil {
		b.Fatalf("Failed to load patterns: %v", err)
	}
	index.Build(patterns)

	// Set validate all flag
	validateAll = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = runValidate(validateCmd, []string{})
		if err != nil {
			b.Fatalf("Validate failed: %v", err)
		}
	}
}
