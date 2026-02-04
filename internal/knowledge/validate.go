package knowledge

import (
	"fmt"
	"strings"
)

// ValidationError represents a single validation error
type ValidationError struct {
	PatternID string
	Field     string
	Message   string
	Severity  string // "error" or "warning"
}

func (e ValidationError) String() string {
	return fmt.Sprintf("[%s] %s: %s - %s", e.Severity, e.PatternID, e.Field, e.Message)
}

// ValidationResult holds all validation errors for a pattern
type ValidationResult struct {
	PatternID string
	IsValid   bool
	Errors    []ValidationError
	Warnings  []ValidationError
}

// Validate validates a single pattern
func Validate(p ThreatPattern) ValidationResult {
	result := ValidationResult{
		PatternID: p.ID,
		IsValid:   true,
		Errors:    make([]ValidationError, 0),
		Warnings:  make([]ValidationError, 0),
	}

	// Required fields for all patterns
	result.checkRequired(p.ID, "id", p.ID)
	result.checkRequired(p.ID, "name", p.Name)
	result.checkRequired(p.ID, "tier", p.Tier)
	result.checkRequired(p.ID, "category", p.Category)
	result.checkRequired(p.ID, "severity", p.Severity)
	result.checkRequired(p.ID, "description", p.Description)

	// Validate tier value
	if p.Tier != "" && p.Tier != "A" && p.Tier != "B" {
		result.addError(p.ID, "tier", "must be 'A' or 'B'")
	}

	// Validate severity value
	validSeverities := map[string]bool{"critical": true, "high": true, "medium": true, "low": true}
	if p.Severity != "" && !validSeverities[strings.ToLower(p.Severity)] {
		result.addError(p.ID, "severity", "must be critical, high, medium, or low")
	}

	// Validate agent summary
	if p.AgentSummary.Threat == "" {
		result.addError(p.ID, "agent_summary.threat", "required")
	}
	if p.AgentSummary.Check == "" {
		result.addError(p.ID, "agent_summary.check", "required")
	}
	if p.AgentSummary.Fix == "" {
		result.addError(p.ID, "agent_summary.fix", "required")
	}

	// Check agent summary token count (rough estimate: 1 token ≈ 4 chars)
	agentSummaryLen := len(p.AgentSummary.Threat) + len(p.AgentSummary.Check) + len(p.AgentSummary.Fix)
	if agentSummaryLen > 400 { // ~100 tokens
		result.addWarning(p.ID, "agent_summary", 
			fmt.Sprintf("may exceed 100 tokens (approx %d chars)", agentSummaryLen))
	}

	// Validate triggers
	if len(p.Triggers.Keywords) == 0 {
		result.addWarning(p.ID, "triggers.keywords", "no keywords defined")
	}

	// Validate mitigations
	if len(p.Mitigations) == 0 {
		result.addError(p.ID, "mitigations", "at least one mitigation required")
	}

	for i, m := range p.Mitigations {
		if m.ID == "" {
			result.addError(p.ID, fmt.Sprintf("mitigations[%d].id", i), "required")
		}
		if m.Description == "" {
			result.addError(p.ID, fmt.Sprintf("mitigations[%d].description", i), "required")
		}
	}

	// Tier A specific requirements
	if p.Tier == "A" {
		result.validateTierA(p)
	}

	// Provenance requirements
	if p.Provenance.SourceType == "" {
		result.addWarning(p.ID, "provenance.source_type", "recommended for traceability")
	}

	return result
}

func (r *ValidationResult) validateTierA(p ThreatPattern) {
	// Tier A requires attack scenario
	if p.AttackScenario == nil {
		r.addError(p.ID, "attack_scenario", "required for Tier A patterns")
	} else {
		if p.AttackScenario.Narrative == "" {
			r.addError(p.ID, "attack_scenario.narrative", "required for Tier A patterns")
		}
		if len(p.AttackScenario.Preconditions) == 0 {
			r.addWarning(p.ID, "attack_scenario.preconditions", "recommended for Tier A patterns")
		}
	}

	// Tier A requires generalizes_to
	if len(p.GeneralizesTo) == 0 {
		r.addWarning(p.ID, "generalizes_to", "recommended for Tier A patterns")
	}

	// Tier A requires security principles
	if len(p.SecurityPrinciples) == 0 {
		r.addWarning(p.ID, "security_principles", "recommended for Tier A patterns")
	}

	// Tier A requires code examples in mitigations
	hasCodeExample := false
	for _, m := range p.Mitigations {
		if len(m.CodeExamples) > 0 {
			hasCodeExample = true
			break
		}
	}
	if !hasCodeExample {
		r.addWarning(p.ID, "mitigations", "Tier A patterns should have code examples")
	}

	// Tier A should have both vulnerable and secure code
	for i, m := range p.Mitigations {
		for j, ex := range m.CodeExamples {
			if ex.VulnerableCode == "" && ex.SecureCode == "" {
				r.addWarning(p.ID, 
					fmt.Sprintf("mitigations[%d].code_examples[%d]", i, j),
					"should have vulnerable_code and/or secure_code")
			}
		}
	}
}

func (r *ValidationResult) checkRequired(patternID, field, value string) {
	if value == "" {
		r.addError(patternID, field, "required field is empty")
	}
}

func (r *ValidationResult) addError(patternID, field, message string) {
	r.IsValid = false
	r.Errors = append(r.Errors, ValidationError{
		PatternID: patternID,
		Field:     field,
		Message:   message,
		Severity:  "error",
	})
}

func (r *ValidationResult) addWarning(patternID, field, message string) {
	r.Warnings = append(r.Warnings, ValidationError{
		PatternID: patternID,
		Field:     field,
		Message:   message,
		Severity:  "warning",
	})
}

// ValidateAll validates all patterns and returns results
func ValidateAll(patterns []ThreatPattern) []ValidationResult {
	results := make([]ValidationResult, 0, len(patterns))
	for _, p := range patterns {
		results = append(results, Validate(p))
	}
	return results
}
