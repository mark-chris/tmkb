package knowledge

import (
	"encoding/json"
	"log"
)

// tokenLimit is the maximum token count for agent-mode responses.
const tokenLimit = 500

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
		if patternsAdded > 0 && totalTokens+patternTokens > tokenLimit {
			result.TokenLimitReached = true
			break
		}

		// Add pattern
		result.Patterns = append(result.Patterns, output)
		totalTokens += patternTokens
		patternsAdded++

		// If first pattern alone exceeds limit, mark it but continue
		if patternsAdded == 1 && totalTokens > tokenLimit {
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

// buildVerboseResponse builds a comprehensive response for human consumption
func buildVerboseResponse(candidates []*ThreatPattern, limit int) QueryResult {
	result := QueryResult{
		PatternCount:    len(candidates),
		VerbosePatterns: make([]PatternOutputVerbose, 0, limit),
	}

	for i, p := range candidates {
		if i >= limit {
			break
		}

		verbose := PatternOutputVerbose{
			ID:          p.ID,
			Name:        p.Name,
			Severity:    p.Severity,
			Likelihood:  p.Likelihood,
			Threat:      p.AgentSummary.Threat,
			Check:       p.AgentSummary.Check,
			Fix:         p.AgentSummary.Fix,
			Description: p.Description,
		}

		// Add attack scenario (Tier A patterns)
		if p.AttackScenario != nil {
			verbose.AttackScenario = &AttackScenarioOutput{
				Narrative:     p.AttackScenario.Narrative,
				Preconditions: p.AttackScenario.Preconditions,
				Steps:         p.AttackScenario.AttackSteps,
				Impact:        p.AttackScenario.Impact,
			}
		}

		// Add mitigations with full code examples
		verbose.Mitigations = make([]MitigationVerbose, len(p.Mitigations))
		for j, m := range p.Mitigations {
			verbose.Mitigations[j] = MitigationVerbose{
				ID:                   m.ID,
				Name:                 m.Name,
				Description:          m.Description,
				Effectiveness:        m.Effectiveness,
				ImplementationEffort: m.ImplementationEffort,
				Tradeoffs:            m.Tradeoffs,
				CodeExamples:         convertCodeExamples(m.CodeExamples),
			}
		}

		// Add related patterns
		for _, rp := range p.RelatedPatterns {
			verbose.RelatedPatterns = append(verbose.RelatedPatterns, rp.ID)
		}

		// Add CWE/OWASP references
		for _, ref := range p.Provenance.PublicReferences {
			if ref.CWE != "" {
				verbose.CWEReferences = append(verbose.CWEReferences, ref.CWE)
			}
			if ref.OWASP != "" {
				verbose.OWASPReferences = append(verbose.OWASPReferences, ref.OWASP)
			}
		}

		result.VerbosePatterns = append(result.VerbosePatterns, verbose)
	}

	result.PatternsIncluded = len(result.VerbosePatterns)
	return result
}

// convertCodeExamples converts CodeExample to CodeExampleVerbose
func convertCodeExamples(examples []CodeExample) []CodeExampleVerbose {
	verbose := make([]CodeExampleVerbose, len(examples))
	for i, ex := range examples {
		verbose[i] = CodeExampleVerbose(ex)
	}
	return verbose
}
