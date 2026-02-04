package knowledge

import (
	"strings"
	"sync"
)

// Index provides fast lookups for patterns
type Index struct {
	patterns    []ThreatPattern
	byID        map[string]*ThreatPattern
	byCategory  map[string][]*ThreatPattern
	byKeyword   map[string][]*ThreatPattern
	byFramework map[string][]*ThreatPattern
	byLanguage  map[string][]*ThreatPattern
	mu          sync.RWMutex
}

// NewIndex creates a new empty index
func NewIndex() *Index {
	return &Index{
		patterns:    make([]ThreatPattern, 0),
		byID:        make(map[string]*ThreatPattern),
		byCategory:  make(map[string][]*ThreatPattern),
		byKeyword:   make(map[string][]*ThreatPattern),
		byFramework: make(map[string][]*ThreatPattern),
		byLanguage:  make(map[string][]*ThreatPattern),
	}
}

// Build creates the index from a slice of patterns
func (idx *Index) Build(patterns []ThreatPattern) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Store patterns
	idx.patterns = patterns

	// Clear maps
	idx.byID = make(map[string]*ThreatPattern)
	idx.byCategory = make(map[string][]*ThreatPattern)
	idx.byKeyword = make(map[string][]*ThreatPattern)
	idx.byFramework = make(map[string][]*ThreatPattern)
	idx.byLanguage = make(map[string][]*ThreatPattern)

	// Build indexes
	for i := range idx.patterns {
		p := &idx.patterns[i]

		// By ID
		idx.byID[p.ID] = p

		// By category
		cat := strings.ToLower(p.Category)
		idx.byCategory[cat] = append(idx.byCategory[cat], p)

		// By framework
		fw := strings.ToLower(p.Framework)
		idx.byFramework[fw] = append(idx.byFramework[fw], p)

		// By language
		lang := strings.ToLower(p.Language)
		idx.byLanguage[lang] = append(idx.byLanguage[lang], p)

		// By keywords (from triggers)
		for _, kw := range p.Triggers.Keywords {
			kwLower := strings.ToLower(kw)
			idx.byKeyword[kwLower] = append(idx.byKeyword[kwLower], p)
		}
	}
}

// GetByID returns a pattern by its ID
func (idx *Index) GetByID(id string) *ThreatPattern {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.byID[id]
}

// GetByCategory returns all patterns in a category
func (idx *Index) GetByCategory(category string) []*ThreatPattern {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.byCategory[strings.ToLower(category)]
}

// GetByKeyword returns all patterns matching a keyword
func (idx *Index) GetByKeyword(keyword string) []*ThreatPattern {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.byKeyword[strings.ToLower(keyword)]
}

// GetByFramework returns all patterns for a framework
func (idx *Index) GetByFramework(framework string) []*ThreatPattern {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.byFramework[strings.ToLower(framework)]
}

// GetByLanguage returns all patterns for a language
func (idx *Index) GetByLanguage(language string) []*ThreatPattern {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.byLanguage[strings.ToLower(language)]
}

// GetAll returns all indexed patterns
func (idx *Index) GetAll() []ThreatPattern {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.patterns
}

// Count returns the number of indexed patterns
func (idx *Index) Count() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.patterns)
}

// MatchContext finds patterns relevant to a given context string
// Uses simple keyword matching; could be enhanced with fuzzy matching
func (idx *Index) MatchContext(context string) []*ThreatPattern {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	contextLower := strings.ToLower(context)
	seen := make(map[string]bool)
	var matches []*ThreatPattern

	// Check each keyword against the context
	for keyword, patterns := range idx.byKeyword {
		if strings.Contains(contextLower, keyword) {
			for _, p := range patterns {
				if !seen[p.ID] {
					seen[p.ID] = true
					matches = append(matches, p)
				}
			}
		}
	}

	// Also check action triggers
	for i := range idx.patterns {
		p := &idx.patterns[i]
		if seen[p.ID] {
			continue
		}
		for _, action := range p.Triggers.Actions {
			if strings.Contains(contextLower, strings.ToLower(action)) {
				seen[p.ID] = true
				matches = append(matches, p)
				break
			}
		}
	}

	return matches
}
