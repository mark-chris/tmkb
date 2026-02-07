# Threat Model Knowledge Base (TMKB)

> A security context source that agents, tools, and humans can consume—starting with threat patterns for authorization enforcement in multi-tenant applications.

[![CI](https://github.com/mark-chris/tmkb/workflows/CI/badge.svg)](https://github.com/mark-chris/tmkb/actions)
[![CodeQL](https://github.com/mark-chris/tmkb/workflows/CodeQL/badge.svg)](https://github.com/mark-chris/tmkb/security/code-scanning)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Project](https://img.shields.io/badge/Project-TMKB%20MVP-blue)](https://github.com/users/mark-chris/projects/2)

## What This Project Demonstrates

- **Threat modeling at architectural level** — Not syntax-level security; focuses on where authorization actually breaks across system boundaries
- **Encoding tacit security judgment** — Captures design review expertise as structured, queryable data
- **Understanding AI agent failure modes** — Documents what LLMs get wrong and provides corrective context
- **Pragmatic validation methodology** — Includes baseline tests proving LLMs produce better code with TMKB
- **Infrastructure thinking** — First component of a Security Context Plane for AI-assisted development

## The Problem

Modern LLMs have substantial security knowledge for well-documented, syntax-level vulnerabilities (SQL injection, XSS, JWT algorithm confusion). However, **LLMs are systematically weak at architectural security patterns** that require reasoning across system boundaries.

| LLM Capability | LLM Limitation |
|----------------|----------------|
| Knows OWASP Top 10 | Doesn't reason across call paths |
| Can implement RBAC given requirements | Assumes "authenticated = authorized" |
| Generates endpoint-local auth checks | Misses inconsistencies between endpoints |
| Knows about IDOR | Doesn't model business object ownership |
| Can write middleware | Doesn't consider background job context loss |

**TMKB ensures that an AI coding agent generating a multi-tenant API will always consider authorization boundaries that span beyond the current function.**

## Validation Results

We tested Claude Code with the prompt: *"Create a Flask API for a multi-tenant SaaS with background job processing for file uploads"*

| Invariant | Without TMKB | With TMKB |
|-----------|--------------|-----------|
| Auth check on mutating endpoints | ✅ Pass | ✅ Pass |
| Object ownership validated server-side | ✅ Pass | ✅ Pass |
| List/detail authorization consistency | ✅ Pass | ✅ Pass |
| **Background jobs re-validate authorization** | ❌ **FAIL** | ✅ Pass |

The generated Celery task accepted only `file_id`—no user context, no tenant validation. This is exactly the pattern TMKB-AUTHZ-001 catches.

## Quick Start

### CLI Usage

```bash
# Build
go build -o bin/tmkb ./cmd/tmkb

# Query patterns by context (agent mode - default)
./bin/tmkb query --context "background job processing"

# Query with verbose output (human-readable)
./bin/tmkb query --context "background job processing" --verbose

# Query with filters
./bin/tmkb query --context "file upload" --language python --framework flask

# Get a specific pattern
./bin/tmkb get TMKB-AUTHZ-001

# List all patterns
./bin/tmkb list

# Validate patterns
./bin/tmkb validate --all
```

### Example Output (Agent Mode)

```json
{
  "pattern_count": 1,
  "patterns_included": 1,
  "token_count": 74,
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
    "secure_template": "..."
  }
}
```

**Output Modes:**
- **Agent mode (default)**: Token-limited (<500 tokens), max 3 patterns, JSON only
- **Verbose mode** (`--verbose`): Unlimited tokens, max 10 patterns, comprehensive details

See [Query Response Format Documentation](docs/api/query-response-format.md) for complete API specification.

## MCP Integration

TMKB provides a Model Context Protocol (MCP) server for AI coding assistants like Claude Code.

### Quick Start with Claude Code

1. **Build TMKB**:
   ```bash
   go build -o tmkb cmd/tmkb/main.go
   ```

2. **Configure Claude Code**:
   Add to `~/.claude/mcp_settings.json`:
   ```json
   {
     "mcpServers": {
       "tmkb": {
         "command": "/path/to/tmkb",
         "args": ["serve"]
       }
     }
   }
   ```

3. **Restart Claude Code** and ask:
   > Query TMKB for authorization security threats

See [MCP Integration Guide](docs/mcp-integration.md) for detailed setup and troubleshooting.

## Pattern Coverage (MVP)

### Tier A (Full Depth)
- **TMKB-AUTHZ-001**: Background Job Authorization Context Loss
- TMKB-AUTHZ-002: List/Detail Authorization Inconsistency *(planned)*
- TMKB-AUTHZ-003: Soft-Delete Resurrection Attack *(planned)*
- TMKB-AUTHZ-004: Tenant Isolation via Application Logic *(planned)*
- TMKB-AUTHZ-005: User/Account/Resource Ownership Confusion *(planned)*

### Tier B (Essential Coverage)
- Additional patterns for common authorization failures *(planned)*

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│           Threat Model Knowledge Base                   │
│         (Security Context Plane - v1)                   │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │   Threat    │  │   Attack    │  │ Mitigation  │     │
│  │  Patterns   │  │  Scenarios  │  │  Patterns   │     │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘     │
│         │                │                │            │
│         └────────────────┼────────────────┘            │
│                          │                             │
│                   ┌──────▼──────┐                      │
│                   │   Indexed   │                      │
│                   │  by Context │                      │
│                   └──────┬──────┘                      │
│                          │                             │
├──────────────────────────┼─────────────────────────────┤
│                          │                             │
│  ┌───────────┐    ┌──────▼──────┐   ┌───────────┐     │
│  │    CLI    │    │  Query API  │   │    MCP    │     │
│  │   Tool    │    │   (Local)   │   │  Server   │     │
│  └───────────┘    └─────────────┘   └───────────┘     │
│                                                        │
└────────────────────────────────────────────────────────┘
```

## Project Structure

```
tmkb/
├── cmd/tmkb/              # CLI entrypoint
├── internal/
│   ├── knowledge/         # Core domain logic
│   │   ├── types.go       # Data model
│   │   ├── loader.go      # YAML loading
│   │   ├── index.go       # Query indexing
│   │   ├── query.go       # Query execution
│   │   ├── output.go      # Formatting
│   │   └── validate.go    # Pattern validation
│   ├── cli/               # CLI commands
│   └── mcp/               # MCP server (WIP)
├── patterns/
│   └── authorization/
│       ├── tier-a/        # Full-depth patterns
│       └── tier-b/        # Essential patterns
├── validation/            # Test protocols and results
└── docs/                  # Documentation
```

## The Core Mental Model

> **Authorization failures occur at boundaries, not functions.**

This mental model shapes everything in TMKB:
- Patterns focus on trust boundary transitions (endpoint → job, service → service, tenant → tenant)
- Code examples show where authorization context is lost or inconsistent
- Mitigations address the boundary, not just the check

## Contributing

See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for guidelines.

## License

MIT License - See [LICENSE](LICENSE) for details.

## References

- [OWASP API Security Top 10 2023](https://owasp.org/API-Security/editions/2023/en/0xa1-broken-object-level-authorization/)
- [CWE-862: Missing Authorization](https://cwe.mitre.org/data/definitions/862.html)
- [Model Context Protocol](https://modelcontextprotocol.io/)
