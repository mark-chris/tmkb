# TMKB - Claude Configuration

## Project Overview

**Threat Model Knowledge Base (TMKB)** is a security context source that AI agents, tools, and humans can consume—starting with threat patterns for authorization enforcement in multi-tenant applications.

TMKB is the first component of a larger vision: a **Security Context Plane for AI-assisted development**—infrastructure that provides structured security knowledge to any tool in the development workflow.

### Core Mental Model

> **Authorization failures occur at boundaries, not functions.**

This mental model shapes everything in TMKB:
- Patterns focus on trust boundary transitions (endpoint → job, service → service, tenant → tenant)
- Code examples show where authorization context is lost or inconsistent
- Mitigations address the boundary, not just the check

### What This Project IS

- A security context source for AI coding agents
- Threat patterns for authorization enforcement failures
- Structured, machine-queryable security knowledge
- Infrastructure, not just a database

### What This Project IS NOT

- Authentication guidance (JWT, OAuth, sessions) — that's Phase 2
- RBAC/ABAC implementation tutorials
- A vulnerability scanner or detection tool
- General security documentation reformatted

## Technology Stack

| Component | Technology | Notes |
|-----------|------------|-------|
| Knowledge Base | YAML | Human-readable, easy to edit |
| CLI Tool | Go | Fast startup, single binary |
| MCP Server | Go | Model Context Protocol for agent integration |
| Query Engine | Embedded (in-memory) | No external dependencies |
| Index | Tag-based + full-text | Simple, sufficient for MVP |

### Go Version
- Go 1.25+

### Key Dependencies
- Standard library for most functionality
- MCP protocol implementation (custom)
- YAML parsing (gopkg.in/yaml.v3)

## Repository Structure

```
tmkb/
├── cmd/
│   └── tmkb/
│       └── main.go           # CLI entrypoint
├── internal/
│   ├── knowledge/
│   │   ├── loader.go         # YAML loading
│   │   ├── index.go          # Query indexing
│   │   ├── query.go          # Query execution
│   │   ├── output.go         # Agent vs human formatting
│   │   └── validate.go       # Pattern validation
│   ├── cli/
│   │   ├── query.go          # Query command
│   │   ├── get.go            # Get command
│   │   ├── validate.go       # Validate patterns command
│   │   └── serve.go          # MCP server command
│   └── mcp/
│       ├── server.go         # MCP protocol
│       └── handlers.go       # Tool handlers
├── patterns/
│   └── authorization/        # MVP: Authorization patterns only
│       ├── tier-a/           # Full-depth patterns (5 max)
│       └── tier-b/           # Essential coverage (5-7)
├── validation/
│   ├── INVARIANTS.md
│   ├── PROTOCOL.md
│   └── smoke-test/
├── docs/
└── examples/
```

## Pattern Schema

### Pattern Tiers

**Tier A (Full Depth):** 5 patterns max
- Complete schema with all fields
- Multiple code examples (vulnerable + secure)
- Detailed attack scenarios
- `generalizes_to` field required

**Tier B (Essential):** 5-7 patterns
- Core fields only
- Single code example
- Brief attack scenario

### Pattern ID Format
```
TMKB-AUTHZ-{NUMBER}
```

Example: `TMKB-AUTHZ-001`

### Required Fields (Tier A)
```yaml
threat_pattern:
  id: string              # TMKB-AUTHZ-XXX
  name: string            # Human-readable name
  tier: "A" | "B"
  version: string         # Semver
  category: "authorization"
  subcategory: string     # e.g., "async-boundaries"
  language: "python"      # MVP: Python only
  framework: "flask"      # MVP: Flask only
  severity: "critical" | "high" | "medium" | "low"
  likelihood: "high" | "medium" | "low"
  generalizes_to: list    # Other frameworks/languages
  provenance: object      # Source and references
  triggers: object        # Keywords, actions, file patterns
  differentiation: object # LLM blindspots
  description: string     # Full description
  agent_summary: object   # <100 tokens, structured
  attack_scenario: object # Narrative + steps
  mitigations: list       # With code examples
  security_principles: list
  related_patterns: list
```

### Agent Summary Format
```yaml
agent_summary:
  threat: string   # One sentence: what can go wrong
  check: string    # One sentence: what to verify
  fix: string      # One sentence: how to fix
```

## Code Style Guidelines

### Go Code

1. **Package Organization**
   - `cmd/` for entrypoints only
   - `internal/` for all business logic
   - No `pkg/` directory — this is not a library

2. **Error Handling**
   - Return errors, don't panic
   - Wrap errors with context: `fmt.Errorf("loading pattern %s: %w", id, err)`
   - Use structured errors for user-facing messages

3. **Naming Conventions**
   - Patterns: `TMKB-AUTHZ-XXX`
   - Files: lowercase with hyphens for patterns, lowercase for Go
   - Functions: CamelCase, exported when needed

4. **Output Formatting**
   - Default: Agent-optimized JSON (<500 tokens)
   - `--verbose`: Human-readable with full context
   - Never prose paragraphs in agent output

### YAML Patterns

1. **Use literal block scalars for multi-line content**
   ```yaml
   description: |
     Multi-line description goes here.
     Second line.
   ```

2. **Code examples must be syntactically valid**
   - Test all vulnerable and secure code examples
   - Include necessary imports in examples

3. **Provenance is required**
   - All patterns must have `provenance.source_type`
   - Public references (CWE, OWASP) required

## CLI Commands

```bash
# Query patterns by context
tmkb query --context "background job" --language python --framework flask

# Query with verbose output for humans
tmkb query --context "multi-tenant API" --verbose

# Get specific pattern by ID
tmkb get TMKB-AUTHZ-001

# Validate all patterns
tmkb validate --all

# Start MCP server
tmkb serve --port 3000
```

## Output Design Rules

1. Default output fits in single agent context window chunk
2. No prose paragraphs in agent output—bullet points and code only
3. Deterministic ordering: severity → likelihood → alphabetical
4. Hard cap: 3 patterns per response unless explicitly requested

## Validation Invariants

These are non-negotiable pass/fail checks:

| ID | Invariant | Description |
|----|-----------|-------------|
| INV-1 | Auth on mutating endpoints | Authorization check exists on every mutating endpoint |
| INV-2 | Server-side ownership | Object ownership is validated server-side |
| INV-3 | List/detail consistency | Authorization logic consistent between list and detail |
| INV-4 | Background job re-auth | Background jobs re-validate authorization |

## Scope Boundaries

### IN Scope (Authorization Enforcement)
- Where authorization checks are enforced
- When authorization context is lost
- How authorization fails at boundaries
- Object + tenant access validation
- Trust boundary transitions

### OUT of Scope
- Authentication (who the user is)
- RBAC/ABAC design patterns
- Permission management UIs
- Policy languages (OPA, Cedar)
- Identity provider integration

**One-sentence boundary:** TMKB covers *enforcement failure modes*, not *implementation guidance*.

## Testing

### Pattern Validation
- All YAML patterns must pass schema validation
- Vulnerable code must actually be vulnerable
- Secure code must actually mitigate the threat

### Integration Testing
```bash
# Run validation smoke test
task validate

# Test MCP server
task test-mcp
```

## Common Tasks

### Adding a New Pattern

1. Determine tier (A or B) based on architectural depth
2. Create YAML file in `patterns/authorization/tier-{a|b}/`
3. Use pattern ID format: `TMKB-AUTHZ-{NEXT_NUMBER}`
4. Include all required fields for the tier
5. Test code examples for correctness
6. Run `tmkb validate --all`

### Modifying Output Format

Output formatting lives in `internal/knowledge/output.go`. Two modes:
- `FormatAgent()`: Concise JSON, <500 tokens
- `FormatHuman()`: Verbose, educational

### MCP Server Development

MCP implementation is in `internal/mcp/`. The server exposes one tool:
- `tmkb_query`: Query patterns by context

## Development Commands

```bash
task setup     # Install dependencies
task build     # Build binary to bin/tmkb
task test      # Run all tests
task validate  # Validate all patterns
task serve     # Start MCP server (dev mode)
```

## Important Context for AI Assistants

1. **This is a portfolio/demonstration project** — quality over quantity
2. **Target audience is AI coding agents** — output must be machine-consumable
3. **Authorization, not authentication** — maintain scope discipline
4. **Patterns must be architectural** — cross-boundary failures, not syntax issues
5. **10-12 patterns total for MVP** — 5 Tier A + 5-7 Tier B
6. **Flask/Python only for MVP** — other frameworks are Phase 2

## References

- [Project Document](/mnt/project/tmkb-project-document-v5-final.md) - Full project specification
- [MCP Specification](https://modelcontextprotocol.io/)
- [OWASP API Security Top 10](https://owasp.org/API-Security/)
- [CWE-862: Missing Authorization](https://cwe.mitre.org/data/definitions/862.html)
- [CWE-863: Incorrect Authorization](https://cwe.mitre.org/data/definitions/863.html)
