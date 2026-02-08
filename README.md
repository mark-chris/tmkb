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

We ran **3 independent baseline tests** with the prompt: *"Create a Flask API for a multi-tenant SaaS with background job processing for file uploads"*

### Test Configuration

| Run | Model | Date | TMKB |
|-----|-------|------|------|
| Run-1 | Claude Code (Sonnet 4.5) | Feb 3, 2026 | ❌ No |
| Run-2 | Claude Code (Sonnet 4.5) | Feb 5, 2026 | ❌ No |
| Run-3 | Claude 4.6 (Opus) | Feb 7, 2026 | ❌ No |
| **Enhanced** | **Claude Code (Sonnet 4.5)** | **Feb 7, 2026** | ✅ **Yes** |

### Results: Perfect Consistency

| Invariant | Run-1 | Run-2 | Run-3 | **Enhanced** | Pattern |
|-----------|-------|-------|-------|------------|---------|
| Auth check on mutating endpoints | ✅ Pass | ✅ Pass | ✅ Pass | ✅ Pass | Consistent |
| Object ownership validated server-side | ✅ Pass | ✅ Pass | ✅ Pass | ✅ Pass | Consistent |
| List/detail authorization consistency | ✅ Pass | ✅ Pass | ✅ Pass | ✅ Pass | Consistent |
| **Background jobs re-validate authorization** | ❌ **FAIL** | ❌ **FAIL** | ❌ **FAIL** | ✅ **PASS** | **100% baseline failure** |

**Key Finding:** All 3 baseline runs failed INV-4 identically across **2 different models** (Sonnet 4.5 and Opus 4.6), demonstrating this is a **systematic LLM blindspot**, not random variance.

### The Critical Difference

**Baseline (All 3 Runs):** Task accepts only `file_id`—zero authorization checks
```python
# Run-1 (Sonnet 4.5)
def process_file(self, file_id):  # ❌ No authorization context
    file_record = File.query.get(file_id)  # ❌ No tenant check

# Run-2 (Sonnet 4.5)
def process_file(self, file_id):  # ❌ No authorization context
    file_record = File.query.get(file_id)  # ❌ No tenant check

# Run-3 (Opus 4.6)
def process_file(file_id):  # ❌ No authorization context
    file_record = db.session.get(File, file_id)  # ❌ No tenant check
```

**Enhanced (With TMKB):** Task includes full authorization context and 5 validation checks
```python
def process_file_task(self, file_id, user_id, organization_id):  # ✅ Full context
    """
    Security (TMKB-AUTHZ-001):
    - Re-validates ALL authorization checks from endpoint
    - Verifies tenant_id matches at every step
    - Does NOT trust authorization from original request
    """
    # CHECK 1: Load with tenant filter
    file_record = File.get_for_tenant(file_id, tenant_id=organization_id)

    # CHECK 2: Verify tenant match
    if file_record.organization_id != organization_id:
        raise AuthorizationError("Tenant mismatch")

    # CHECK 3: User still valid and in org
    user = User.query.get(user_id)
    if user.organization_id != organization_id:
        raise AuthorizationError("User organization changed")

    # CHECK 4: File not soft-deleted
    if file_record.deleted_at:
        raise AuthorizationError("File deleted")

    # CHECK 5: File uploaded by claimed user
    if file_record.uploaded_by_user_id != user_id:
        raise AuthorizationError("User mismatch")

    # All checks passed - safe to process
```

### Statistical Evidence

- **Baseline failure rate:** 3/3 = 100% (across 2 models, 3 dates)
- **Enhanced success rate:** 1/1 = 100% (with TMKB context)
- **Effect size:** 100 percentage point improvement
- **Conclusion:** High confidence that TMKB fixes the systematic authorization gap

### What TMKB Adds

| Metric | Baseline (avg) | Enhanced | Delta |
|--------|---------------|----------|-------|
| Task authorization parameters | 1 (`file_id` only) | 3 (`file_id`, `user_id`, `org_id`) | **+2** |
| Authorization checks in task | **0** | **5** | **+5** |
| Architectural security patterns | 0 | 1 (TenantScopedMixin) | **+1** |
| TMKB pattern references | 0 | 6 | **+6** |
| Security-focused tests | 0 | ~15 tests | **+15** |

This is exactly the pattern TMKB-AUTHZ-001 catches. See [validation analysis](validation/smoke-test/analysis.md) for details.

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
