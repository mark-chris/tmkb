# Query Response Format Documentation

This document specifies the output format for TMKB query operations, which differ based on the intended consumer (AI agent vs. human).

## Overview

TMKB provides two output modes:

1. **Agent Mode (default)**: Optimized for AI coding agents
   - Token-limited responses (<500 tokens)
   - Concise summaries only
   - Maximum 3 patterns by default
   - JSON output only

2. **Verbose Mode**: Optimized for human consumption
   - Unlimited token budget
   - Comprehensive pattern details
   - Maximum 10 patterns by default
   - JSON or text output

## Agent Mode (Default)

### Purpose

Agent mode is designed for consumption by AI coding agents (Claude, GPT-4, etc.) that need:
- Fast, focused security context
- Token-efficient responses
- Actionable guidance without overwhelming detail
- Structured data for programmatic parsing

### Token Budget

- **Hard limit**: 500 tokens
- **Counting method**: tiktoken (cl100k_base encoding)
- **Behavior**: Stops including patterns when next pattern would exceed limit
- **Response field**: `token_limit_reached: true` if limit hit

### Output Structure

```json
{
  "pattern_count": 12,           // Total patterns matched by query
  "patterns_included": 3,        // Patterns included in response
  "token_count": 217,            // Actual tokens used
  "token_limit_reached": false,  // True if stopped due to limit
  "patterns": [
    {
      "id": "TMKB-AUTHZ-001",
      "severity": "high",
      "threat": "Background jobs execute without the authorization context from the original request",
      "check": "Verify authorization is re-checked in the job, not just the endpoint",
      "fix": "Pass user_id and tenant_id to job; re-validate permissions before operating on resources"
    }
  ],
  "code_pattern": {
    "language": "python",
    "framework": "flask-celery",
    "secure_template": "# Full secure code example here..."
  }
}
```

### Field Specifications

#### Top-Level Fields

| Field | Type | Description |
|-------|------|-------------|
| `pattern_count` | int | Total number of patterns matching the query |
| `patterns_included` | int | Number of patterns included in this response |
| `token_count` | int | Total tokens used by patterns array |
| `token_limit_reached` | bool | True if token limit prevented including more patterns |
| `patterns` | array | Array of PatternOutput objects (max 3 by default) |
| `code_pattern` | object | Code template from most relevant pattern (optional) |

#### PatternOutput Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Pattern identifier (e.g., TMKB-AUTHZ-001) |
| `severity` | string | Yes | critical, high, medium, or low |
| `threat` | string | Yes | One-line threat description |
| `check` | string | Yes | What to verify in code review |
| `fix` | string | Yes | How to remediate the vulnerability |
| `name` | string | No | Human-readable pattern name (optional) |

#### CodePatternOutput Object

| Field | Type | Description |
|-------|------|-------------|
| `language` | string | Programming language (e.g., python, go) |
| `framework` | string | Framework name (e.g., flask, django) |
| `secure_template` | string | Full secure code implementation |

### CLI Usage

```bash
# Agent mode (default)
tmkb query --context "background job processing"

# Agent mode with custom limit
tmkb query --context "multi-tenant API" --limit 5

# Agent mode with filters
tmkb query --context "file upload" --language python --framework flask
```

### Pattern Selection

Patterns are selected and ordered by:

1. **Relevance score** (if context provided)
   - Keyword matching between query and pattern triggers
   - Weighted by keyword frequency and position

2. **Severity** (tiebreaker)
   - critical > high > medium > low

3. **Likelihood** (tiebreaker)
   - high > medium > low

4. **Alphabetical by ID** (final tiebreaker)

### Token Limit Behavior

The token counter evaluates each pattern before adding it:

```go
// Pseudocode
for each pattern:
    patternTokens = count_tokens(pattern_json)
    if totalTokens + patternTokens > 500:
        set token_limit_reached = true
        break
    add pattern to response
    totalTokens += patternTokens
```

**Edge case**: If the first pattern alone exceeds 500 tokens, it is still included but `token_limit_reached` is set to true.

## Verbose Mode

### Purpose

Verbose mode is designed for human security reviewers who need:
- Complete pattern details including attack scenarios
- Full mitigation descriptions with tradeoffs
- Both vulnerable and secure code examples
- Related patterns and external references

### Token Budget

- **No token limit** (unlimited)
- **Pattern limit**: 10 by default (configurable)
- **Use case**: Deep analysis, security review, threat modeling sessions

### Output Structure

```json
{
  "pattern_count": 12,
  "patterns_included": 10,
  "verbose_patterns": [
    {
      "id": "TMKB-AUTHZ-001",
      "name": "Background Job Authorization Context Loss",
      "severity": "high",
      "likelihood": "high",
      "threat": "Background jobs execute without the authorization context from the original request",
      "check": "Verify authorization is re-checked in the job, not just the endpoint",
      "fix": "Pass user_id and tenant_id to job; re-validate permissions before operating on resources",
      "description": "Full multi-paragraph description of the vulnerability...",
      "attack_scenario": {
        "narrative": "Detailed attack narrative...",
        "preconditions": [
          "Application uses background job system",
          "Jobs process user-generated data"
        ],
        "steps": [
          {
            "step": 1,
            "action": "Attacker uploads malicious file",
            "outcome": "File ID queued for processing"
          }
        ],
        "impact": {
          "confidentiality": "high",
          "integrity": "high",
          "availability": "none"
        }
      },
      "mitigations": [
        {
          "id": "M1",
          "name": "Context Propagation",
          "description": "Pass authorization context to background jobs",
          "effectiveness": "high",
          "implementation_effort": "medium",
          "tradeoffs": [
            "Increased job payload size",
            "Requires job signature changes"
          ],
          "code_examples": [
            {
              "language": "python",
              "framework": "celery",
              "description": "Pass tenant_id to background job",
              "vulnerable_code": "# VULNERABLE: No context\n@celery.task\ndef process_file(file_id):\n    file = File.query.get(file_id)\n    ...",
              "secure_code": "# SECURE: Context included\n@celery.task\ndef process_file(file_id, user_id, tenant_id):\n    file = File.get_for_tenant(file_id, tenant_id)\n    ..."
            }
          ]
        }
      ],
      "related_patterns": [
        "TMKB-AUTHZ-004",
        "TMKB-AUTHZ-011"
      ],
      "cwe_references": [
        "CWE-862"
      ],
      "owasp_references": [
        "API1:2023"
      ]
    }
  ]
}
```

### Field Specifications

#### PatternOutputVerbose Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Pattern identifier |
| `name` | string | Yes | Human-readable pattern name |
| `severity` | string | Yes | critical, high, medium, or low |
| `likelihood` | string | Yes | Probability of exploitation |
| `threat` | string | Yes | One-line threat summary |
| `check` | string | Yes | Code review verification point |
| `fix` | string | Yes | Remediation guidance |
| `description` | string | Yes | Full vulnerability description |
| `attack_scenario` | object | Tier A only | Detailed attack walkthrough |
| `mitigations` | array | Yes | All mitigation strategies |
| `related_patterns` | array | Optional | Related pattern IDs |
| `cwe_references` | array | Optional | CWE identifiers |
| `owasp_references` | array | Optional | OWASP references |

#### AttackScenarioOutput Object

| Field | Type | Description |
|-------|------|-------------|
| `narrative` | string | Prose description of the attack |
| `preconditions` | array | Required conditions for attack |
| `steps` | array | Ordered attack steps |
| `impact` | object | CIA impact assessment |

#### MitigationVerbose Object

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Mitigation identifier |
| `name` | string | Mitigation name |
| `description` | string | Full mitigation description |
| `effectiveness` | string | high, medium, or low |
| `implementation_effort` | string | high, medium, or low |
| `tradeoffs` | array | Implementation considerations |
| `code_examples` | array | Vulnerable and secure code examples |

#### CodeExampleVerbose Object

| Field | Type | Description |
|-------|------|-------------|
| `language` | string | Programming language |
| `framework` | string | Framework name |
| `description` | string | Example context |
| `vulnerable_code` | string | Insecure implementation (optional) |
| `secure_code` | string | Secure implementation (optional) |

### CLI Usage

```bash
# Verbose mode
tmkb query --context "background job processing" --verbose

# Verbose mode with custom limit
tmkb query --context "multi-tenant API" --verbose --limit 5

# Verbose mode with text output (default when --verbose)
tmkb query --context "authorization" --verbose --format text

# Verbose mode with JSON
tmkb query --context "authorization" --verbose --format json
```

## Comparison Table

| Feature | Agent Mode | Verbose Mode |
|---------|------------|--------------|
| Token limit | 500 | Unlimited |
| Default pattern limit | 3 | 10 |
| Pattern details | Summary only | Full details |
| Attack scenarios | No | Yes (Tier A) |
| Code examples | Secure only | Vulnerable + Secure |
| Mitigations | No | All with tradeoffs |
| References | No | CWE, OWASP |
| Output format | JSON only | JSON or Text |
| Use case | AI agent consumption | Human security review |

## Implementation Details

### Token Counting

Agent mode uses tiktoken-go with the cl100k_base encoding (GPT-4/Claude):

```go
import "github.com/pkoukk/tiktoken-go"

counter, _ := NewTokenCounter()
tokens := counter.CountTokens(jsonString)
```

Fallback: If tiktoken fails to load, approximate as `len(text) / 4`.

### Response Building

The query engine uses two separate response builders:

```go
// Agent mode
func buildAgentResponse(candidates []*ThreatPattern, limit int) QueryResult

// Verbose mode
func buildVerboseResponse(candidates []*ThreatPattern, limit int) QueryResult
```

Key differences:
- Agent mode: iterates patterns and stops when token limit reached
- Verbose mode: includes all fields, no token checking

### Code Pattern Selection

For agent mode, the code pattern is selected from the most relevant pattern:

1. Find first mitigation with `effectiveness: high`
2. Look for code example matching language/framework filters
3. Extract `secure_code` field
4. Fallback: use any mitigation's code example

Verbose mode includes code examples within each mitigation object.

## Usage Examples

### Example 1: Agent Mode Query

```bash
tmkb query --context "multi-tenant SaaS with background jobs"
```

Response:
```json
{
  "pattern_count": 2,
  "patterns_included": 2,
  "token_count": 138,
  "patterns": [
    {
      "id": "TMKB-AUTHZ-001",
      "severity": "high",
      "threat": "Background jobs execute without authorization context",
      "check": "Verify authorization is re-checked in the job",
      "fix": "Pass user_id and tenant_id to job; re-validate permissions"
    },
    {
      "id": "TMKB-AUTHZ-004",
      "severity": "critical",
      "threat": "Missing tenant_id filter in ANY query exposes all tenants' data",
      "check": "Verify EVERY query includes tenant filter",
      "fix": "Use base query class with automatic tenant filter"
    }
  ],
  "code_pattern": {
    "language": "python",
    "framework": "celery",
    "secure_template": "..."
  }
}
```

### Example 2: Verbose Mode Query

```bash
tmkb query --context "authorization bypass" --verbose --limit 2
```

Response includes full `verbose_patterns` array with attack scenarios, all mitigations, and references.

### Example 3: Filtered Query

```bash
tmkb query --context "file upload" --language python --framework flask --limit 5
```

Only patterns matching Python + Flask are returned.

## Integration Guide

### For AI Agents

When integrating TMKB into an AI coding workflow:

1. Query TMKB when generating security-sensitive code
2. Parse the `patterns` array
3. Incorporate `threat`, `check`, and `fix` into code generation
4. Use `code_pattern.secure_template` as implementation guidance
5. Check `token_limit_reached` - if true, consider refining the query

### For Security Tools

When building security review tools:

1. Use verbose mode for comprehensive analysis
2. Parse `verbose_patterns` for full threat model
3. Display attack scenarios and mitigations to reviewers
4. Link to external references (CWE, OWASP)
5. Present vulnerable vs. secure code examples side-by-side

## API Stability

This response format is considered **stable** as of v1.0.0.

Breaking changes will:
- Increment major version
- Be announced in release notes
- Include migration guide

Non-breaking additions (new optional fields) may be added in minor versions.
