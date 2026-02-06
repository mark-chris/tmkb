package knowledge

import (
	"encoding/json"
	"testing"
)

func TestQueryResult_AgentMode_JSON(t *testing.T) {
	result := QueryResult{
		PatternCount:      5,
		PatternsIncluded:  3,
		TokenCount:        456,
		TokenLimitReached: false,
		Patterns: []PatternOutput{
			{ID: "TMKB-001", Severity: "high", Threat: "test", Check: "test", Fix: "test"},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify JSON structure
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded["pattern_count"].(float64) != 5 {
		t.Errorf("Expected pattern_count=5, got %v", decoded["pattern_count"])
	}

	if decoded["patterns_included"].(float64) != 3 {
		t.Errorf("Expected patterns_included=3, got %v", decoded["patterns_included"])
	}
}

func TestQueryResult_VerboseMode_JSON(t *testing.T) {
	result := QueryResult{
		PatternCount:     2,
		PatternsIncluded: 2,
		VerbosePatterns: []PatternOutputVerbose{
			{
				PatternOutput: PatternOutput{
					ID:       "TMKB-001",
					Name:     "Test Pattern",
					Severity: "high",
					Threat:   "threat",
					Check:    "check",
					Fix:      "fix",
				},
				Description: "Full description",
			},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify verbose_patterns field exists
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if _, ok := decoded["verbose_patterns"]; !ok {
		t.Error("Expected verbose_patterns field in JSON")
	}
}
