package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// PatternWrapper handles the top-level threat_pattern key in YAML files
type PatternWrapper struct {
	ThreatPattern ThreatPattern `yaml:"threat_pattern"`
}

// Loader handles loading patterns from the filesystem
type Loader struct {
	basePath string
}

// NewLoader creates a new pattern loader with the given base path
func NewLoader(basePath string) *Loader {
	return &Loader{basePath: basePath}
}

// LoadAll loads all patterns from the patterns directory
func (l *Loader) LoadAll() ([]ThreatPattern, error) {
	var patterns []ThreatPattern

	err := filepath.Walk(l.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-YAML files
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		pattern, err := l.LoadFile(path)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}

		patterns = append(patterns, pattern)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk patterns directory: %w", err)
	}

	return patterns, nil
}

// LoadFile loads a single pattern from a YAML file
func (l *Loader) LoadFile(path string) (ThreatPattern, error) {
	// Validate path to prevent directory traversal attacks
	if err := l.validatePath(path); err != nil {
		return ThreatPattern{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ThreatPattern{}, fmt.Errorf("failed to read file: %w", err)
	}

	var wrapper PatternWrapper
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return ThreatPattern{}, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return wrapper.ThreatPattern, nil
}

// LoadByID loads a specific pattern by its ID
func (l *Loader) LoadByID(id string) (ThreatPattern, error) {
	patterns, err := l.LoadAll()
	if err != nil {
		return ThreatPattern{}, err
	}

	for _, p := range patterns {
		if p.ID == id {
			return p, nil
		}
	}

	return ThreatPattern{}, fmt.Errorf("pattern not found: %s", id)
}

// LoadByCategory loads all patterns in a category
func (l *Loader) LoadByCategory(category string) ([]ThreatPattern, error) {
	patterns, err := l.LoadAll()
	if err != nil {
		return nil, err
	}

	var filtered []ThreatPattern
	for _, p := range patterns {
		if strings.EqualFold(p.Category, category) {
			filtered = append(filtered, p)
		}
	}

	return filtered, nil
}

// validatePath ensures the given path is within the loader's basePath
// and prevents directory traversal attacks
func (l *Loader) validatePath(path string) error {
	// Clean and resolve the paths to absolute form
	cleanPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	cleanBase, err := filepath.Abs(filepath.Clean(l.basePath))
	if err != nil {
		return fmt.Errorf("failed to resolve base path: %w", err)
	}

	// Check if the clean path is within the base path
	relPath, err := filepath.Rel(cleanBase, cleanPath)
	if err != nil {
		return fmt.Errorf("failed to compute relative path: %w", err)
	}

	// If the relative path starts with "..", it's outside the base path
	if strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
		return fmt.Errorf("path traversal detected: %s is outside base path %s", path, l.basePath)
	}

	return nil
}
