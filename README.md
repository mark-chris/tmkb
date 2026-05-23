# Threat Model Knowledge Base (TMKB)

> A CLI and MCP server that feeds AI coding agents structured threat patterns, so they generate code that holds up at architectural security boundaries — starting with authorization in multi-tenant apps.

[![CI](https://github.com/mark-chris/tmkb/workflows/CI/badge.svg)](https://github.com/mark-chris/tmkb/actions)
[![CodeQL](https://github.com/mark-chris/tmkb/workflows/CodeQL/badge.svg)](https://github.com/mark-chris/tmkb/security/code-scanning)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Project](https://img.shields.io/badge/Project-TMKB%20MVP-blue)](https://github.com/users/mark-chris/projects/2)

> **Status:** MVP / research preview. The pattern set, query API, and MCP integration are stable enough to use; the validation methodology is the primary thing being iterated on.

When an agent asks TMKB about a context like "background job processing," it gets back a compact, structured pattern it can act on:

```json
{
  "id": "TMKB-AUTHZ-001",
  "severity": "high",
  "threat": "Background jobs execute without the authorization context from the original request",
  "check": "Verify authorization is re-checked in the job, not just the endpoint",
  "fix": "Pass user_id and tenant_id to job; re-validate permissions before operating on resources"
}
```

In baseline tests, 8/8 runs across 3 providers and 5 models failed to re-authorize at async boundaries. With TMKB context, the same prompt passes — adding 5 authorization checks and full tenant context to the generated background task.

TMKB ships 12 authorization patterns today (5 tier-A, 7 tier-B) and is the first component of what we're calling a Security Context Plane for AI-assisted development.

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

We ran **8 independent baseline tests** across **3 providers**, **5 models**, **2 application types**, and **2 extended-thinking configurations**:
- **Runs 1-5, 7, 8:** *"Create a Flask API for a multi-tenant SaaS with background job processing for file uploads"*
- **Run-6:** *"Create a Flask API that receives webhooks from external services and processes them asynchronously"*
- **Run-7:** Re-tests Opus 4.6 with `CLAUDE_CODE_EFFORT_LEVEL=high` (maximum extended thinking)
- **Run-8:** Re-tests with Claude Opus 4.7, the newest Claude release

### Test Configuration

| Run | Model | Provider | Application | Date | Thinking | TMKB |
|-----|-------|----------|-------------|------|----------|------|
| Run-1 | Claude Sonnet 4.5 | Anthropic | File upload | Feb 3, 2026 | Default | ❌ No |
| Run-2 | Claude Sonnet 4.5 | Anthropic | File upload | Feb 5, 2026 | Default | ❌ No |
| Run-3 | Claude Opus 4.6 | Anthropic | File upload | Feb 7, 2026 | Default | ❌ No |
| Run-4 | GPT-5.2 | OpenAI | File upload | Feb 8, 2026 | N/A | ❌ No |
| Run-5 | Gemini | Google | File upload | Feb 8, 2026 | N/A | ❌ No |
| Run-6 | Claude Sonnet 4.5 | Anthropic | **Webhook** | Feb 8, 2026 | Default | ❌ No |
| Run-7 | Claude Opus 4.6 | Anthropic | File upload | Feb 25, 2026 | **High** | ❌ No |
| Run-8 | **Claude Opus 4.7** | Anthropic | File upload | May 14, 2026 | Default | ❌ No |
| **Enhanced** | **Claude Sonnet 4.5** | **Anthropic** | **File upload** | **Feb 7, 2026** | Default | ✅ **Yes** |

### Results: Async Boundary Fails 100% Across All Providers and Application Types

| Invariant | Run-1 | Run-2 | Run-3 | Run-4 | Run-5 | Run-6 | Run-7 | Run-8 | **Enhanced** |
|-----------|-------|-------|-------|-------|-------|-------|-------|-------|------------|
| Auth on mutating endpoints | ✅ | ✅ | ✅ | ✅ | ❌ | ✅¹ | ✅ | ✅ | ✅ |
| Object ownership validated | ✅ | ✅ | ✅ | ✅ | ❌ | ❌² | ✅ | ✅ | ✅ |
| List/detail consistency | ✅ | ✅ | ✅ | ✅ | ❌ | N/A | ✅ | ✅ | ✅ |
| **Async boundary re-auth** | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ **PASS** |

¹ Run-6 uses webhook-specific invariants: W-INV-1 (origin verification) partial pass — GitHub HMAC correct, Stripe checks header presence only
² W-INV-2 (payload distrust) fail — all tasks blindly trust webhook payloads

**Key Finding:** All 8 baseline runs failed the async boundary invariant across **3 providers** (Anthropic, OpenAI, Google), **5 models**, **2 application types** (file upload, webhooks), and **2 extended-thinking configurations** (default and `high`), demonstrating this is a **systematic, provider-invariant, model-invariant, application-type-invariant, and reasoning-budget-invariant LLM blindspot**.

Run-6 confirms the pattern generalizes across application types: webhook signature verification at the HTTP boundary is not propagated to Celery workers, just as user authentication is not propagated to background jobs in runs 1-5. **Run-7 eliminates reasoning budget as a confound** — even with `CLAUDE_CODE_EFFORT_LEVEL=high`, the Opus 4.6 model produces the same INV-4 violation. **Run-8 extends this to the newest Claude release** — Opus 4.7 reproduces the identical task signature `(self, file_id)` with zero authorization checks, confirming a generational model upgrade does not surface the missing pattern.

### The Critical Difference

**Baseline (All 6 Runs):** Task accepts only resource ID or raw payload — zero re-authorization
```python
# Runs 1-5 (File Upload): Task accepts only file_id
def process_file(self, file_id):                   # ❌ No user/org context
    file_record = File.query.get(file_id)

# Run-6 (Webhooks): Task accepts raw payload, no origin re-verification
def process_github_webhook(data):                  # ❌ No signature re-check
    event_type = data.get('action')
def process_stripe_webhook(data):                  # ❌ No signature re-check
    event_type = data.get('type')
```

**Enhanced (With TMKB):** Task includes full authorization context and 5 validation checks
```python
def process_file_task(self, file_id, user_id, organization_id):  # ✅ Full context
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
```

### Statistical Evidence

- **Baseline async boundary failure rate:** 8/8 = 100% (3 providers, 5 models, 2 app types, 2 thinking configs)
- **95% confidence interval:** [63.1%, 100%] (Wilson score)
- **Enhanced success rate:** 1/1 = 100% (with TMKB context)
- **Effect size:** 100 percentage point improvement
- **Probability of 8/8 failures under a ≤50% baseline-failure null:** 0.39% (p < 0.005)

### What TMKB Adds

| Metric | Baseline (avg) | Enhanced | Delta |
|--------|---------------|----------|-------|
| Task authorization parameters | 1 (`file_id` only) | 3 (`file_id`, `user_id`, `org_id`) | **+2** |
| Authorization checks in task | **0** | **5** | **+5** |
| Architectural security patterns | 0 | 1 (TenantScopedMixin) | **+1** |
| TMKB pattern references | 0 | 6 | **+6** |
| Security-focused tests | 0 | ~15 tests | **+15** |

See [cross-run comparison](validation/smoke-test/baseline-cross-run-comparison.md) and individual run analyses in [validation/smoke-test/baseline/](validation/smoke-test/baseline/) for details.

## Why This Project Exists

- **Threat modeling at architectural level** — Not syntax-level security; focuses on where authorization actually breaks across system boundaries
- **Encoding tacit security judgment** — Captures design review expertise as structured, queryable data
- **Understanding AI agent failure modes** — Documents what LLMs get wrong and provides corrective context
- **Pragmatic validation methodology** — Includes baseline tests proving LLMs produce better code with TMKB
- **Infrastructure thinking** — First component of a Security Context Plane for AI-assisted development

## Installation

### Prerequisites

- **Go 1.25+**: [Download Go](https://go.dev/dl/)
- **Git**: For cloning the repository

### From Source

```bash
# Clone
git clone https://github.com/mark-chris/tmkb.git
cd tmkb

# Build
go build -o bin/tmkb ./cmd/tmkb

# (Optional) Install to GOPATH
go install ./cmd/tmkb
```

Or using [Task](https://taskfile.dev/):
```bash
task setup && task build
```

## Quick Start

### CLI Usage

```bash
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
- **TMKB-AUTHZ-002**: List/Detail Authorization Inconsistency
- **TMKB-AUTHZ-003**: Soft-Delete Resurrection Attack
- **TMKB-AUTHZ-004**: Tenant Isolation via Application Logic
- **TMKB-AUTHZ-005**: User/Account/Resource Ownership Confusion

### Tier B (Essential Coverage)
- **TMKB-AUTHZ-006**: Mass Assignment of Ownership Fields
- **TMKB-AUTHZ-007**: Insecure Direct Object Reference via Sequential IDs
- **TMKB-AUTHZ-008**: Authorization Bypass via HTTP Method Override
- **TMKB-AUTHZ-009**: State Transition Authorization Bypass
- **TMKB-AUTHZ-010**: Unauthorized Access via Relationship Traversal
- **TMKB-AUTHZ-011**: Authorization Check in Wrong Layer
- **TMKB-AUTHZ-012**: Inconsistent Authorization in Bulk Operations

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
│   └── mcp/               # MCP server
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
