package knowledge

import (
	"encoding/json"
	"fmt"
	"strings"
)

// OutputFormat specifies the output format
type OutputFormat string

// Output format constants.
const (
	FormatJSON OutputFormat = "json"
	FormatText OutputFormat = "text"
)

// FormatOutput formats a query result for display
func FormatOutput(result QueryResult, format OutputFormat, verbose bool) (string, error) {
	switch format {
	case FormatJSON:
		return formatJSON(result)
	case FormatText:
		return formatText(result, verbose)
	default:
		return formatJSON(result)
	}
}

func formatJSON(result QueryResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

func formatText(result QueryResult, _ bool) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Found %d relevant threat pattern(s)\n", result.PatternCount))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")

	for i, p := range result.Patterns {
		sb.WriteString(fmt.Sprintf("[%d] %s", i+1, p.ID))
		if p.Name != "" {
			sb.WriteString(fmt.Sprintf(": %s", p.Name))
		}
		sb.WriteString(fmt.Sprintf(" (Severity: %s)\n", p.Severity))
		sb.WriteString(strings.Repeat("-", 40) + "\n")

		sb.WriteString(fmt.Sprintf("THREAT: %s\n\n", p.Threat))
		sb.WriteString(fmt.Sprintf("CHECK:  %s\n\n", p.Check))
		sb.WriteString(fmt.Sprintf("FIX:    %s\n\n", p.Fix))
	}

	if result.CodePattern != nil {
		sb.WriteString(strings.Repeat("=", 50) + "\n")
		sb.WriteString("SECURE CODE TEMPLATE\n")
		sb.WriteString(fmt.Sprintf("Language: %s | Framework: %s\n",
			result.CodePattern.Language, result.CodePattern.Framework))
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		sb.WriteString(result.CodePattern.SecureTemplate)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// FormatPatternDetail formats a single pattern for detailed display
func FormatPatternDetail(p *ThreatPattern, format OutputFormat) (string, error) {
	switch format {
	case FormatJSON:
		data, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return string(data), nil
	case FormatText:
		return formatPatternText(p), nil
	default:
		data, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return string(data), nil
	}
}

func formatPatternText(p *ThreatPattern) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("%s: %s\n", p.ID, p.Name))
	sb.WriteString(fmt.Sprintf("Tier: %s | Severity: %s | Likelihood: %s\n",
		p.Tier, p.Severity, p.Likelihood))
	sb.WriteString(fmt.Sprintf("Category: %s > %s\n", p.Category, p.Subcategory))
	sb.WriteString(fmt.Sprintf("Language: %s | Framework: %s\n", p.Language, p.Framework))
	sb.WriteString(strings.Repeat("=", 60) + "\n\n")

	// Description
	sb.WriteString("DESCRIPTION\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")
	sb.WriteString(strings.TrimSpace(p.Description) + "\n\n")

	// Agent Summary
	sb.WriteString("AGENT SUMMARY\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")
	sb.WriteString(fmt.Sprintf("Threat: %s\n", p.AgentSummary.Threat))
	sb.WriteString(fmt.Sprintf("Check:  %s\n", p.AgentSummary.Check))
	sb.WriteString(fmt.Sprintf("Fix:    %s\n\n", p.AgentSummary.Fix))

	// LLM Blindspots
	if len(p.Differentiation.LLMBlindspots) > 0 {
		sb.WriteString("LLM BLINDSPOTS\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, bs := range p.Differentiation.LLMBlindspots {
			sb.WriteString(fmt.Sprintf("• %s\n", bs))
		}
		sb.WriteString("\n")
	}

	// Mitigations
	if len(p.Mitigations) > 0 {
		sb.WriteString("MITIGATIONS\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, m := range p.Mitigations {
			name := m.Name
			if name == "" {
				name = m.ID
			}
			sb.WriteString(fmt.Sprintf("[%s] %s\n", m.ID, name))
			sb.WriteString(fmt.Sprintf("    Effectiveness: %s | Effort: %s\n",
				m.Effectiveness, m.ImplementationEffort))
			sb.WriteString(fmt.Sprintf("    %s\n\n", strings.Split(m.Description, "\n")[0]))
		}
	}

	// Generalizes To
	if len(p.GeneralizesTo) > 0 {
		sb.WriteString("GENERALIZES TO\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, g := range p.GeneralizesTo {
			sb.WriteString(fmt.Sprintf("• %s\n", g))
		}
		sb.WriteString("\n")
	}

	// References
	if len(p.Provenance.PublicReferences) > 0 {
		sb.WriteString("REFERENCES\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, ref := range p.Provenance.PublicReferences {
			if ref.CWE != "" {
				sb.WriteString(fmt.Sprintf("• %s: %s\n", ref.CWE, ref.Name))
			} else if ref.OWASP != "" {
				sb.WriteString(fmt.Sprintf("• %s: %s\n", ref.OWASP, ref.Name))
			}
			if ref.URL != "" {
				sb.WriteString(fmt.Sprintf("  %s\n", ref.URL))
			}
		}
	}

	return sb.String()
}
