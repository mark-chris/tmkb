# Contributing to TMKB

Thank you for your interest in contributing to the Threat Model Knowledge Base.

## Getting Started

### Prerequisites

- Go 1.25+
- Git

### Setup

```bash
git clone https://github.com/mark-chris/tmkb.git
cd tmkb
go build -o bin/tmkb ./cmd/tmkb
go test ./...
```

### Verify your setup

```bash
./bin/tmkb validate --all
./bin/tmkb list
./bin/tmkb query --context "background job processing"
```

## Adding a New Threat Pattern

Patterns live in `patterns/authorization/` under `tier-a/` or `tier-b/` directories.

### Choosing a Tier

| Tier | Depth | Requirements |
|------|-------|-------------|
| **A** | Full depth | Attack scenario, code examples (vulnerable + secure), security principles, testing guidance, generalization list |
| **B** | Essential coverage | Agent summary, mitigations with code examples, triggers |

Use Tier A for patterns where you can demonstrate a concrete, validated LLM blindspot with before/after code. Use Tier B for known authorization anti-patterns that need concise coverage.

### Pattern File Structure

Create a new YAML file: `patterns/authorization/tier-{a,b}/TMKB-AUTHZ-NNN.yaml`

Use the next available ID number. The file must be wrapped in a top-level `threat_pattern:` key.

#### Required Fields (All Tiers)

```yaml
threat_pattern:
  id: "TMKB-AUTHZ-NNN"
  name: "Descriptive Name"
  tier: "A"  # or "B"
  version: "1.0.0"
  last_updated: "YYYY-MM-DD"

  category: "authorization"
  subcategory: "relevant-subcategory"
  language: "python"
  framework: "flask"

  severity: "critical|high|medium|low"
  likelihood: "high|medium|low"

  provenance:
    source_type: "generalized_observation"
    description: "Where this pattern comes from"
    public_references:
      - cwe: "CWE-NNN"
        name: "CWE Name"
        url: "https://cwe.mitre.org/..."

  triggers:
    keywords: ["keyword1", "keyword2"]
    actions: ["creating something", "implementing something"]
    file_patterns: ["**/relevant_files.py"]

  differentiation:
    llm_knowledge_state: "What LLMs already know"
    tmkb_value: "What TMKB adds beyond that"
    llm_blindspots:
      - "Specific thing LLMs get wrong"

  description: |
    Multi-paragraph description of the threat.

  agent_summary:
    threat: "One-line threat description"
    check: "What to verify"
    fix: "How to fix it"

  mitigations:
    - id: "MIT-AUTHZ-NNNa"
      description: "How to mitigate"
      effectiveness: "high|medium|low"
      implementation_effort: "high|medium|low"
      code_examples:
        - language: "python"
          framework: "flask"
          description: "What this example shows"
          vulnerable_code: |
            # VULNERABLE: explanation
          secure_code: |
            # SECURE: explanation
```

#### Additional Fields for Tier A

Tier A patterns additionally require:

- `attack_scenario` with narrative, preconditions, attack_steps, and impact
- `security_principles` listing general principles this pattern illustrates
- `generalizes_to` listing other frameworks where this pattern applies
- `testing` with manual_verification and automated_checks
- `validation.baseline_test` documenting LLM behavior without TMKB

See `TMKB-AUTHZ-001.yaml` as the reference implementation for a complete Tier A pattern.

### Agent Summary Guidelines

The `agent_summary` is the most critical field — it's what AI agents consume. Keep it under 100 tokens total across all three fields:

- **threat**: What goes wrong (one sentence)
- **check**: What to verify in code (one sentence)
- **fix**: How to fix it (one sentence)

### Validation

Every pattern must pass validation before merge:

```bash
# Validate a specific pattern
./bin/tmkb validate --all

# Verify it appears in queries
./bin/tmkb query --context "your pattern's trigger keywords"
```

Validation checks:
- All required fields are present and non-empty
- Tier is `A` or `B`
- Severity is `critical`, `high`, `medium`, or `low`
- Agent summary exists with threat, check, and fix
- At least one mitigation with ID and description
- Tier A patterns have attack scenarios and code examples

## Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/knowledge/...
go test ./internal/cli/...
go test ./internal/mcp/...

# With verbose output
go test -v ./...
```

## Pull Request Process

1. Create a feature branch from `main`
2. Make your changes
3. Run `go test ./...` and `./bin/tmkb validate --all`
4. Submit a PR with a clear description of what the pattern covers and why it matters

## What Makes a Good Pattern

- **Demonstrates a real LLM blindspot**: The pattern should address something LLMs consistently get wrong, not just general security advice
- **Actionable**: The agent summary should give an AI agent enough context to generate secure code
- **Grounded**: Include CWE/OWASP references where applicable
- **Differentiated**: Explain what LLMs already know vs. what TMKB adds — if an LLM already handles this well, it doesn't belong here

## Scope

TMKB focuses on **authorization patterns in multi-tenant applications** — specifically architectural-level threats that cross system boundaries. Out of scope:

- Syntax-level vulnerabilities (SQL injection, XSS) — LLMs handle these well already
- Authentication (login, session management) — related but distinct domain
- Infrastructure security (network, cloud config) — different layer
- Non-authorization application security — future expansion possible
