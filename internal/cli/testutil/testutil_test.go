package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestSetupTestPatterns_CreatesDirectory verifies that SetupTestPatterns creates a temporary directory
func TestSetupTestPatterns_CreatesDirectory(t *testing.T) {
	fixture := SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Verify directory exists
	if _, err := os.Stat(fixture.Dir); os.IsNotExist(err) {
		t.Errorf("SetupTestPatterns did not create directory at %s", fixture.Dir)
	}

	// Verify it's a directory
	info, err := os.Stat(fixture.Dir)
	if err != nil {
		t.Fatalf("Failed to stat directory: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("SetupTestPatterns created %s but it's not a directory", fixture.Dir)
	}
}

// TestSetupTestPatterns_CreatesPatterns verifies that SetupTestPatterns creates exactly 3 test patterns
func TestSetupTestPatterns_CreatesPatterns(t *testing.T) {
	fixture := SetupTestPatterns(t)
	defer fixture.Cleanup()

	// Verify we have exactly 3 patterns
	if len(fixture.Patterns) != 3 {
		t.Errorf("SetupTestPatterns created %d patterns, want 3", len(fixture.Patterns))
	}

	// Verify pattern IDs
	expectedIDs := map[string]bool{
		"TMKB-TEST-001": false,
		"TMKB-TEST-002": false,
		"TMKB-TEST-003": false,
	}

	for _, pattern := range fixture.Patterns {
		if _, exists := expectedIDs[pattern.ID]; !exists {
			t.Errorf("Unexpected pattern ID: %s", pattern.ID)
		}
		expectedIDs[pattern.ID] = true
	}

	// Verify all expected IDs were found
	for id, found := range expectedIDs {
		if !found {
			t.Errorf("Expected pattern ID %s not found", id)
		}
	}

	// Verify pattern-specific details
	for _, pattern := range fixture.Patterns {
		switch pattern.ID {
		case "TMKB-TEST-001":
			if pattern.Language != "Python" {
				t.Errorf("TMKB-TEST-001 language = %s, want Python", pattern.Language)
			}
			if pattern.Framework != "Flask" {
				t.Errorf("TMKB-TEST-001 framework = %s, want Flask", pattern.Framework)
			}
			expectedKeywords := []string{"background", "job", "authorization"}
			if !equalStringSlices(pattern.Triggers.Keywords, expectedKeywords) {
				t.Errorf("TMKB-TEST-001 keywords = %v, want %v", pattern.Triggers.Keywords, expectedKeywords)
			}

		case "TMKB-TEST-002":
			if pattern.Language != "Go" {
				t.Errorf("TMKB-TEST-002 language = %s, want Go", pattern.Language)
			}
			if pattern.Framework != "any" {
				t.Errorf("TMKB-TEST-002 framework = %s, want any", pattern.Framework)
			}
			expectedKeywords := []string{"session", "token"}
			if !equalStringSlices(pattern.Triggers.Keywords, expectedKeywords) {
				t.Errorf("TMKB-TEST-002 keywords = %v, want %v", pattern.Triggers.Keywords, expectedKeywords)
			}

		case "TMKB-TEST-003":
			if pattern.Language != "JavaScript" {
				t.Errorf("TMKB-TEST-003 language = %s, want JavaScript", pattern.Language)
			}
			if pattern.Framework != "Express" {
				t.Errorf("TMKB-TEST-003 framework = %s, want Express", pattern.Framework)
			}
			expectedKeywords := []string{"tenant", "isolation"}
			if !equalStringSlices(pattern.Triggers.Keywords, expectedKeywords) {
				t.Errorf("TMKB-TEST-003 keywords = %v, want %v", pattern.Triggers.Keywords, expectedKeywords)
			}
		}
	}

	// Verify pattern files were written to disk
	yamlFiles, err := filepath.Glob(filepath.Join(fixture.Dir, "*.yaml"))
	if err != nil {
		t.Fatalf("Failed to glob YAML files: %v", err)
	}
	if len(yamlFiles) != 3 {
		t.Errorf("Found %d YAML files, want 3", len(yamlFiles))
	}
}

// TestCreateTestPattern_ValidStructure verifies that CreateTestPattern generates a valid pattern
func TestCreateTestPattern_ValidStructure(t *testing.T) {
	pattern := CreateTestPattern("TMKB-TEST-999", "Test Pattern", "Python", "Django")

	// Verify required fields are set
	if pattern.ID != "TMKB-TEST-999" {
		t.Errorf("pattern.ID = %s, want TMKB-TEST-999", pattern.ID)
	}
	if pattern.Name != "Test Pattern" {
		t.Errorf("pattern.Name = %s, want Test Pattern", pattern.Name)
	}
	if pattern.Language != "Python" {
		t.Errorf("pattern.Language = %s, want Python", pattern.Language)
	}
	if pattern.Framework != "Django" {
		t.Errorf("pattern.Framework = %s, want Django", pattern.Framework)
	}

	// Verify tier and version are set
	if pattern.Tier == "" {
		t.Error("pattern.Tier is empty, want non-empty value")
	}
	if pattern.Version == "" {
		t.Error("pattern.Version is empty, want non-empty value")
	}

	// Verify category and severity are set
	if pattern.Category == "" {
		t.Error("pattern.Category is empty, want non-empty value")
	}
	if pattern.Severity == "" {
		t.Error("pattern.Severity is empty, want non-empty value")
	}
	if pattern.Likelihood == "" {
		t.Error("pattern.Likelihood is empty, want non-empty value")
	}

	// Verify description is set
	if pattern.Description == "" {
		t.Error("pattern.Description is empty, want non-empty value")
	}

	// Verify agent_summary is set
	if pattern.AgentSummary.Threat == "" {
		t.Error("pattern.AgentSummary.Threat is empty, want non-empty value")
	}
	if pattern.AgentSummary.Check == "" {
		t.Error("pattern.AgentSummary.Check is empty, want non-empty value")
	}
	if pattern.AgentSummary.Fix == "" {
		t.Error("pattern.AgentSummary.Fix is empty, want non-empty value")
	}

	// Verify triggers has at least one keyword
	if len(pattern.Triggers.Keywords) == 0 {
		t.Error("pattern.Triggers.Keywords is empty, want at least one keyword")
	}

	// Verify provenance is set
	if pattern.Provenance.SourceType == "" {
		t.Error("pattern.Provenance.SourceType is empty, want non-empty value")
	}
	if pattern.Provenance.Description == "" {
		t.Error("pattern.Provenance.Description is empty, want non-empty value")
	}

	// Verify the pattern can be serialized to YAML
	wrapper := PatternWrapper{ThreatPattern: pattern}
	data, err := yaml.Marshal(&wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal pattern to YAML: %v", err)
	}

	// Verify the YAML can be unmarshaled back
	var unwrapped PatternWrapper
	if err := yaml.Unmarshal(data, &unwrapped); err != nil {
		t.Fatalf("Failed to unmarshal pattern from YAML: %v", err)
	}

	// Verify key fields are preserved
	if unwrapped.ThreatPattern.ID != pattern.ID {
		t.Errorf("After YAML round-trip, ID = %s, want %s", unwrapped.ThreatPattern.ID, pattern.ID)
	}
}

// equalStringSlices compares two string slices for equality
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
