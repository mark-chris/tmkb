package knowledge

import (
	"testing"
)

func TestBuildAgentResponse_UnderLimit(t *testing.T) {
	candidates := []*ThreatPattern{
		{
			ID:       "TMKB-001",
			Severity: "high",
			AgentSummary: AgentSummary{
				Threat: "Short threat",
				Check:  "Short check",
				Fix:    "Short fix",
			},
		},
		{
			ID:       "TMKB-002",
			Severity: "medium",
			AgentSummary: AgentSummary{
				Threat: "Another threat",
				Check:  "Another check",
				Fix:    "Another fix",
			},
		},
	}

	result := buildAgentResponse(candidates, 3)

	if result.PatternCount != 2 {
		t.Errorf("Expected pattern_count=2, got %d", result.PatternCount)
	}

	if result.PatternsIncluded != 2 {
		t.Errorf("Expected patterns_included=2, got %d", result.PatternsIncluded)
	}

	if result.TokenCount == 0 {
		t.Error("Expected token_count > 0")
	}

	if result.TokenCount > 500 {
		t.Errorf("Token count %d exceeds limit of 500", result.TokenCount)
	}

	if result.TokenLimitReached {
		t.Error("Expected token_limit_reached=false")
	}

	if len(result.Patterns) != 2 {
		t.Errorf("Expected 2 patterns, got %d", len(result.Patterns))
	}
}

func TestBuildAgentResponse_ExceedsLimit(t *testing.T) {
	// Create patterns with moderate length that will exceed 500 tokens when combined
	// Each pattern should be ~125 tokens, so 4 patterns = 500 tokens, 5 patterns > 500 limit
	longText := ""
	for i := 0; i < 12; i++ {
		longText += "This is a moderately long sentence with several words that will consume tokens. "
	}

	candidates := make([]*ThreatPattern, 5)
	for i := 0; i < 5; i++ {
		candidates[i] = &ThreatPattern{
			ID:       "TMKB-00" + string(rune('1'+i)),
			Severity: "high",
			AgentSummary: AgentSummary{
				Threat: longText,
				Check:  longText,
				Fix:    longText,
			},
		}
	}

	result := buildAgentResponse(candidates, 5)

	if result.PatternCount != 5 {
		t.Errorf("Expected pattern_count=5, got %d", result.PatternCount)
	}

	// Should have fewer patterns included due to token limit
	if result.PatternsIncluded >= 5 {
		t.Errorf("Expected patterns_included < 5, got %d (token_count=%d)", result.PatternsIncluded, result.TokenCount)
	}

	if !result.TokenLimitReached {
		t.Errorf("Expected token_limit_reached=true (patterns_included=%d, token_count=%d)", result.PatternsIncluded, result.TokenCount)
	}

	// Token count might slightly exceed 500 if first pattern is large
	// but we should be reasonably close
	if result.TokenCount > 550 {
		t.Errorf("Token count %d exceeds acceptable range (500 limit + tolerance)", result.TokenCount)
	}

	if len(result.Patterns) != result.PatternsIncluded {
		t.Errorf("Mismatch: patterns_included=%d but len(patterns)=%d",
			result.PatternsIncluded, len(result.Patterns))
	}
}

func TestBuildAgentResponse_SinglePatternTooLarge(t *testing.T) {
	// Create one pattern that's extremely large
	largeText := ""
	for i := 0; i < 200; i++ {
		largeText += "This is a very long sentence with many words. "
	}

	candidates := []*ThreatPattern{
		{
			ID:       "TMKB-001",
			Severity: "critical",
			AgentSummary: AgentSummary{
				Threat: largeText,
				Check:  largeText,
				Fix:    largeText,
			},
		},
	}

	result := buildAgentResponse(candidates, 3)

	// Should include at least one pattern even if too large
	if result.PatternsIncluded < 1 {
		t.Error("Expected at least 1 pattern included even if over limit")
	}

	if result.TokenLimitReached {
		t.Log("Token limit reached (expected for oversized pattern)")
	}
}

func TestBuildAgentResponse_TokenCounting(t *testing.T) {
	candidates := []*ThreatPattern{
		{
			ID:       "TMKB-001",
			Severity: "high",
			AgentSummary: AgentSummary{
				Threat: "Test",
				Check:  "Test",
				Fix:    "Test",
			},
		},
	}

	result := buildAgentResponse(candidates, 1)

	// Verify token count is reasonable
	if result.TokenCount < 10 {
		t.Errorf("Token count %d seems too low", result.TokenCount)
	}

	if result.TokenCount > 100 {
		t.Errorf("Token count %d seems too high for minimal pattern", result.TokenCount)
	}
}
