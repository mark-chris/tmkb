package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidatePath_ValidPaths tests that valid paths within basePath are accepted
func TestValidatePath_ValidPaths(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "patterns")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	loader := NewLoader(baseDir)

	tests := []struct {
		name string
		path string
	}{
		{
			name: "file in base directory",
			path: filepath.Join(baseDir, "pattern.yaml"),
		},
		{
			name: "file in subdirectory",
			path: filepath.Join(baseDir, "authz", "pattern.yaml"),
		},
		{
			name: "file in nested subdirectory",
			path: filepath.Join(baseDir, "authz", "api", "pattern.yaml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.validatePath(tt.path)
			if err != nil {
				t.Errorf("validatePath(%q) returned error: %v, want nil", tt.path, err)
			}
		})
	}
}

// TestValidatePath_DirectoryTraversal tests that directory traversal attempts are blocked
func TestValidatePath_DirectoryTraversal(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "patterns")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a file outside the base directory
	outsideDir := filepath.Join(tmpDir, "outside")
	if err := os.MkdirAll(outsideDir, 0755); err != nil {
		t.Fatalf("Failed to create outside directory: %v", err)
	}

	loader := NewLoader(baseDir)

	tests := []struct {
		name string
		path string
	}{
		{
			name: "parent directory traversal with ..",
			path: filepath.Join(baseDir, "..", "outside", "pattern.yaml"),
		},
		{
			name: "multiple parent directory traversal",
			path: filepath.Join(baseDir, "..", "..", "etc", "passwd"),
		},
		{
			name: "absolute path outside base",
			path: filepath.Join(outsideDir, "pattern.yaml"),
		},
		{
			name: "root directory",
			path: "/etc/passwd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.validatePath(tt.path)
			if err == nil {
				t.Errorf("validatePath(%q) returned nil, want error for path traversal", tt.path)
			}
			if !strings.Contains(err.Error(), "outside base path") && !strings.Contains(err.Error(), "path traversal") {
				t.Errorf("validatePath(%q) error = %v, want error mentioning path traversal or outside base path", tt.path, err)
			}
		})
	}
}

// TestLoadFile_DirectoryTraversalBlocked tests that LoadFile prevents directory traversal
func TestLoadFile_DirectoryTraversalBlocked(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "patterns")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a sensitive file outside the base directory
	outsideFile := filepath.Join(tmpDir, "sensitive.yaml")
	sensitiveContent := []byte("threat_pattern:\n  id: EVIL\n  name: Should Not Load\n")
	if err := os.WriteFile(outsideFile, sensitiveContent, 0644); err != nil {
		t.Fatalf("Failed to create sensitive file: %v", err)
	}

	loader := NewLoader(baseDir)

	// Attempt to load file using directory traversal
	traversalPath := filepath.Join(baseDir, "..", "sensitive.yaml")
	_, err := loader.LoadFile(traversalPath)

	if err == nil {
		t.Error("LoadFile with directory traversal returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "outside base path") && !strings.Contains(err.Error(), "path traversal") {
		t.Errorf("LoadFile error = %v, want error mentioning path traversal", err)
	}
}

// TestLoadFile_ValidFile tests loading a valid pattern file
func TestLoadFile_ValidFile(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "patterns")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a valid pattern file
	patternFile := filepath.Join(baseDir, "test-pattern.yaml")
	patternContent := []byte(`threat_pattern:
  id: TMKB-TEST-001
  name: Test Pattern
  severity: high
  likelihood: medium
  category: testing
  language: go
  framework: test
  description: A test pattern
  agent_summary:
    threat: Test threat
    check: Test check
    fix: Test fix
  triggers:
    keywords:
      - test
      - pattern
  mitigations: []
  provenance:
    author: Test Author
    date: 2026-02-06
    public_references: []
`)
	if err := os.WriteFile(patternFile, patternContent, 0644); err != nil {
		t.Fatalf("Failed to create pattern file: %v", err)
	}

	loader := NewLoader(baseDir)
	pattern, err := loader.LoadFile(patternFile)

	if err != nil {
		t.Fatalf("LoadFile returned error: %v, want nil", err)
	}

	// Verify pattern was loaded correctly
	if pattern.ID != "TMKB-TEST-001" {
		t.Errorf("pattern.ID = %q, want %q", pattern.ID, "TMKB-TEST-001")
	}
	if pattern.Name != "Test Pattern" {
		t.Errorf("pattern.Name = %q, want %q", pattern.Name, "Test Pattern")
	}
	if pattern.Severity != "high" {
		t.Errorf("pattern.Severity = %q, want %q", pattern.Severity, "high")
	}
}

// TestValidatePath_SymlinkTraversal tests that symlinks pointing outside base are blocked
func TestValidatePath_SymlinkTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "patterns")
	outsideDir := filepath.Join(tmpDir, "outside")

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("Failed to create base directory: %v", err)
	}
	if err := os.MkdirAll(outsideDir, 0755); err != nil {
		t.Fatalf("Failed to create outside directory: %v", err)
	}

	// Create a target file outside the base directory
	outsideFile := filepath.Join(outsideDir, "secret.yaml")
	if err := os.WriteFile(outsideFile, []byte("secret data"), 0644); err != nil {
		t.Fatalf("Failed to create outside file: %v", err)
	}

	// Create a symlink inside the base directory pointing outside
	symlinkPath := filepath.Join(baseDir, "evil.yaml")
	if err := os.Symlink(outsideFile, symlinkPath); err != nil {
		t.Skipf("Cannot create symlinks (permission denied): %v", err)
	}

	loader := NewLoader(baseDir)
	err := loader.validatePath(symlinkPath)

	if err == nil {
		t.Error("validatePath with symlink traversal returned nil, want error")
	}
}

// TestLoadAll_SecurityIsolation tests that LoadAll respects path boundaries
func TestLoadAll_SecurityIsolation(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "patterns")
	outsideDir := filepath.Join(tmpDir, "outside")

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("Failed to create base directory: %v", err)
	}
	if err := os.MkdirAll(outsideDir, 0755); err != nil {
		t.Fatalf("Failed to create outside directory: %v", err)
	}

	// Create pattern inside base directory
	insidePattern := filepath.Join(baseDir, "inside.yaml")
	insideContent := []byte(`threat_pattern:
  id: TMKB-INSIDE-001
  name: Inside Pattern
  severity: high
  likelihood: medium
  category: testing
  language: go
  framework: test
  description: Inside pattern
  agent_summary:
    threat: Test
    check: Test
    fix: Test
  triggers:
    keywords: [test]
  mitigations: []
  provenance:
    author: Test
    date: 2026-02-06
    public_references: []
`)
	if err := os.WriteFile(insidePattern, insideContent, 0644); err != nil {
		t.Fatalf("Failed to create inside pattern: %v", err)
	}

	// Create pattern outside base directory (should not be loaded)
	outsidePattern := filepath.Join(outsideDir, "outside.yaml")
	outsideContent := []byte(`threat_pattern:
  id: TMKB-OUTSIDE-001
  name: Outside Pattern
  severity: critical
  likelihood: high
  category: testing
  language: go
  framework: test
  description: Outside pattern
  agent_summary:
    threat: Test
    check: Test
    fix: Test
  triggers:
    keywords: [test]
  mitigations: []
  provenance:
    author: Test
    date: 2026-02-06
    public_references: []
`)
	if err := os.WriteFile(outsidePattern, outsideContent, 0644); err != nil {
		t.Fatalf("Failed to create outside pattern: %v", err)
	}

	loader := NewLoader(baseDir)
	patterns, err := loader.LoadAll()

	if err != nil {
		t.Fatalf("LoadAll returned error: %v, want nil", err)
	}

	// Should only load the inside pattern
	if len(patterns) != 1 {
		t.Errorf("LoadAll loaded %d patterns, want 1", len(patterns))
	}

	// Verify only the inside pattern was loaded
	if len(patterns) > 0 && patterns[0].ID != "TMKB-INSIDE-001" {
		t.Errorf("LoadAll loaded pattern with ID %q, want %q", patterns[0].ID, "TMKB-INSIDE-001")
	}

	// Verify outside pattern was not loaded
	for _, p := range patterns {
		if p.ID == "TMKB-OUTSIDE-001" {
			t.Error("LoadAll loaded pattern from outside base directory, should be isolated")
		}
	}
}
