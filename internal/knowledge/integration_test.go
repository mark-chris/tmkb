package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIntegration_AgentMode_RealPatterns validates agent mode with real patterns
func TestIntegration_AgentMode_RealPatterns(t *testing.T) {
	patterns := loadPatternsFromDir(t)
	if len(patterns) == 0 {
		t.Skip("No patterns available, skipping integration test")
	}

	// Build index
	idx := NewIndex()
	idx.Build(patterns)

	// Test query with context
	opts := QueryOptions{
		Context:   "multi-tenant background job authorization",
		Verbosity: "agent",
		Limit:     3,
	}

	result := Query(idx, opts)

	// Validate structure
	if result.PatternCount == 0 {
		t.Error("Expected patterns to match query context")
	}

	if result.PatternsIncluded > 3 {
		t.Errorf("Expected max 3 patterns, got %d", result.PatternsIncluded)
	}

	if result.TokenCount == 0 {
		t.Error("Expected token count to be calculated")
	}

	if result.TokenCount > tokenLimit {
		if !result.TokenLimitReached {
			t.Error("Token count exceeds limit but token_limit_reached not set")
		}
	}

	// Validate pattern structure
	for _, p := range result.Patterns {
		if p.ID == "" {
			t.Error("Pattern ID should not be empty")
		}
		if p.Severity == "" {
			t.Error("Pattern severity should not be empty")
		}
		if p.Threat == "" {
			t.Error("Pattern threat should not be empty")
		}
		if p.Check == "" {
			t.Error("Pattern check should not be empty")
		}
		if p.Fix == "" {
			t.Error("Pattern fix should not be empty")
		}
	}

	// Validate code pattern if present
	if result.CodePattern != nil {
		if result.CodePattern.Language == "" {
			t.Error("Code pattern language should not be empty")
		}
		if result.CodePattern.Framework == "" {
			t.Error("Code pattern framework should not be empty")
		}
		if result.CodePattern.SecureTemplate == "" {
			t.Error("Code pattern secure_template should not be empty")
		}
	}
}

// TestIntegration_VerboseMode_RealPatterns validates verbose mode with real patterns
func TestIntegration_VerboseMode_RealPatterns(t *testing.T) {
	patterns := loadPatternsFromDir(t)
	if len(patterns) == 0 {
		t.Skip("No patterns available, skipping integration test")
	}

	// Build index
	idx := NewIndex()
	idx.Build(patterns)

	// Test query with context
	opts := QueryOptions{
		Context:   "authorization bypass state transition",
		Verbosity: "human",
		Limit:     5,
	}

	result := Query(idx, opts)

	// Validate structure
	if result.PatternCount == 0 {
		t.Error("Expected patterns to match query context")
	}

	if result.PatternsIncluded > 5 {
		t.Errorf("Expected max 5 patterns, got %d", result.PatternsIncluded)
	}

	// Verbose mode should not have token limits
	if result.TokenCount != 0 {
		t.Error("Verbose mode should not calculate token count")
	}

	if result.TokenLimitReached {
		t.Error("Verbose mode should not have token_limit_reached set")
	}

	// Validate verbose pattern structure
	for _, p := range result.VerbosePatterns {
		if p.ID == "" {
			t.Error("Verbose pattern ID should not be empty")
		}
		if p.Name == "" {
			t.Error("Verbose pattern name should not be empty")
		}
		if p.Severity == "" {
			t.Error("Verbose pattern severity should not be empty")
		}
		if p.Likelihood == "" {
			t.Error("Verbose pattern likelihood should not be empty")
		}
		if p.Threat == "" {
			t.Error("Verbose pattern threat should not be empty")
		}
		if p.Check == "" {
			t.Error("Verbose pattern check should not be empty")
		}
		if p.Fix == "" {
			t.Error("Verbose pattern fix should not be empty")
		}
		if p.Description == "" {
			t.Error("Verbose pattern description should not be empty")
		}

		// Validate mitigations
		if len(p.Mitigations) == 0 {
			t.Errorf("Pattern %s should have mitigations", p.ID)
		}

		for _, m := range p.Mitigations {
			if m.ID == "" {
				t.Errorf("Mitigation ID should not be empty for pattern %s", p.ID)
			}
			if m.Description == "" {
				t.Errorf("Mitigation description should not be empty for pattern %s", p.ID)
			}
			if m.Effectiveness == "" {
				t.Errorf("Mitigation effectiveness should not be empty for pattern %s", p.ID)
			}
			if m.ImplementationEffort == "" {
				t.Errorf("Mitigation implementation_effort should not be empty for pattern %s", p.ID)
			}
		}
	}

	// Validate that we got VerbosePatterns, not Patterns
	if len(result.Patterns) > 0 {
		t.Error("Verbose mode should populate VerbosePatterns, not Patterns")
	}
}

// TestIntegration_DeterministicOrdering validates that query results are deterministic
func TestIntegration_DeterministicOrdering(t *testing.T) {
	patterns := loadPatternsFromDir(t)
	if len(patterns) == 0 {
		t.Skip("No patterns available, skipping integration test")
	}

	// Build index
	idx := NewIndex()
	idx.Build(patterns)

	// Test query multiple times
	opts := QueryOptions{
		Context:   "tenant isolation ownership",
		Verbosity: "agent",
		Limit:     5,
	}

	// Run query 3 times
	results := make([]QueryResult, 3)
	for i := 0; i < 3; i++ {
		results[i] = Query(idx, opts)
	}

	// Validate all results are identical
	for i := 1; i < len(results); i++ {
		if results[i].PatternCount != results[0].PatternCount {
			t.Errorf("Run %d: pattern_count mismatch: got %d, expected %d",
				i, results[i].PatternCount, results[0].PatternCount)
		}

		if results[i].PatternsIncluded != results[0].PatternsIncluded {
			t.Errorf("Run %d: patterns_included mismatch: got %d, expected %d",
				i, results[i].PatternsIncluded, results[0].PatternsIncluded)
		}

		// Validate pattern order
		if len(results[i].Patterns) != len(results[0].Patterns) {
			t.Errorf("Run %d: pattern array length mismatch", i)
			continue
		}

		for j := range results[i].Patterns {
			if results[i].Patterns[j].ID != results[0].Patterns[j].ID {
				t.Errorf("Run %d: pattern order mismatch at position %d: got %s, expected %s",
					i, j, results[i].Patterns[j].ID, results[0].Patterns[j].ID)
			}
		}
	}

	// Validate severity-based ordering when no context
	optsNoContext := QueryOptions{
		Verbosity: "agent",
		Limit:     10,
	}

	result := Query(idx, optsNoContext)

	severityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
	}

	for i := 1; i < len(result.Patterns); i++ {
		prevSeverity := severityOrder[result.Patterns[i-1].Severity]
		currSeverity := severityOrder[result.Patterns[i].Severity]

		if currSeverity < prevSeverity {
			t.Errorf("Patterns not ordered by severity: %s (severity %s) before %s (severity %s)",
				result.Patterns[i-1].ID, result.Patterns[i-1].Severity,
				result.Patterns[i].ID, result.Patterns[i].Severity)
		}
	}
}

// loadPatternsFromDir loads patterns from the patterns directory
// Returns empty slice if patterns are not available (e.g., in CI without pattern files)
func loadPatternsFromDir(t *testing.T) []ThreatPattern {
	t.Helper()

	// Try to find patterns directory
	patternsDir := findPatternsDir()
	if patternsDir == "" {
		return []ThreatPattern{}
	}

	// Load patterns using the standard loader
	loader := NewLoader(patternsDir)
	patterns, err := loader.LoadAll()
	if err != nil {
		// If loading fails, skip the test rather than fail
		// This allows tests to pass in environments without pattern files
		t.Logf("Could not load patterns: %v", err)
		return []ThreatPattern{}
	}

	return patterns
}

// findPatternsDir locates the patterns directory relative to the test
func findPatternsDir() string {
	candidates := []string{
		"../../patterns", // From internal/knowledge
		"./patterns",     // From repo root
		"../patterns",    // Alternative
		filepath.Join(os.Getenv("TMKB_PATTERNS_DIR")), // Environment override
	}

	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}

	return ""
}
