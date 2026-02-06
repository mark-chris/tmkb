package knowledge

import (
	"strings"
)

// ExtractKeywords extracts 1-grams, 2-grams, and 3-grams from the input string.
// Returns a deduplicated list of case-insensitive keywords.
func ExtractKeywords(input string) []string {
	// Trim and handle empty input
	input = strings.TrimSpace(input)
	if input == "" {
		return []string{}
	}

	// Split into words
	words := strings.Fields(input)
	if len(words) == 0 {
		return []string{}
	}

	// Use map for deduplication
	keywordMap := make(map[string]bool)

	// Extract N-grams (1, 2, and 3)
	for n := 1; n <= 3 && n <= len(words); n++ {
		for i := 0; i <= len(words)-n; i++ {
			// Take n words starting at position i
			ngram := strings.Join(words[i:i+n], " ")
			// Store lowercase version
			keywordMap[strings.ToLower(ngram)] = true
		}
	}

	// Convert map to slice
	keywords := make([]string, 0, len(keywordMap))
	for keyword := range keywordMap {
		keywords = append(keywords, keyword)
	}

	return keywords
}

// CalculateRelevance calculates the relevance score for a pattern based on keyword matches.
// Uses the hybrid formula: (matched_keywords × 2) + (matched_keywords / total_pattern_keywords)
// Returns 0.0 if no matches or invalid input.
func CalculateRelevance(queryKeywords, patternKeywords []string) float64 {
	// Handle empty inputs
	if len(queryKeywords) == 0 || len(patternKeywords) == 0 {
		return 0.0
	}

	// Build a map of pattern keywords (case-insensitive)
	patternMap := make(map[string]bool)
	for _, keyword := range patternKeywords {
		patternMap[strings.ToLower(keyword)] = true
	}

	// Count matches
	matchCount := 0
	for _, queryKeyword := range queryKeywords {
		if patternMap[strings.ToLower(queryKeyword)] {
			matchCount++
		}
	}

	// No matches
	if matchCount == 0 {
		return 0.0
	}

	// Calculate hybrid score: (matched × 2) + (matched / total)
	matchWeight := float64(matchCount) * 2.0
	coverageRatio := float64(matchCount) / float64(len(patternKeywords))

	return matchWeight + coverageRatio
}
