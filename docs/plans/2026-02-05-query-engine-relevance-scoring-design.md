# Query Engine Relevance Scoring Design

**Date:** 2026-02-05
**Issue:** #3 - Query Engine Improvements
**Status:** Design Approved

---

## Overview

### Goal
Improve query relevance by scoring how well patterns match the user's query context, prioritizing patterns that match multiple keywords over those that match only one.

### High-Level Changes
We'll add a new `scoring` module to `internal/knowledge/` with two main components:

1. **Keyword Extractor**: Converts query string into N-grams (1, 2, and 3-word phrases)
2. **Relevance Scorer**: Calculates match scores using hybrid formula: `(matched_count × 2) + (matched_keywords / total_pattern_keywords)`

### Modified Flow
```
Current:  Query → MatchContext → Filter → Sort by Severity → Return
New:      Query → Extract N-grams → MatchContext → Score Each Match →
          Sort by Relevance → Filter → Return
```

### Key Design Decisions
- **Relevance-first sorting**: Most relevant patterns appear first, regardless of severity
- **Severity as tiebreaker**: When relevance scores are equal, severity/likelihood determines order
- **Backward compatible**: If no context provided, falls back to existing severity-based sorting
- **No external dependencies**: Pure Go implementation, no NLP libraries needed
- **Keywords only**: Focus on keyword matching; keep action/file pattern matching as-is for now

### Files to Create/Modify
- **New**: `internal/knowledge/scoring.go` - keyword extraction and scoring logic
- **Modify**: `internal/knowledge/query.go` - integrate scoring into query flow
- **New**: `internal/knowledge/scoring_test.go` - unit tests for scoring
- **Modify**: `internal/knowledge/query_test.go` - integration tests

---

## Component 1: Keyword Extraction

### N-gram Extractor Implementation

The keyword extractor will generate 1-gram, 2-gram, and 3-gram phrases from the query context:

```go
// ExtractKeywords extracts N-grams (1, 2, 3-word phrases) from query context
func ExtractKeywords(context string) []string {
    // Normalize: lowercase, split on spaces/punctuation
    words := tokenize(context)

    // Generate N-grams
    keywords := make(map[string]bool) // deduplicate

    // 1-grams: individual words
    for _, word := range words {
        keywords[word] = true
    }

    // 2-grams: consecutive pairs
    for i := 0; i < len(words)-1; i++ {
        keywords[words[i] + " " + words[i+1]] = true
    }

    // 3-grams: consecutive triples
    for i := 0; i < len(words)-2; i++ {
        keywords[words[i] + " " + words[i+1] + " " + words[i+2]] = true
    }

    return mapKeysToSlice(keywords)
}
```

### Tokenization Strategy

```go
// tokenize splits on whitespace and common punctuation
func tokenize(text string) []string {
    // Replace punctuation with spaces
    text = strings.ToLower(text)
    text = regexp.MustCompile(`[^a-z0-9\s-]`).ReplaceAllString(text, " ")

    // Split and filter empty strings
    words := strings.Fields(text)

    // Keep hyphens in compound words (e.g., "multi-tenant" stays together)
    return words
}
```

### Example
Input: `"building a multi-tenant API background job"`

Output N-grams:
- 1-word: `["building", "a", "multi-tenant", "api", "background", "job"]`
- 2-word: `["building a", "a multi-tenant", "multi-tenant api", "api background", "background job"]`
- 3-word: `["building a multi-tenant", "a multi-tenant api", "multi-tenant api background", "api background job"]`

Total: ~17 keywords extracted (with duplicates removed)

---

## Component 2: Relevance Scoring Algorithm

### Scoring Function

```go
// RelevanceScore calculates how well a pattern matches query keywords
type RelevanceScore struct {
    PatternID       string
    Score           float64
    MatchedCount    int
    MatchedKeywords []string
}

func CalculateRelevance(queryKeywords []string, pattern *ThreatPattern) RelevanceScore {
    patternKeywords := pattern.Triggers.Keywords

    // Find matches (case-insensitive)
    matched := findMatches(queryKeywords, patternKeywords)
    matchCount := len(matched)

    // Hybrid formula: (matched × 2) + (matched / total_pattern_keywords)
    coverage := float64(matchCount) / float64(len(patternKeywords))
    score := (float64(matchCount) * 2.0) + coverage

    return RelevanceScore{
        PatternID:       pattern.ID,
        Score:           score,
        MatchedCount:    matchCount,
        MatchedKeywords: matched,
    }
}

// findMatches returns query keywords that appear in pattern keywords
func findMatches(queryKW []string, patternKW []string) []string {
    var matches []string
    patternSet := make(map[string]bool)

    for _, pk := range patternKW {
        patternSet[strings.ToLower(pk)] = true
    }

    for _, qk := range queryKW {
        if patternSet[strings.ToLower(qk)] {
            matches = append(matches, qk)
        }
    }

    return matches
}
```

### Scoring Examples

**Pattern A**: 10 keywords, matches 3 query keywords
- Score = (3 × 2) + (3 / 10) = 6.0 + 0.3 = **6.3**

**Pattern B**: 5 keywords, matches 2 query keywords
- Score = (2 × 2) + (2 / 5) = 4.0 + 0.4 = **4.4**

**Pattern C**: 20 keywords, matches 4 query keywords
- Score = (4 × 2) + (4 / 20) = 8.0 + 0.2 = **8.2**

**Interpretation**: Pattern C wins despite having lowest coverage because it matched the most query terms. The match count dominates (2× weight), coverage breaks ties between patterns with same match count.

---

## Component 3: Integration with Query Flow

### Modified Query Function

```go
func Query(idx *Index, opts QueryOptions) QueryResult {
    var candidates []*ThreatPattern
    var scores map[string]float64 // pattern ID -> relevance score

    // Extract keywords from context if provided
    if opts.Context != "" {
        queryKeywords := ExtractKeywords(opts.Context)

        // Get initial matches
        candidates = idx.MatchContext(opts.Context)

        // Score each match
        scores = make(map[string]float64)
        for _, p := range candidates {
            score := CalculateRelevance(queryKeywords, p)
            scores[p.ID] = score.Score
        }

        // Sort by relevance (highest first)
        sortByRelevance(candidates, scores)

    } else {
        // No context: use all patterns, sort by severity (existing behavior)
        all := idx.GetAll()
        for i := range all {
            candidates = append(candidates, &all[i])
        }
        sortBySeverity(candidates) // existing function
    }

    // Apply filters (language, framework, category)
    candidates = applyFilters(candidates, opts)

    // Apply limit and build output (existing logic)
    return buildOutput(candidates, opts)
}
```

### New Sorting Function

```go
// sortByRelevance sorts patterns by relevance score (desc),
// then severity, then likelihood
func sortByRelevance(patterns []*ThreatPattern, scores map[string]float64) {
    sort.Slice(patterns, func(i, j int) bool {
        scoreI := scores[patterns[i].ID]
        scoreJ := scores[patterns[j].ID]

        // Primary: relevance score (higher is better)
        if scoreI != scoreJ {
            return scoreI > scoreJ
        }

        // Tiebreaker 1: severity
        si := severityOrder[strings.ToLower(patterns[i].Severity)]
        sj := severityOrder[strings.ToLower(patterns[j].Severity)]
        if si != sj {
            return si < sj // lower order = higher severity
        }

        // Tiebreaker 2: likelihood
        li := likelihoodOrder[strings.ToLower(patterns[i].Likelihood)]
        lj := likelihoodOrder[strings.ToLower(patterns[j].Likelihood)]
        if li != lj {
            return li < lj
        }

        // Final tiebreaker: ID
        return patterns[i].ID < patterns[j].ID
    })
}
```

### Backward Compatibility

- **No context provided**: Falls back to `sortBySeverity()` (existing behavior)
- **Context provided**: Uses relevance scoring
- **All existing filters still work**: Language, framework, category applied after scoring
- **Output format unchanged**: No breaking changes to API

---

## Performance Analysis

### Requirements
From Issue #3 acceptance criteria: **"Sub-second response time maintained"**

### Complexity Analysis

**Current Implementation (baseline):**
- MatchContext: O(K × P) where K = pattern keywords, P = patterns
- Sorting: O(P log P)
- **Total: O(K × P + P log P)**

**New Implementation:**
- Extract N-grams: O(W²) where W = words in query (~10-20 words → ~400 ops max)
- MatchContext: O(K × P) - unchanged
- Score calculations: O(P × Q × K) where Q = query keywords (~20-30)
- Sorting: O(P log P) - unchanged
- **Total: O(W² + K × P + P × Q × K + P log P)**

### Optimization Strategies

1. **Keyword Set Lookup**: Convert pattern keywords to hash set once
   ```go
   // O(K) instead of O(Q × K) per pattern
   patternSet := make(map[string]bool)
   for _, pk := range pattern.Triggers.Keywords {
       patternSet[strings.ToLower(pk)] = true
   }
   ```

2. **Early Filtering**: Only score patterns that matched in MatchContext
   - Don't score patterns with 0 matches
   - Typical query: 12 patterns → ~5-8 actually match → only score those

3. **N-gram Deduplication**: Use map during extraction to avoid duplicate work

4. **Lazy Evaluation**: Only extract keywords once, cache in QueryOptions

### Expected Performance

**Assumptions:**
- 50 patterns in index (current: ~12, growing)
- 5-10 patterns match per query
- 10 words in query → ~30 N-grams

**Rough Calculation:**
- Extract N-grams: 0.1ms
- MatchContext: 0.5ms (existing)
- Score 8 matches: 8 × (30 keywords × 10 pattern keywords) = 2,400 comparisons → 0.2ms
- Sort 8 patterns: negligible
- **Total: ~1ms**

**Conclusion**: Well under 1 second requirement, even with 10x growth to 500 patterns.

---

## Testing Strategy

### Unit Tests

**`scoring_test.go`:**
```go
func TestExtractKeywords(t *testing.T) {
    tests := []struct {
        input    string
        expected []string
    }{
        {
            input: "multi-tenant API",
            expected: []string{"multi-tenant", "api", "multi-tenant api"},
        },
        {
            input: "background job processing",
            expected: []string{"background", "job", "processing",
                              "background job", "job processing",
                              "background job processing"},
        },
    }
    // Test extraction produces correct N-grams
}

func TestCalculateRelevance(t *testing.T) {
    // Test scoring formula with known inputs
    // Pattern with 10 keywords, 3 matches → score 6.3
    // Pattern with 5 keywords, 2 matches → score 4.4
}

func TestSortByRelevance(t *testing.T) {
    // Test that higher scores rank first
    // Test severity tiebreaker works
    // Test likelihood secondary tiebreaker works
}
```

### Integration Tests

**`query_test.go`:**
```go
func TestQueryWithRelevanceScoring(t *testing.T) {
    // Setup: Index with known patterns
    // Query: "multi-tenant API background job"
    // Assert: Pattern matching both "multi-tenant" and "background"
    //         ranks above pattern matching only "API"
}

func TestBackwardCompatibility(t *testing.T) {
    // No context provided → severity-based sorting
    // Empty context → severity-based sorting
}
```

### Manual Test Scenarios (Varied Contexts)

Test with real-world queries to validate rankings:

1. **"Building a multi-tenant SaaS API"**
   - Should rank: tenant isolation, API auth patterns high
   - Should not rank: file upload patterns (not mentioned)

2. **"Background job processing uploaded files"**
   - Should rank: async job auth, file processing patterns
   - Should match: "background", "job", "processing", "files"

3. **"JWT token validation in Flask"**
   - Should rank: JWT-specific patterns first
   - Language filter: Python patterns prioritized
   - Framework filter: Flask patterns prioritized

4. **"Implementing admin dashboard"**
   - Should rank: authorization, role-based access patterns
   - Should match: "admin", "dashboard" keywords

5. **Single keyword: "authorization"**
   - Should match many patterns
   - Sorted by severity within matched set

---

## Acceptance Criteria Validation

From Issue #3:

- ✅ **Queries return most relevant patterns for given context**
  - Relevance scoring prioritizes multi-match patterns
  - Hybrid formula (match × 2 + coverage) balances breadth and specificity

- ✅ **Scoring produces consistent, sensible rankings**
  - Deterministic formula with clear semantics
  - Severity/likelihood as tiebreakers preserve critical pattern visibility

- ✅ **Sub-second response time maintained**
  - Performance analysis shows ~1ms typical query
  - Optimizations ensure scalability to 500+ patterns

- ✅ **Test with varied query contexts**
  - Manual test scenarios cover diverse use cases
  - Unit and integration tests validate correctness

---

## Implementation Tasks

1. Create `internal/knowledge/scoring.go`
   - Implement `ExtractKeywords()`
   - Implement `tokenize()`
   - Implement `CalculateRelevance()`
   - Implement `findMatches()`

2. Modify `internal/knowledge/query.go`
   - Add relevance scoring path to `Query()`
   - Implement `sortByRelevance()`
   - Preserve backward compatibility

3. Create `internal/knowledge/scoring_test.go`
   - Unit tests for keyword extraction
   - Unit tests for relevance scoring
   - Unit tests for sorting

4. Modify `internal/knowledge/query_test.go`
   - Integration tests for relevance-scored queries
   - Backward compatibility tests

5. Manual testing with real patterns
   - Run test scenarios
   - Verify rankings make sense
   - Performance validation

---

## Future Enhancements (Out of Scope)

These were considered but deferred for future iterations:

- **Stop word removal**: May reduce noise, but YAGNI for now
- **Action matching with relevance**: Extend scoring to `triggers.actions`
- **File pattern matching**: Extend scoring to `triggers.file_patterns`
- **Fuzzy matching**: Handle synonyms ("async" → "asynchronous")
- **Semantic understanding**: Embeddings-based matching (requires ML dependencies)

---

## Conclusion

This design provides a focused, implementable solution for multi-keyword relevance scoring. It maintains backward compatibility, meets performance requirements, and establishes a foundation for future query improvements.

The hybrid scoring formula balances match count (primary signal) with coverage (tie-breaker), ensuring patterns that match more query terms rank higher while preventing overly generic patterns from dominating.
