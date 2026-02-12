package knowledge

import (
	"reflect"
	"testing"
)

// TestExtractKeywords_SingleWord tests extraction of 1-grams from a single word
func TestExtractKeywords_SingleWord(t *testing.T) {
	input := "background"
	expected := []string{"background"}

	result := ExtractKeywords(input)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ExtractKeywords(%q) = %v, want %v", input, result, expected)
	}
}

// TestExtractKeywords_TwoWords tests extraction of 1-grams and 2-grams
func TestExtractKeywords_TwoWords(t *testing.T) {
	input := "background job"
	expected := []string{"background", "job", "background job"}

	result := ExtractKeywords(input)

	if !equalIgnoreOrder(result, expected) {
		t.Errorf("ExtractKeywords(%q) = %v, want %v (any order)", input, result, expected)
	}
}

// TestExtractKeywords_ThreeWords tests extraction of 1-grams, 2-grams, and 3-grams
func TestExtractKeywords_ThreeWords(t *testing.T) {
	input := "multi tenant API"
	expected := []string{
		"multi", "tenant", "api",
		"multi tenant", "tenant api",
		"multi tenant api",
	}

	result := ExtractKeywords(input)

	if !equalIgnoreOrder(result, expected) {
		t.Errorf("ExtractKeywords(%q) = %v, want %v (any order)", input, result, expected)
	}
}

// TestExtractKeywords_CaseInsensitive tests that keywords are lowercased
func TestExtractKeywords_CaseInsensitive(t *testing.T) {
	input := "JWT Validation"
	expected := []string{"jwt", "validation", "jwt validation"}

	result := ExtractKeywords(input)

	if !equalIgnoreOrder(result, expected) {
		t.Errorf("ExtractKeywords(%q) = %v, want %v", input, result, expected)
	}
}

// TestExtractKeywords_Deduplication tests that duplicate keywords are removed
func TestExtractKeywords_Deduplication(t *testing.T) {
	input := "API API endpoint"
	expected := []string{
		"api", "endpoint",
		"api api", "api endpoint",
		"api api endpoint",
	}

	result := ExtractKeywords(input)

	if !equalIgnoreOrder(result, expected) {
		t.Errorf("ExtractKeywords(%q) = %v, want %v", input, result, expected)
	}
}

// TestExtractKeywords_EmptyString tests handling of empty input
func TestExtractKeywords_EmptyString(t *testing.T) {
	input := ""
	expected := []string{}

	result := ExtractKeywords(input)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ExtractKeywords(%q) = %v, want %v", input, result, expected)
	}
}

// TestExtractKeywords_Whitespace tests handling of whitespace-only input
func TestExtractKeywords_Whitespace(t *testing.T) {
	input := "   \t\n  "
	expected := []string{}

	result := ExtractKeywords(input)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ExtractKeywords(%q) = %v, want %v", input, result, expected)
	}
}

// TestExtractKeywords_LongPhrase tests N-gram limits with longer phrases
func TestExtractKeywords_LongPhrase(t *testing.T) {
	input := "multi tenant API background job authorization"

	// Should extract 1-grams, 2-grams, and 3-grams only
	// Not 4-grams or higher
	mustInclude := []string{
		"multi", "tenant", "api", "background", "job", "authorization",
		"multi tenant", "tenant api", "api background", "background job", "job authorization",
		"multi tenant api", "tenant api background", "api background job", "background job authorization",
	}

	mustNotInclude := []string{
		"multi tenant api background",                   // 4-gram
		"multi tenant api background job authorization", // 6-gram
	}

	result := ExtractKeywords(input)

	// Check all required keywords are present
	for _, keyword := range mustInclude {
		if !contains(result, keyword) {
			t.Errorf("ExtractKeywords(%q) missing expected keyword %q", input, keyword)
		}
	}

	// Check no 4-grams or higher are present
	for _, keyword := range mustNotInclude {
		if contains(result, keyword) {
			t.Errorf("ExtractKeywords(%q) should not include %q (exceeds 3-gram limit)", input, keyword)
		}
	}
}

// Helper function to check if two slices contain the same elements (order-independent)
func equalIgnoreOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aMap := make(map[string]bool)
	for _, val := range a {
		aMap[val] = true
	}

	for _, val := range b {
		if !aMap[val] {
			return false
		}
	}

	return true
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TestCalculateRelevance_NoMatches tests scoring when no keywords match
func TestCalculateRelevance_NoMatches(t *testing.T) {
	queryKeywords := []string{"jwt", "validation"}
	patternKeywords := []string{"background", "job", "authorization"}

	score := CalculateRelevance(queryKeywords, patternKeywords)

	if score != 0.0 {
		t.Errorf("CalculateRelevance with no matches = %v, want 0.0", score)
	}
}

// TestCalculateRelevance_SingleMatch tests scoring with one matching keyword
func TestCalculateRelevance_SingleMatch(t *testing.T) {
	queryKeywords := []string{"background", "job"}
	patternKeywords := []string{"authorization", "background", "tenant"}

	// 1 match: (1 × 2) + (1/3) = 2.333...
	expected := 2.0 + (1.0 / 3.0)

	score := CalculateRelevance(queryKeywords, patternKeywords)

	if !floatEqual(score, expected) {
		t.Errorf("CalculateRelevance with 1 match = %v, want %v", score, expected)
	}
}

// TestCalculateRelevance_MultipleMatches tests scoring with multiple matching keywords
func TestCalculateRelevance_MultipleMatches(t *testing.T) {
	queryKeywords := []string{"background", "job", "authorization"}
	patternKeywords := []string{"background", "job", "tenant", "api"}

	// 2 matches: (2 × 2) + (2/4) = 4.5
	expected := 4.0 + 0.5

	score := CalculateRelevance(queryKeywords, patternKeywords)

	if !floatEqual(score, expected) {
		t.Errorf("CalculateRelevance with 2 matches = %v, want %v", score, expected)
	}
}

// TestCalculateRelevance_AllMatch tests scoring when all pattern keywords match
func TestCalculateRelevance_AllMatch(t *testing.T) {
	queryKeywords := []string{"background", "job", "api", "tenant", "authorization"}
	patternKeywords := []string{"background", "job", "api"}

	// 3 matches: (3 × 2) + (3/3) = 7.0
	expected := 7.0

	score := CalculateRelevance(queryKeywords, patternKeywords)

	if !floatEqual(score, expected) {
		t.Errorf("CalculateRelevance with all matches = %v, want %v", score, expected)
	}
}

// TestCalculateRelevance_CaseInsensitiveMatching tests that matching is case-insensitive
func TestCalculateRelevance_CaseInsensitiveMatching(t *testing.T) {
	queryKeywords := []string{"jwt", "validation"}
	patternKeywords := []string{"JWT", "Authorization"}

	// 1 match: (1 × 2) + (1/2) = 2.5
	expected := 2.5

	score := CalculateRelevance(queryKeywords, patternKeywords)

	if !floatEqual(score, expected) {
		t.Errorf("CalculateRelevance with case mismatch = %v, want %v", score, expected)
	}
}

// TestCalculateRelevance_EmptyQuery tests handling of empty query keywords
func TestCalculateRelevance_EmptyQuery(t *testing.T) {
	queryKeywords := []string{}
	patternKeywords := []string{"background", "job"}

	score := CalculateRelevance(queryKeywords, patternKeywords)

	if score != 0.0 {
		t.Errorf("CalculateRelevance with empty query = %v, want 0.0", score)
	}
}

// TestCalculateRelevance_EmptyPattern tests handling of empty pattern keywords
func TestCalculateRelevance_EmptyPattern(t *testing.T) {
	queryKeywords := []string{"background", "job"}
	patternKeywords := []string{}

	score := CalculateRelevance(queryKeywords, patternKeywords)

	if score != 0.0 {
		t.Errorf("CalculateRelevance with empty pattern = %v, want 0.0", score)
	}
}

// TestCalculateRelevance_HybridFormulaExample tests the example from the design doc
// Pattern C: 4 matches out of 20 keywords should score higher than
// Pattern A: 3 matches out of 10 keywords
func TestCalculateRelevance_HybridFormulaExample(t *testing.T) {
	queryKeywords := []string{"multi", "tenant", "api", "background", "job"}

	// Pattern A: 3 matches out of 10 keywords
	patternA := []string{"api", "background", "job", "k4", "k5", "k6", "k7", "k8", "k9", "k10"}
	scoreA := CalculateRelevance(queryKeywords, patternA)
	// (3 × 2) + (3/10) = 6.3
	expectedA := 6.0 + 0.3

	// Pattern C: 4 matches out of 20 keywords
	patternC := []string{
		"multi", "tenant", "api", "background",
		"k5", "k6", "k7", "k8", "k9", "k10",
		"k11", "k12", "k13", "k14", "k15", "k16", "k17", "k18", "k19", "k20",
	}
	scoreC := CalculateRelevance(queryKeywords, patternC)
	// (4 × 2) + (4/20) = 8.2
	expectedC := 8.0 + 0.2

	if !floatEqual(scoreA, expectedA) {
		t.Errorf("Pattern A score = %v, want %v", scoreA, expectedA)
	}

	if !floatEqual(scoreC, expectedC) {
		t.Errorf("Pattern C score = %v, want %v", scoreC, expectedC)
	}

	// Pattern C should rank higher than Pattern A
	if scoreC <= scoreA {
		t.Errorf("Pattern C score (%v) should be > Pattern A score (%v)", scoreC, scoreA)
	}
}

// Helper function to compare floats with tolerance
func floatEqual(a, b float64) bool {
	const epsilon = 0.0001
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}
