package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mark-chris/tmkb/internal/knowledge"
	"gopkg.in/yaml.v3"
)

// TestFixture holds test resources and provides cleanup
type TestFixture struct {
	Dir      string                     // Temporary directory containing test patterns
	Patterns []knowledge.ThreatPattern  // Loaded test patterns
	Cleanup  func()                      // Cleanup function to remove temporary resources
}

// PatternWrapper handles the top-level threat_pattern key in YAML files
// This mirrors knowledge.PatternWrapper but is defined here to avoid import cycles
type PatternWrapper struct {
	ThreatPattern knowledge.ThreatPattern `yaml:"threat_pattern"`
}

// SetupTestPatterns creates a temporary directory with 3 test patterns
// It returns a TestFixture with the directory path, loaded patterns, and cleanup function
func SetupTestPatterns(t *testing.T) *TestFixture {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()

	// Create test patterns
	patterns := []knowledge.ThreatPattern{
		CreateTestPattern("TMKB-TEST-001", "Test Pattern 001", "Python", "Flask"),
		CreateTestPattern("TMKB-TEST-002", "Test Pattern 002", "Go", "any"),
		CreateTestPattern("TMKB-TEST-003", "Test Pattern 003", "JavaScript", "Express"),
	}

	// Set specific keywords for each pattern as required
	patterns[0].Triggers.Keywords = []string{"background", "job", "authorization"}
	patterns[1].Triggers.Keywords = []string{"session", "token"}
	patterns[2].Triggers.Keywords = []string{"tenant", "isolation"}

	// Write patterns to disk
	for _, pattern := range patterns {
		if err := writePatternFile(tmpDir, pattern); err != nil {
			t.Fatalf("Failed to write pattern file: %v", err)
		}
	}

	return &TestFixture{
		Dir:      tmpDir,
		Patterns: patterns,
		Cleanup:  func() {}, // t.TempDir() handles cleanup automatically
	}
}

// CreateTestPattern generates a minimal valid threat pattern for testing
func CreateTestPattern(id, name, language, framework string) knowledge.ThreatPattern {
	return knowledge.ThreatPattern{
		ID:          id,
		Name:        name,
		Tier:        "B",
		Version:     "1.0",
		LastUpdated: "2026-02-06",
		Category:    "testing",
		Subcategory: "test",
		Language:    language,
		Framework:   framework,
		Severity:    "medium",
		Likelihood:  "medium",
		Description: "Test pattern for " + name,
		AgentSummary: knowledge.AgentSummary{
			Threat: "Test threat description",
			Check:  "Test check description",
			Fix:    "Test fix description",
		},
		Triggers: knowledge.Triggers{
			Keywords:     []string{"test"},
			Actions:      []string{},
			FilePatterns: []string{},
		},
		Differentiation: knowledge.Differentiation{
			LLMKnowledgeState: "partial",
			TMKBValue:         "test value",
			LLMBlindspots:     []string{"test blindspot"},
		},
		Provenance: knowledge.Provenance{
			SourceType:       "test",
			Description:      "Test pattern for unit testing",
			PublicReferences: []knowledge.PublicReference{},
		},
		Mitigations: []knowledge.Mitigation{
			{
				ID:                   "M1",
				Name:                 "Test Mitigation",
				Description:          "Test mitigation description",
				Effectiveness:        "high",
				ImplementationEffort: "medium",
			},
		},
	}
}

// writePatternFile writes a threat pattern to a YAML file in the specified directory
func writePatternFile(dir string, pattern knowledge.ThreatPattern) error {
	// Wrap pattern in threat_pattern key
	wrapper := PatternWrapper{ThreatPattern: pattern}

	// Marshal to YAML
	data, err := yaml.Marshal(&wrapper)
	if err != nil {
		return err
	}

	// Write to file
	filename := filepath.Join(dir, pattern.ID+".yaml")
	// #nosec G306 -- Test files don't need restrictive permissions
	return os.WriteFile(filename, data, 0644)
}
