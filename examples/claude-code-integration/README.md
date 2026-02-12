# Claude Code Integration Example

Set up TMKB as an MCP tool in Claude Code so it automatically provides authorization security context during code generation.

## Setup

### 1. Build TMKB

```bash
git clone https://github.com/mark-chris/tmkb.git
cd tmkb
go build -o bin/tmkb ./cmd/tmkb

# Verify it works
./bin/tmkb list
```

### 2. Configure Claude Code

Copy the example config into your Claude Code MCP settings:

```bash
# Find your absolute path
realpath bin/tmkb
```

Edit `~/.claude/mcp_settings.json` (create it if it doesn't exist):

```json
{
  "mcpServers": {
    "tmkb": {
      "command": "/home/you/tmkb/bin/tmkb",
      "args": ["serve"]
    }
  }
}
```

Replace `/home/you/tmkb/bin/tmkb` with the actual absolute path from `realpath` above.

### 3. Restart Claude Code

Quit and reopen Claude Code. You should see `tmkb` listed as an available MCP server.

### 4. Verify

Ask Claude Code:

> Query TMKB for authorization security threats

It should invoke the `tmkb_query` tool and return threat patterns.

## Example Prompts and Expected Outputs

### Background Job Authorization

**Prompt:** "Create a Flask API for a multi-tenant SaaS with background job processing for file uploads"

**What TMKB provides:** Claude Code will query TMKB with context "background job processing" and receive:

```json
{
  "pattern_count": 1,
  "patterns_included": 1,
  "token_count": 64,
  "patterns": [
    {
      "id": "TMKB-AUTHZ-001",
      "severity": "high",
      "threat": "Background jobs execute without the authorization context from the original request",
      "check": "Verify authorization is re-checked in the job, not just the endpoint",
      "fix": "Pass user_id and tenant_id to job; re-validate permissions before operating on resources"
    }
  ]
}
```

**Result:** The generated Celery task will accept `user_id` and `organization_id` parameters and re-validate authorization before processing, instead of accepting only a `file_id`.

### Multi-Tenant API

**Prompt:** "Add a search endpoint to the project management API"

**What TMKB provides:** Claude Code will query with context "multi-tenant API endpoint" and receive:

```json
{
  "pattern_count": 1,
  "patterns_included": 1,
  "token_count": 74,
  "patterns": [
    {
      "id": "TMKB-AUTHZ-004",
      "severity": "critical",
      "threat": "Missing tenant_id filter in ANY query exposes all tenants' data; application-level isolation is fragile",
      "check": "Verify EVERY query includes tenant filter, including joins, lookups, and background jobs",
      "fix": "Use base query class with automatic tenant filter; never use Model.query.get() directly"
    }
  ]
}
```

**Result:** The generated search endpoint will include tenant filtering in the query, and the response will include a `TenantScopedMixin` pattern for consistent isolation across all endpoints.

### Bulk Operations

**Prompt:** "Add a bulk delete endpoint for cleaning up old files"

**What TMKB provides:** Claude Code will query with context "bulk delete operation" and receive patterns for both HTTP method override bypass and bulk authorization:

```json
{
  "patterns": [
    {
      "id": "TMKB-AUTHZ-008",
      "severity": "medium",
      "threat": "HTTP method override allows bypassing method-based authorization or rate limits"
    },
    {
      "id": "TMKB-AUTHZ-012",
      "severity": "high",
      "threat": "Bulk operation endpoints skip per-item authorization that single-item endpoints enforce"
    }
  ]
}
```

**Result:** The generated bulk delete endpoint will verify authorization for every item in the batch, not just check that the user is authenticated.

## How It Works

1. You ask Claude Code to write code involving authorization-sensitive operations
2. Claude Code recognizes the context (background jobs, multi-tenant queries, bulk operations, etc.)
3. Claude Code calls the `tmkb_query` tool with the relevant context
4. TMKB returns concise threat patterns and secure code templates
5. Claude Code incorporates the security guidance into the generated code

The key difference: without TMKB, Claude Code generates code that is functionally correct but architecturally vulnerable. With TMKB, it generates code that handles authorization across system boundaries.

## Troubleshooting

**"Server failed to start"**
- Verify the path is absolute: `realpath bin/tmkb`
- Check the binary is executable: `chmod +x bin/tmkb`
- Test manually: `echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{}}}' | ./bin/tmkb serve`

**Tool not appearing in Claude Code**
- Restart Claude Code completely (quit and reopen)
- Check `~/.claude/mcp_settings.json` is valid JSON
- Look for MCP errors in Claude Code logs

**Queries return 0 patterns**
- The patterns directory must be accessible from where the binary runs
- If you installed TMKB outside the repo, set the `TMKB_PATTERNS` environment variable in the config:

```json
{
  "mcpServers": {
    "tmkb": {
      "command": "/path/to/tmkb",
      "args": ["serve"],
      "env": {
        "TMKB_PATTERNS": "/path/to/tmkb/patterns"
      }
    }
  }
}
```

See the full [MCP Integration Guide](../../docs/mcp-integration.md) for advanced configuration.
