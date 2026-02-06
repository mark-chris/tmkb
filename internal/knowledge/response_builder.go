package knowledge

import (
	"encoding/json"
	"log"
)

const TOKEN_LIMIT = 500

// buildAgentResponse builds a token-limited response for agent consumption
func buildAgentResponse(candidates []*ThreatPattern, limit int) QueryResult {
	counter, err := NewTokenCounter()
	if err != nil {
		log.Printf("Warning: Token counter initialization failed: %v, using approximation", err)
	}

	result := QueryResult{
		PatternCount: len(candidates),
		Patterns:     make([]PatternOutput, 0, limit),
	}

	totalTokens := 0
	patternsAdded := 0

	for i, p := range candidates {
		if i >= limit {
			break
		}

		output := PatternOutput{
			ID:       p.ID,
			Severity: p.Severity,
			Threat:   p.AgentSummary.Threat,
			Check:    p.AgentSummary.Check,
			Fix:      p.AgentSummary.Fix,
		}

		// Calculate tokens for this pattern
		patternJSON, _ := json.Marshal(output)
		patternTokens := counter.CountTokens(string(patternJSON))

		// Check if adding this pattern would exceed limit
		if patternsAdded > 0 && totalTokens+patternTokens > TOKEN_LIMIT {
			result.TokenLimitReached = true
			break
		}

		// Add pattern
		result.Patterns = append(result.Patterns, output)
		totalTokens += patternTokens
		patternsAdded++

		// If first pattern alone exceeds limit, mark it but continue
		if patternsAdded == 1 && totalTokens > TOKEN_LIMIT {
			result.TokenLimitReached = true
			break
		}
	}

	result.PatternsIncluded = len(result.Patterns)
	result.TokenCount = totalTokens

	// Add code pattern from most relevant match
	if len(candidates) > 0 {
		codePattern := extractCodePattern(candidates[0], "", "")
		if codePattern != nil {
			result.CodePattern = codePattern
		}
	}

	return result
}
