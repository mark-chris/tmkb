package knowledge

import (
	"encoding/json"
	"testing"
)

func TestPatternOutputVerbose_JSON(t *testing.T) {
	verbose := PatternOutputVerbose{
		ID:          "TMKB-001",
		Name:        "Test Pattern",
		Severity:    "critical",
		Likelihood:  "high",
		Threat:      "Test threat",
		Check:       "Test check",
		Fix:         "Test fix",
		Description: "Full description here",
		AttackScenario: &AttackScenarioOutput{
			Narrative:     "Attack narrative",
			Preconditions: []string{"precond1", "precond2"},
			Steps: []AttackStep{
				{Step: 1, Action: "Step 1", Detail: "Details"},
			},
			Impact: Impact{
				Confidentiality: "high",
				Integrity:       "high",
				Availability:    "low",
			},
		},
		Mitigations: []MitigationVerbose{
			{
				ID:                   "MIT-001",
				Name:                 "Mitigation",
				Description:          "Details",
				Effectiveness:        "high",
				ImplementationEffort: "medium",
			},
		},
		RelatedPatterns: []string{"TMKB-002"},
		CWEReferences:   []string{"CWE-285"},
	}

	data, err := json.Marshal(verbose)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded["id"] != "TMKB-001" {
		t.Errorf("Expected id=TMKB-001, got %v", decoded["id"])
	}

	if _, ok := decoded["attack_scenario"]; !ok {
		t.Error("Expected attack_scenario in output")
	}
}

func TestPatternOutputVerbose_OmitsNilFields(t *testing.T) {
	verbose := PatternOutputVerbose{
		ID:          "TMKB-001",
		Name:        "Test",
		Severity:    "high",
		Likelihood:  "medium",
		Threat:      "t",
		Check:       "c",
		Fix:         "f",
		Description: "d",
		// AttackScenario nil (Tier B pattern)
	}

	data, err := json.Marshal(verbose)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Should omit attack_scenario when nil
	if _, ok := decoded["attack_scenario"]; ok {
		t.Error("Expected attack_scenario to be omitted when nil")
	}
}

func TestCodeExampleVerbose_BothCodes(t *testing.T) {
	example := CodeExampleVerbose{
		Language:       "python",
		Framework:      "flask",
		Description:    "Example",
		VulnerableCode: "bad code",
		SecureCode:     "good code",
	}

	data, err := json.Marshal(example)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if _, ok := decoded["vulnerable_code"]; !ok {
		t.Error("Expected vulnerable_code in output")
	}

	if _, ok := decoded["secure_code"]; !ok {
		t.Error("Expected secure_code in output")
	}
}
