package knowledge

import (
	"sort"
	"strings"
)

// QueryOptions configures a query
type QueryOptions struct {
	Context   string
	Language  string
	Framework string
	Category  string
	Limit     int
	Verbosity string // "agent" or "human"
}

// QueryResult holds the results of a query
type QueryResult struct {
	PatternCount      int                     `json:"pattern_count"`
	PatternsIncluded  int                     `json:"patterns_included"`
	TokenCount        int                     `json:"token_count,omitempty"`
	TokenLimitReached bool                    `json:"token_limit_reached,omitempty"`
	Patterns          []PatternOutput         `json:"patterns,omitempty"`
	VerbosePatterns   []PatternOutputVerbose  `json:"verbose_patterns,omitempty"`
	CodePattern       *CodePatternOutput      `json:"code_pattern,omitempty"`
}

// PatternOutput is the agent-facing summary of a pattern
type PatternOutput struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	Severity string `json:"severity"`
	Threat   string `json:"threat"`
	Check    string `json:"check"`
	Fix      string `json:"fix"`
}

// PatternOutputVerbose is the human-facing detailed output
type PatternOutputVerbose struct {
	ID                string                `json:"id"`
	Name              string                `json:"name"`
	Severity          string                `json:"severity"`
	Likelihood        string                `json:"likelihood"`
	Threat            string                `json:"threat"`
	Check             string                `json:"check"`
	Fix               string                `json:"fix"`
	Description       string                `json:"description"`
	AttackScenario    *AttackScenarioOutput `json:"attack_scenario,omitempty"`
	Mitigations       []MitigationVerbose   `json:"mitigations"`
	RelatedPatterns   []string              `json:"related_patterns,omitempty"`
	CWEReferences     []string              `json:"cwe_references,omitempty"`
	OWASPReferences   []string              `json:"owasp_references,omitempty"`
}

// AttackScenarioOutput provides full attack scenario details
type AttackScenarioOutput struct {
	Narrative     string       `json:"narrative"`
	Preconditions []string     `json:"preconditions"`
	Steps         []AttackStep `json:"steps"`
	Impact        Impact       `json:"impact"`
}

// MitigationVerbose provides detailed mitigation information
type MitigationVerbose struct {
	ID                   string               `json:"id"`
	Name                 string               `json:"name"`
	Description          string               `json:"description"`
	Effectiveness        string               `json:"effectiveness"`
	ImplementationEffort string               `json:"implementation_effort"`
	Tradeoffs            []string             `json:"tradeoffs,omitempty"`
	CodeExamples         []CodeExampleVerbose `json:"code_examples,omitempty"`
}

// CodeExampleVerbose shows both vulnerable and secure code
type CodeExampleVerbose struct {
	Language       string `json:"language"`
	Framework      string `json:"framework"`
	Description    string `json:"description"`
	VulnerableCode string `json:"vulnerable_code,omitempty"`
	SecureCode     string `json:"secure_code,omitempty"`
}

// CodePatternOutput provides a code template for the most relevant pattern
type CodePatternOutput struct {
	Language       string `json:"language"`
	Framework      string `json:"framework"`
	SecureTemplate string `json:"secure_template"`
}

// patternWithScore holds a pattern and its relevance score
type patternWithScore struct {
	pattern *ThreatPattern
	score   float64
}

// Query executes a query against the index
func Query(idx *Index, opts QueryOptions) QueryResult {
	var candidates []*ThreatPattern

	// Start with context-based matching if provided
	if opts.Context != "" {
		candidates = idx.MatchContext(opts.Context)
	} else {
		// Otherwise get all patterns
		all := idx.GetAll()
		for i := range all {
			candidates = append(candidates, &all[i])
		}
	}

	// Filter by language if specified
	if opts.Language != "" {
		candidates = filterByLanguage(candidates, opts.Language)
	}

	// Filter by framework if specified
	if opts.Framework != "" && opts.Framework != "any" {
		candidates = filterByFramework(candidates, opts.Framework)
	}

	// Filter by category if specified
	if opts.Category != "" {
		candidates = filterByCategory(candidates, opts.Category)
	}

	// Sort by relevance if context provided, otherwise by severity
	if opts.Context != "" && strings.TrimSpace(opts.Context) != "" {
		// Extract keywords from context
		queryKeywords := ExtractKeywords(opts.Context)

		// Calculate relevance scores
		scored := make([]patternWithScore, len(candidates))
		for i, p := range candidates {
			score := CalculateRelevance(queryKeywords, p.Triggers.Keywords)
			scored[i] = patternWithScore{
				pattern: p,
				score:   score,
			}
		}

		// Sort by relevance (highest first), then severity, then likelihood
		sortByRelevance(scored)

		// Extract sorted patterns
		for i, s := range scored {
			candidates[i] = s.pattern
		}
	} else {
		// Sort by severity (critical > high > medium > low) then by likelihood
		sortBySeverity(candidates)
	}

	// Apply limit (default to 3 for agent output)
	limit := opts.Limit
	if limit <= 0 {
		if opts.Verbosity == "human" {
			limit = 10
		} else {
			limit = 3
		}
	}
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	// Build output
	result := QueryResult{
		PatternCount: len(candidates),
		Patterns:     make([]PatternOutput, 0, len(candidates)),
	}

	for _, p := range candidates {
		output := PatternOutput{
			ID:       p.ID,
			Severity: p.Severity,
			Threat:   p.AgentSummary.Threat,
			Check:    p.AgentSummary.Check,
			Fix:      p.AgentSummary.Fix,
		}

		if opts.Verbosity == "human" {
			output.Name = p.Name
		}

		result.Patterns = append(result.Patterns, output)
	}

	// Add code pattern from most relevant match
	if len(candidates) > 0 {
		codePattern := extractCodePattern(candidates[0], opts.Language, opts.Framework)
		if codePattern != nil {
			result.CodePattern = codePattern
		}
	}

	return result
}

// filterByLanguage filters patterns by programming language
func filterByLanguage(patterns []*ThreatPattern, language string) []*ThreatPattern {
	var filtered []*ThreatPattern
	langLower := strings.ToLower(language)
	for _, p := range patterns {
		if strings.ToLower(p.Language) == langLower {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// filterByFramework filters patterns by framework
func filterByFramework(patterns []*ThreatPattern, framework string) []*ThreatPattern {
	var filtered []*ThreatPattern
	fwLower := strings.ToLower(framework)
	for _, p := range patterns {
		if strings.ToLower(p.Framework) == fwLower {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// filterByCategory filters patterns by category
func filterByCategory(patterns []*ThreatPattern, category string) []*ThreatPattern {
	var filtered []*ThreatPattern
	catLower := strings.ToLower(category)
	for _, p := range patterns {
		if strings.ToLower(p.Category) == catLower {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// sortByRelevance sorts patterns by relevance score (highest first),
// with severity and likelihood as tiebreakers
func sortByRelevance(scored []patternWithScore) {
	severityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
	}

	likelihoodOrder := map[string]int{
		"high":   0,
		"medium": 1,
		"low":    2,
	}

	sort.Slice(scored, func(i, j int) bool {
		// Primary: relevance score (higher is better)
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}

		// Secondary: severity (critical > high > medium > low)
		si := severityOrder[strings.ToLower(scored[i].pattern.Severity)]
		sj := severityOrder[strings.ToLower(scored[j].pattern.Severity)]
		if si != sj {
			return si < sj
		}

		// Tertiary: likelihood (high > medium > low)
		li := likelihoodOrder[strings.ToLower(scored[i].pattern.Likelihood)]
		lj := likelihoodOrder[strings.ToLower(scored[j].pattern.Likelihood)]
		if li != lj {
			return li < lj
		}

		// Quaternary: alphabetical by ID
		return scored[i].pattern.ID < scored[j].pattern.ID
	})
}

// sortBySeverity sorts patterns by severity (critical > high > medium > low)
func sortBySeverity(patterns []*ThreatPattern) {
	severityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
	}

	likelihoodOrder := map[string]int{
		"high":   0,
		"medium": 1,
		"low":    2,
	}

	sort.Slice(patterns, func(i, j int) bool {
		si := severityOrder[strings.ToLower(patterns[i].Severity)]
		sj := severityOrder[strings.ToLower(patterns[j].Severity)]
		if si != sj {
			return si < sj
		}

		li := likelihoodOrder[strings.ToLower(patterns[i].Likelihood)]
		lj := likelihoodOrder[strings.ToLower(patterns[j].Likelihood)]
		if li != lj {
			return li < lj
		}

		// Alphabetical by ID as tiebreaker
		return patterns[i].ID < patterns[j].ID
	})
}

// extractCodePattern finds the best code example for the query
func extractCodePattern(p *ThreatPattern, language, framework string) *CodePatternOutput {
	if len(p.Mitigations) == 0 {
		return nil
	}

	// Find the most effective mitigation with code examples
	for _, m := range p.Mitigations {
		if m.Effectiveness != "high" {
			continue
		}
		for _, ex := range m.CodeExamples {
			if language != "" && !strings.EqualFold(ex.Language, language) {
				continue
			}
			if framework != "" && framework != "any" && !strings.Contains(strings.ToLower(ex.Framework), strings.ToLower(framework)) {
				continue
			}
			if ex.SecureCode != "" {
				return &CodePatternOutput{
					Language:       ex.Language,
					Framework:      ex.Framework,
					SecureTemplate: ex.SecureCode,
				}
			}
		}
	}

	// Fallback: any mitigation with code
	for _, m := range p.Mitigations {
		for _, ex := range m.CodeExamples {
			if ex.SecureCode != "" {
				return &CodePatternOutput{
					Language:       ex.Language,
					Framework:      ex.Framework,
					SecureTemplate: ex.SecureCode,
				}
			}
		}
	}

	return nil
}
