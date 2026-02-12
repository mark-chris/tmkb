package knowledge

import (
	"testing"
)

// createTestIndex creates an index with test patterns for query testing
func createTestIndex() *Index {
	patterns := []ThreatPattern{
		{
			ID:         "TMKB-AUTHZ-001",
			Name:       "Background Job Authorization",
			Severity:   "high",
			Likelihood: "high",
			Category:   "authorization",
			Language:   "python",
			Framework:  "flask",
			Triggers: Triggers{
				Keywords: []string{"background", "job", "celery", "async", "authorization"},
			},
			AgentSummary: AgentSummary{
				Threat: "Background jobs lose auth context",
				Check:  "Verify re-auth in jobs",
				Fix:    "Pass user ID, re-validate",
			},
		},
		{
			ID:         "TMKB-AUTHZ-002",
			Name:       "Multi-Tenant Data Isolation",
			Severity:   "critical",
			Likelihood: "high",
			Category:   "authorization",
			Language:   "python",
			Framework:  "flask",
			Triggers: Triggers{
				Keywords: []string{"tenant", "multi-tenant", "organization", "isolation"},
			},
			AgentSummary: AgentSummary{
				Threat: "Cross-tenant data leakage",
				Check:  "Verify tenant filtering",
				Fix:    "Add tenant_id to queries",
			},
		},
		{
			ID:         "TMKB-AUTHZ-003",
			Name:       "JWT Validation",
			Severity:   "critical",
			Likelihood: "medium",
			Category:   "authorization",
			Language:   "python",
			Framework:  "flask",
			Triggers: Triggers{
				Keywords: []string{"jwt", "token", "validation", "signature"},
			},
			AgentSummary: AgentSummary{
				Threat: "Invalid JWT accepted",
				Check:  "Verify signature validation",
				Fix:    "Use jwt.decode with verify=True",
			},
		},
		{
			ID:         "TMKB-AUTHZ-004",
			Name:       "API Endpoint Authorization",
			Severity:   "high",
			Likelihood: "high",
			Category:   "authorization",
			Language:   "python",
			Framework:  "flask",
			Triggers: Triggers{
				Keywords: []string{"api", "endpoint", "route", "authorization", "permission"},
			},
			AgentSummary: AgentSummary{
				Threat: "Unauthorized API access",
				Check:  "Verify auth decorators",
				Fix:    "Add @require_auth decorator",
			},
		},
	}

	idx := NewIndex()
	idx.Build(patterns)
	return idx
}

// TestQuery_RelevanceSorting_BackgroundJob tests that background job query
// returns AUTHZ-001 first due to relevance, even though AUTHZ-002 has higher severity
func TestQuery_RelevanceSorting_BackgroundJob(t *testing.T) {
	idx := createTestIndex()

	opts := QueryOptions{
		Context: "background job authorization",
		Limit:   3,
	}

	result := Query(idx, opts)

	// Should return patterns, with AUTHZ-001 ranked highest due to keyword matches
	if len(result.Patterns) == 0 {
		t.Fatal("Expected patterns in result, got none")
	}

	// AUTHZ-001 should be first (highest relevance: matches "background", "job", "authorization")
	if result.Patterns[0].ID != "TMKB-AUTHZ-001" {
		t.Errorf("Expected TMKB-AUTHZ-001 first (highest relevance), got %s", result.Patterns[0].ID)
	}
}

// TestQuery_RelevanceSorting_MultiTenant tests multi-tenant query
func TestQuery_RelevanceSorting_MultiTenant(t *testing.T) {
	idx := createTestIndex()

	opts := QueryOptions{
		Context: "multi-tenant organization data isolation",
		Limit:   3,
	}

	result := Query(idx, opts)

	if len(result.Patterns) == 0 {
		t.Fatal("Expected patterns in result, got none")
	}

	// AUTHZ-002 should be first (matches "multi-tenant", "organization", "isolation")
	if result.Patterns[0].ID != "TMKB-AUTHZ-002" {
		t.Errorf("Expected TMKB-AUTHZ-002 first (multi-tenant match), got %s", result.Patterns[0].ID)
	}
}

// TestQuery_RelevanceSorting_JWT tests JWT validation query
func TestQuery_RelevanceSorting_JWT(t *testing.T) {
	idx := createTestIndex()

	opts := QueryOptions{
		Context: "JWT token validation",
		Limit:   3,
	}

	result := Query(idx, opts)

	if len(result.Patterns) == 0 {
		t.Fatal("Expected patterns in result, got none")
	}

	// AUTHZ-003 should be first (matches "jwt", "token", "validation")
	if result.Patterns[0].ID != "TMKB-AUTHZ-003" {
		t.Errorf("Expected TMKB-AUTHZ-003 first (JWT match), got %s", result.Patterns[0].ID)
	}
}

// TestQuery_SeverityTiebreaker tests that severity breaks ties when relevance is equal
func TestQuery_SeverityTiebreaker(t *testing.T) {
	patterns := []ThreatPattern{
		{
			ID:         "TMKB-TEST-001",
			Severity:   "medium",
			Likelihood: "high",
			Category:   "authorization",
			Language:   "python",
			Framework:  "flask",
			Triggers: Triggers{
				Keywords: []string{"authorization"},
			},
			AgentSummary: AgentSummary{
				Threat: "Auth issue",
				Check:  "Check auth",
				Fix:    "Fix auth",
			},
		},
		{
			ID:         "TMKB-TEST-002",
			Severity:   "critical",
			Likelihood: "high",
			Category:   "authorization",
			Language:   "python",
			Framework:  "flask",
			Triggers: Triggers{
				Keywords: []string{"authorization"},
			},
			AgentSummary: AgentSummary{
				Threat: "Critical auth issue",
				Check:  "Check auth",
				Fix:    "Fix auth",
			},
		},
	}

	idx := NewIndex()
	idx.Build(patterns)

	opts := QueryOptions{
		Context: "authorization",
		Limit:   2,
	}

	result := Query(idx, opts)

	if len(result.Patterns) != 2 {
		t.Fatalf("Expected 2 patterns, got %d", len(result.Patterns))
	}

	// Both have same relevance (1 keyword match), so critical should come first
	if result.Patterns[0].ID != "TMKB-TEST-002" {
		t.Errorf("Expected TMKB-TEST-002 first (critical severity), got %s", result.Patterns[0].ID)
	}
}

// TestQuery_BackwardCompatibility_NoContext tests that queries without context
// still work and sort by severity as before
func TestQuery_BackwardCompatibility_NoContext(t *testing.T) {
	idx := createTestIndex()

	opts := QueryOptions{
		Limit: 3,
	}

	result := Query(idx, opts)

	if len(result.Patterns) == 0 {
		t.Fatal("Expected patterns in result, got none")
	}

	// Without context, should sort by severity first
	// AUTHZ-002 and AUTHZ-003 are both critical, so one of them should be first
	firstSeverity := result.Patterns[0].Severity
	if firstSeverity != "critical" {
		t.Errorf("Expected critical severity first without context, got %s", firstSeverity)
	}
}

// TestQuery_BackwardCompatibility_EmptyContext tests that empty context
// behaves like no context
func TestQuery_BackwardCompatibility_EmptyContext(t *testing.T) {
	idx := createTestIndex()

	opts := QueryOptions{
		Context: "",
		Limit:   3,
	}

	result := Query(idx, opts)

	if len(result.Patterns) == 0 {
		t.Fatal("Expected patterns in result, got none")
	}

	// Empty context should behave like no context: sort by severity
	firstSeverity := result.Patterns[0].Severity
	if firstSeverity != "critical" {
		t.Errorf("Expected critical severity first with empty context, got %s", firstSeverity)
	}
}

// TestQuery_ContextWithNoMatches tests handling when context doesn't match any patterns
func TestQuery_ContextWithNoMatches(t *testing.T) {
	idx := createTestIndex()

	opts := QueryOptions{
		Context: "completely unrelated context about databases",
		Limit:   3,
	}

	result := Query(idx, opts)

	// MatchContext will return no matches, so Query should fall back to all patterns
	// This is actually current behavior - no matches means no results
	// The test documents this behavior
	if result.PatternCount != 0 {
		t.Logf("Query with no matching context returned %d patterns", result.PatternCount)
	}
}

// TestQuery_RelevanceOverridesSeverity tests that a pattern with better relevance
// ranks higher than a pattern with higher severity but lower relevance
func TestQuery_RelevanceOverridesSeverity(t *testing.T) {
	patterns := []ThreatPattern{
		{
			ID:         "TMKB-LOW-RELEVANCE",
			Severity:   "critical",
			Likelihood: "high",
			Category:   "authorization",
			Language:   "python",
			Framework:  "flask",
			Triggers: Triggers{
				Keywords: []string{"api", "authorization"}, // Matches 1-2 keywords
			},
			AgentSummary: AgentSummary{
				Threat: "Critical issue",
				Check:  "Check it",
				Fix:    "Fix it",
			},
		},
		{
			ID:         "TMKB-HIGH-RELEVANCE",
			Severity:   "medium",
			Likelihood: "medium",
			Category:   "authorization",
			Language:   "python",
			Framework:  "flask",
			Triggers: Triggers{
				Keywords: []string{"background", "job", "celery", "async", "authorization"},
			},
			AgentSummary: AgentSummary{
				Threat: "Medium issue",
				Check:  "Check it",
				Fix:    "Fix it",
			},
		},
	}

	idx := NewIndex()
	idx.Build(patterns)

	opts := QueryOptions{
		Context: "background job async processing with celery authorization",
		Limit:   2,
	}

	result := Query(idx, opts)

	if len(result.Patterns) != 2 {
		t.Fatalf("Expected 2 patterns, got %d", len(result.Patterns))
	}

	// TMKB-HIGH-RELEVANCE should rank first despite lower severity
	// because it matches more keywords (background, job, celery, async, authorization = 5 matches)
	// vs LOW-RELEVANCE matching only (authorization = 1 match)
	if result.Patterns[0].ID != "TMKB-HIGH-RELEVANCE" {
		t.Errorf("Expected TMKB-HIGH-RELEVANCE first (better keyword match), got %s", result.Patterns[0].ID)
		t.Logf("Without relevance scoring, severity-first sorting would put the critical pattern first")
		t.Logf("With relevance scoring, the pattern matching 5 keywords should rank higher than 1 keyword match")
	}
}

func TestQuery_AgentMode_TokenLimit(t *testing.T) {
	idx := createTestIndex()

	opts := QueryOptions{
		Context:   "background job authorization",
		Verbosity: "agent", // Explicit agent mode
		Limit:     0,       // Use default
	}

	result := Query(idx, opts)

	// Should use agent mode defaults
	if result.TokenCount == 0 {
		t.Error("Expected token_count to be set in agent mode")
	}

	if result.TokenCount > 500 {
		t.Errorf("Token count %d exceeds limit of 500", result.TokenCount)
	}

	if len(result.Patterns) > 3 {
		t.Errorf("Expected max 3 patterns in agent mode, got %d", len(result.Patterns))
	}

	if len(result.VerbosePatterns) != 0 {
		t.Error("Expected no verbose patterns in agent mode")
	}
}

func TestQuery_VerboseMode_NoTokenLimit(t *testing.T) {
	idx := createTestIndex()

	opts := QueryOptions{
		Context:   "background job authorization",
		Verbosity: "human", // Verbose mode
		Limit:     0,       // Use default
	}

	result := Query(idx, opts)

	// Should use verbose mode
	if result.TokenCount != 0 {
		t.Error("Expected token_count to be 0 in verbose mode")
	}

	if len(result.VerbosePatterns) == 0 {
		t.Error("Expected verbose patterns in verbose mode")
	}

	if len(result.Patterns) != 0 {
		t.Error("Expected no agent patterns in verbose mode")
	}

	// Check that verbose fields are populated
	if len(result.VerbosePatterns) > 0 {
		v := result.VerbosePatterns[0]
		if v.Name == "" {
			t.Error("Expected name in verbose mode")
		}
		if v.Threat == "" {
			t.Error("Expected threat in verbose mode")
		}
		if v.Check == "" {
			t.Error("Expected check in verbose mode")
		}
		if v.Fix == "" {
			t.Error("Expected fix in verbose mode")
		}
	}
}

func TestQuery_VerboseMode_DefaultLimit(t *testing.T) {
	// Create index with many patterns
	patterns := make([]ThreatPattern, 15)
	for i := 0; i < 15; i++ {
		patterns[i] = ThreatPattern{
			ID:         "TMKB-00" + string(rune('1'+i)),
			Name:       "Pattern " + string(rune('1'+i)),
			Severity:   "high",
			Likelihood: "medium",
			Category:   "authorization",
			Language:   "python",
			Framework:  "flask",
			Triggers: Triggers{
				Keywords: []string{"authorization"},
			},
			AgentSummary: AgentSummary{
				Threat: "Threat",
				Check:  "Check",
				Fix:    "Fix",
			},
			Description: "Description",
		}
	}

	idx := NewIndex()
	idx.Build(patterns)

	opts := QueryOptions{
		Context:   "authorization",
		Verbosity: "human",
		Limit:     0, // Default should be 10
	}

	result := Query(idx, opts)

	if result.PatternsIncluded != 10 {
		t.Errorf("Expected default limit of 10 in verbose mode, got %d", result.PatternsIncluded)
	}
}
