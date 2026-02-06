# Output Format Polish Design

**Date**: 2026-02-06
**Issue**: #4
**Status**: Approved

## Overview

Polish the TMKB output format to meet specifications for token limits, ordering, and verbose mode completeness. Ensures agent output stays under 500 tokens while providing comprehensive human-readable verbose output.

## Goals

1. Enforce <500 token limit for agent output
2. Verify deterministic ordering (already implemented via Issue #3)
3. Implement full verbose mode with attack narratives and code examples
4. Maintain hard cap of 3 patterns for agent mode, 10 for verbose mode

## Design Decisions

### 1. Token Counting Strategy

**Decision**: Strict enforcement with actual tokenizer

Use `pkoukk/tiktoken-go` library for accurate token counting:
- GPT-4/Claude-compatible approximation (cl100k_base encoding)
- Count tokens for each pattern before adding to response
- Fallback to character-based approximation if tokenizer fails

**Rationale**: Production-quality enforcement ensures reliable behavior across all queries. Approximations (1 token ≈ 4 chars) are too loose.

### 2. Output Mode Separation

**Decision**: Separate types for agent vs. verbose output

- **Agent Mode**: `PatternOutput` (concise, token-limited)
- **Verbose Mode**: `PatternOutputVerbose` (comprehensive, unlimited)

**Rationale**: Type-safe contracts make the distinction explicit. Agent output optimized for LLM consumption, verbose output for human learning.

### 3. Token Limit Behavior

**Decision**: Remove patterns that don't fit

When approaching 500 token limit:
1. Add patterns one-by-one from sorted list
2. Stop when next pattern would exceed limit
3. Return only patterns that fit completely
4. Set `token_limit_reached: true` if limit hit

**Rationale**: Maintains quality (no truncated sentences). Patterns already sorted by relevance, so removing least relevant makes sense.

### 4. Verbose Mode Content

**Decision**: Full attack scenarios and code examples

Include for each pattern:
- Complete attack scenario (Tier A patterns)
  - Full narrative
  - All preconditions
  - Step-by-step attack flow
  - Complete impact assessment
- All mitigations with vulnerable + secure code examples
- Related patterns
- CWE/OWASP references

**Rationale**: Verbose mode is for human consumption (security professionals, learners). No token limits mean we can provide comprehensive educational content.

## Architecture

### Two Output Modes

```
┌─────────────────────────────────────────────────────────┐
│                      Query Engine                        │
└───────────────────────┬─────────────────────────────────┘
                        │
           ┌────────────┴────────────┐
           │                         │
    ┌──────▼──────┐         ┌───────▼────────┐
    │ Agent Mode  │         │ Verbose Mode   │
    │ (concise)   │         │ (detailed)     │
    └──────┬──────┘         └───────┬────────┘
           │                         │
    ┌──────▼──────────┐     ┌───────▼─────────────┐
    │ Token Counter   │     │ Full Data Extract   │
    │ Limit: 500      │     │ No Limits           │
    │ Patterns: 1-3   │     │ Patterns: 1-10      │
    └─────────────────┘     └─────────────────────┘
```

### Data Structures

#### Enhanced QueryResult

```go
type QueryResult struct {
    PatternCount      int                     `json:"pattern_count"`
    PatternsIncluded  int                     `json:"patterns_included"`
    TokenCount        int                     `json:"token_count,omitempty"`
    TokenLimitReached bool                    `json:"token_limit_reached,omitempty"`
    Patterns          []PatternOutput         `json:"patterns,omitempty"`
    VerbosePatterns   []PatternOutputVerbose  `json:"verbose_patterns,omitempty"`
    CodePattern       *CodePatternOutput      `json:"code_pattern,omitempty"`
}
```

**Fields**:
- `pattern_count`: Total patterns matched by query
- `patterns_included`: Actual patterns returned (may be less if token limit hit)
- `token_count`: Actual tokens used (agent mode only)
- `token_limit_reached`: True if we stopped adding patterns due to token limit
- `patterns`: Agent mode output (1-3 patterns)
- `verbose_patterns`: Verbose mode output (1-10 patterns)

#### PatternOutputVerbose

```go
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

type AttackScenarioOutput struct {
    Narrative     string       `json:"narrative"`
    Preconditions []string     `json:"preconditions"`
    Steps         []AttackStep `json:"steps"`
    Impact        Impact       `json:"impact"`
}

type MitigationVerbose struct {
    ID                   string               `json:"id"`
    Name                 string               `json:"name"`
    Description          string               `json:"description"`
    Effectiveness        string               `json:"effectiveness"`
    ImplementationEffort string               `json:"implementation_effort"`
    Tradeoffs            []string             `json:"tradeoffs,omitempty"`
    CodeExamples         []CodeExampleVerbose `json:"code_examples,omitempty"`
}

type CodeExampleVerbose struct {
    Language       string `json:"language"`
    Framework      string `json:"framework"`
    Description    string `json:"description"`
    VulnerableCode string `json:"vulnerable_code,omitempty"`
    SecureCode     string `json:"secure_code,omitempty"`
}
```

## Implementation Details

### Token Counter

```go
type TokenCounter struct {
    encoder *tiktoken.Tiktoken
}

func NewTokenCounter() (*TokenCounter, error) {
    enc, err := tiktoken.GetEncoding("cl100k_base")
    if err != nil {
        return nil, err
    }
    return &TokenCounter{encoder: enc}, nil
}

func (tc *TokenCounter) CountTokens(text string) int {
    if tc.encoder == nil {
        // Fallback: approximate with character count
        return len(text) / 4
    }
    return len(tc.encoder.Encode(text, nil, nil))
}
```

### Agent Mode Response Builder

```go
func buildAgentResponse(candidates []*ThreatPattern, limit int) QueryResult {
    counter, err := NewTokenCounter()
    if err != nil {
        // Log warning, continue with fallback
        log.Warn("Token counter initialization failed, using approximation")
    }

    result := QueryResult{
        PatternCount: len(candidates),
        Patterns:     make([]PatternOutput, 0, limit),
    }

    totalTokens := 0
    const TOKEN_LIMIT = 500

    for _, p := range candidates {
        output := PatternOutput{
            ID:       p.ID,
            Severity: p.Severity,
            Threat:   p.AgentSummary.Threat,
            Check:    p.AgentSummary.Check,
            Fix:      p.AgentSummary.Fix,
        }

        // Calculate tokens for this pattern
        patternJSON, _ := json.Marshal(output)
        patternTokens := counter.CountTokens(string(patternJSON))

        // Check if adding this pattern would exceed limit
        if totalTokens + patternTokens > TOKEN_LIMIT {
            result.TokenLimitReached = true
            break
        }

        result.Patterns = append(result.Patterns, output)
        totalTokens += patternTokens
    }

    result.PatternsIncluded = len(result.Patterns)
    result.TokenCount = totalTokens

    // Add code pattern from most relevant match
    if len(result.Patterns) > 0 {
        result.CodePattern = extractCodePattern(candidates[0], "", "")
    }

    return result
}
```

### Verbose Mode Response Builder

```go
func buildVerboseResponse(candidates []*ThreatPattern, limit int) QueryResult {
    result := QueryResult{
        PatternCount:    len(candidates),
        VerbosePatterns: make([]PatternOutputVerbose, 0, limit),
    }

    for i, p := range candidates {
        if i >= limit {
            break
        }

        verbose := PatternOutputVerbose{
            ID:          p.ID,
            Name:        p.Name,
            Severity:    p.Severity,
            Likelihood:  p.Likelihood,
            Threat:      p.AgentSummary.Threat,
            Check:       p.AgentSummary.Check,
            Fix:         p.AgentSummary.Fix,
            Description: p.Description,
        }

        // Add attack scenario (Tier A patterns)
        if p.AttackScenario != nil {
            verbose.AttackScenario = &AttackScenarioOutput{
                Narrative:     p.AttackScenario.Narrative,
                Preconditions: p.AttackScenario.Preconditions,
                Steps:         p.AttackScenario.AttackSteps,
                Impact:        p.AttackScenario.Impact,
            }
        }

        // Add mitigations with full code examples
        verbose.Mitigations = make([]MitigationVerbose, len(p.Mitigations))
        for j, m := range p.Mitigations {
            verbose.Mitigations[j] = MitigationVerbose{
                ID:                   m.ID,
                Name:                 m.Name,
                Description:          m.Description,
                Effectiveness:        m.Effectiveness,
                ImplementationEffort: m.ImplementationEffort,
                Tradeoffs:            m.Tradeoffs,
                CodeExamples:         convertCodeExamples(m.CodeExamples),
            }
        }

        // Add related patterns
        for _, rp := range p.RelatedPatterns {
            verbose.RelatedPatterns = append(verbose.RelatedPatterns, rp.ID)
        }

        // Add CWE/OWASP references
        for _, ref := range p.Provenance.PublicReferences {
            if ref.CWE != "" {
                verbose.CWEReferences = append(verbose.CWEReferences, ref.CWE)
            }
            if ref.OWASP != "" {
                verbose.OWASPReferences = append(verbose.OWASPReferences, ref.OWASP)
            }
        }

        result.VerbosePatterns = append(result.VerbosePatterns, verbose)
    }

    result.PatternsIncluded = len(result.VerbosePatterns)
    return result
}

func convertCodeExamples(examples []CodeExample) []CodeExampleVerbose {
    verbose := make([]CodeExampleVerbose, len(examples))
    for i, ex := range examples {
        verbose[i] = CodeExampleVerbose{
            Language:       ex.Language,
            Framework:      ex.Framework,
            Description:    ex.Description,
            VulnerableCode: ex.VulnerableCode,
            SecureCode:     ex.SecureCode,
        }
    }
    return verbose
}
```

### Modified Query Function

```go
func Query(idx *Index, opts QueryOptions) QueryResult {
    // ... existing candidate selection and sorting logic ...

    // Build response based on verbosity
    if opts.Verbosity == "human" {
        limit := opts.Limit
        if limit <= 0 {
            limit = 10
        }
        return buildVerboseResponse(candidates, limit)
    } else {
        limit := opts.Limit
        if limit <= 0 {
            limit = 3
        }
        return buildAgentResponse(candidates, limit)
    }
}
```

## Error Handling

### Token Counter Initialization Fails

**Scenario**: tiktoken library fails to load encoding

**Handling**:
```go
counter, err := NewTokenCounter()
if err != nil {
    log.Warn("Token counter initialization failed: %v, using character approximation", err)
    // Continue with fallback in CountTokens method
}
```

**Fallback**: Character-based approximation (1 token ≈ 4 characters)

### Zero Patterns Fit in Token Limit

**Scenario**: First pattern exceeds 500 tokens (shouldn't happen with well-formed patterns)

**Handling**:
```go
if len(result.Patterns) == 0 && len(candidates) > 0 {
    // Force include at least one pattern
    result.Patterns = append(result.Patterns, buildPatternOutput(candidates[0]))
    result.TokenLimitReached = true
    log.Warn("Single pattern exceeds token limit: %s", candidates[0].ID)
}
```

**Rationale**: Better to return one oversized pattern than nothing. Indicates pattern summaries need refinement.

### Missing Verbose Fields

**Scenario**: Tier B patterns don't have AttackScenario

**Handling**:
- Use `omitempty` JSON tag on `AttackScenario` field
- Field omitted from JSON rather than showing `null`
- Graceful degradation

## Testing Strategy

### Unit Tests

**Token Counting**:
```go
func TestTokenCounter_Accuracy(t *testing.T)
func TestTokenCounter_FallbackMode(t *testing.T)
```

**Response Building**:
```go
func TestBuildAgentResponse_UnderLimit(t *testing.T)
func TestBuildAgentResponse_ExactlyAtLimit(t *testing.T)
func TestBuildAgentResponse_ExceedsLimit(t *testing.T)
func TestBuildAgentResponse_SinglePatternTooLarge(t *testing.T)
func TestBuildVerboseResponse_AllFields(t *testing.T)
func TestBuildVerboseResponse_TierBPattern(t *testing.T) // No attack scenario
```

**Limit Enforcement**:
```go
func TestQuery_AgentModeTokenLimit(t *testing.T)
func TestQuery_VerboseModeNoLimit(t *testing.T)
```

### Integration Tests

**Real Pattern Testing**:
```go
func TestQuery_RealPatterns_AgentMode(t *testing.T) {
    // Load actual patterns from patterns/authorization/
    // Verify agent output < 500 tokens
}

func TestQuery_RealPatterns_VerboseMode(t *testing.T) {
    // Load actual patterns
    // Verify all verbose fields populated
}
```

**Ordering Verification**:
```go
func TestQuery_DeterministicOrdering(t *testing.T) {
    // Run same query multiple times
    // Verify identical ordering
}
```

### Manual Validation

1. Test with TMKB-AUTHZ-001 (Tier A, has attack scenario)
2. Test with TMKB-AUTHZ-006 (Tier B, no attack scenario)
3. Verify attack narratives render correctly in verbose mode
4. Check code examples show both vulnerable & secure versions
5. Confirm token counts are accurate

## Dependencies

### New Dependencies

```go
require (
    github.com/pkoukk/tiktoken-go v0.1.6
)
```

**Rationale**: Accurate token counting for GPT/Claude models. Pure Go implementation, no CGO.

## Migration Plan

### Phase 1: Add Token Counter
1. Add tiktoken-go dependency
2. Implement TokenCounter with fallback
3. Unit test token counting

### Phase 2: Enhance Data Structures
1. Add new fields to QueryResult
2. Create PatternOutputVerbose and related types
3. Update JSON tags and omitempty

### Phase 3: Implement Response Builders
1. Implement buildAgentResponse with token limiting
2. Implement buildVerboseResponse with full data
3. Unit test both builders

### Phase 4: Integrate with Query
1. Modify Query function to call appropriate builder
2. Integration tests
3. Manual validation with real patterns

### Phase 5: Documentation
1. Update CLI help text
2. Update MCP server documentation
3. Add examples to README

## Success Criteria

- [x] Design approved
- [ ] Agent output consistently < 500 tokens for typical queries
- [ ] Verbose mode includes full attack scenarios (Tier A)
- [ ] Verbose mode includes vulnerable + secure code examples
- [ ] All tests pass (unit + integration)
- [ ] Manual validation with real patterns succeeds
- [ ] Deterministic ordering verified (already implemented)

## Trade-offs

### Token Counter Dependency

**Pro**: Accurate token counting, production-ready
**Con**: External dependency adds ~200KB to binary
**Decision**: Worth it for reliability

### Separate Response Builders

**Pro**: Clear separation of concerns, easier to test
**Con**: Some code duplication between builders
**Decision**: Maintainability > DRY in this case

### Verbose Mode Unlimited Tokens

**Pro**: Comprehensive educational content
**Con**: Large responses for queries with many patterns
**Decision**: Acceptable for human consumption, users can set limit

## Future Enhancements

1. **Streaming Output**: For very large verbose responses, support streaming JSON
2. **Custom Token Limits**: Allow users to configure token limit (default 500)
3. **Token Budget per Pattern**: Ensure even distribution across patterns
4. **Compression**: Optionally compress verbose output for network efficiency

## References

- Issue #4: Output Format Polish
- Issue #3: Query Engine Improvements (ordering already implemented)
- tiktoken-go: https://github.com/pkoukk/tiktoken-go
- OpenAI Tokenizer: https://platform.openai.com/tokenizer
