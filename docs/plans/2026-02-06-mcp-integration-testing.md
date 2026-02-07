# MCP Integration Testing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix serve command to start MCP server, test with Claude Code, and document integration.

**Architecture:** Three-phase approach: (1) Fix serve command to call MCP server via stdio, (2) Manual testing with 7 scenarios, (3) Documentation in README + detailed guide.

**Tech Stack:** Go, Cobra CLI, MCP protocol, JSON-RPC 2.0, Claude Code

---

## Task 1: Fix Serve Command Implementation

**Files:**
- Modify: `internal/cli/serve.go:1-47`

**Step 1: Remove port flag and variable**

Update `internal/cli/serve.go`:

Remove lines 9-11:
```go
var (
	servePort int
)
```

Remove lines 30-32:
```go
func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 3000,
		"Port to listen on")
}
```

**Step 2: Update help text**

Replace lines 16-26 with:
```go
	Long: `Start a Model Context Protocol (MCP) server that AI agents can query.

The MCP server communicates via stdin/stdout using the JSON-RPC 2.0 protocol.
It is designed to be invoked by MCP clients like Claude Code.

Examples:
  # Start MCP server (typically invoked by Claude Code)
  tmkb serve`,
```

**Step 3: Update imports**

Replace lines 3-7 with:
```go
import (
	"log"
	"os"

	"github.com/mark-chris/tmkb/internal/mcp"
	"github.com/spf13/cobra"
)
```

**Step 4: Implement runServe function**

Replace lines 35-46 with:
```go
func runServe(cmd *cobra.Command, args []string) error {
	// Create MCP server with the loaded pattern index
	server := mcp.NewServer(index)

	// Log to stderr (stdout is reserved for protocol communication)
	log.SetOutput(os.Stderr)
	log.Printf("Starting MCP server with %d patterns loaded", index.Count())
	log.Println("Server ready for MCP protocol communication via stdio")

	// Run server - blocks until stdin closes (EOF)
	if err := server.ServeStdio(os.Stdin, os.Stdout); err != nil {
		return fmt.Errorf("MCP server error: %w", err)
	}

	log.Println("MCP server shutdown")
	return nil
}
```

**Step 5: Add fmt import back**

Update imports to include fmt (needed for fmt.Errorf):
```go
import (
	"fmt"
	"log"
	"os"

	"github.com/mark-chris/tmkb/internal/mcp"
	"github.com/spf13/cobra"
)
```

**Step 6: Verify code compiles**

Run: `go build ./internal/cli`
Expected: No errors

**Step 7: Test serve command starts**

Run: `go run cmd/tmkb/main.go serve &`
Expected: Log shows "Starting MCP server with N patterns loaded"

Send EOF to stop:
Run: `pkill -f "tmkb serve"`

**Step 8: Commit**

```bash
git add internal/cli/serve.go
git commit -m "feat: implement MCP server in serve command

- Remove unused --port flag (stdio doesn't use ports)
- Call mcp.NewServer(index).ServeStdio(os.Stdin, os.Stdout)
- Log to stderr to avoid interfering with JSON-RPC on stdout
- Connect CLI to MCP protocol implementation

Resolves part of Issue #7

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Manual Testing - Server Startup

**Prerequisites:**
- Build binary: `go build -o tmkb cmd/tmkb/main.go`
- Ensure patterns directory exists

**Step 1: Verify binary works**

Run: `./tmkb --version`
Expected: Version output

Run: `./tmkb list`
Expected: Shows loaded patterns

**Step 2: Test server starts manually**

Run in terminal: `./tmkb serve`
Expected: Logs to stderr:
```
Starting MCP server with N patterns loaded
Server ready for MCP protocol communication via stdio
```

**Step 3: Send initialize request**

In another terminal:
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./tmkb serve
```

Expected: JSON response with:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2025-11-25",
    "capabilities": {
      "tools": {}
    },
    "serverInfo": {
      "name": "tmkb",
      "version": "0.1.0"
    }
  }
}
```

**Step 4: Document test results**

Create: `docs/testing/2026-02-06-mcp-integration-manual-tests.md`

```markdown
# MCP Integration Manual Testing Results

**Date**: 2026-02-06
**Tester**: [Your name]

## Scenario 1: Server Startup and Connection

**Status**: ✅ Pass / ❌ Fail

**Steps Executed**:
1. Built binary: `go build -o tmkb cmd/tmkb/main.go`
2. Started server: `./tmkb serve`
3. Sent initialize request

**Results**:
- [ ] Server started without errors
- [ ] Logged "Starting MCP server with N patterns loaded"
- [ ] Initialize handshake succeeded
- [ ] Returned valid JSON-RPC response

**Notes**:
[Any observations or issues]
```

---

## Task 3: Manual Testing - Claude Code Integration

**Step 1: Get absolute path to tmkb**

Run: `pwd`
Note the path, e.g., `/home/mark/Projects/tmkb/.worktrees/issue-7-mcp-integration`

**Step 2: Create MCP configuration**

Edit/create: `~/.claude/mcp_settings.json`

```json
{
  "mcpServers": {
    "tmkb": {
      "command": "/home/mark/Projects/tmkb/.worktrees/issue-7-mcp-integration/tmkb",
      "args": ["serve"],
      "env": {
        "TMKB_PATTERNS": "/home/mark/Projects/tmkb/patterns"
      }
    }
  }
}
```

**Important**: Replace paths with your actual absolute paths.

**Step 3: Restart Claude Code**

- Completely quit Claude Code (not just close window)
- Reopen Claude Code

**Step 4: Verify server appears**

In Claude Code, check if tmkb server is recognized in settings/MCP section.

Expected: tmkb server shows as active/available

**Step 5: Document connection test**

Update `docs/testing/2026-02-06-mcp-integration-manual-tests.md`:

```markdown
## Claude Code Configuration

**Config File**: `~/.claude/mcp_settings.json`

**Configuration Used**:
```json
{
  "mcpServers": {
    "tmkb": {
      "command": "/absolute/path/to/tmkb",
      "args": ["serve"]
    }
  }
}
```

**Server Status**: ✅ Active / ❌ Failed to start

**Notes**:
[Any configuration issues or observations]
```

---

## Task 4: Manual Testing - Basic Query (Scenario 2)

**Step 1: Open Claude Code chat**

Start a new conversation in Claude Code.

**Step 2: Request TMKB query**

Ask: "Query TMKB for background job authorization threats"

**Step 3: Observe tool call**

Check if Claude Code:
- Recognizes the request requires tmkb_query tool
- Invokes the tool with appropriate parameters
- Displays the tool call in chat

**Step 4: Verify response**

Expected response structure:
```json
{
  "pattern_count": 3,
  "patterns_included": 3,
  "patterns": [
    {
      "id": "TMKB-AUTH-001",
      "severity": "high",
      "threat": "...",
      "check": "...",
      "fix": "..."
    }
  ]
}
```

**Step 5: Document results**

Update `docs/testing/2026-02-06-mcp-integration-manual-tests.md`:

```markdown
## Scenario 2: Basic Query (Success Path)

**Query**: "Query TMKB for background job authorization threats"

**Status**: ✅ Pass / ❌ Fail

**Tool Call**:
```json
{
  "name": "tmkb_query",
  "arguments": {
    "context": "background job authorization",
    ...
  }
}
```

**Response**:
- [ ] Tool call succeeded
- [ ] pattern_count > 0
- [ ] Patterns relevant to query
- [ ] Valid JSON structure

**Patterns Returned**:
[List pattern IDs, e.g., TMKB-AUTH-001]

**Notes**:
[Observations about relevance, response time, etc.]
```

---

## Task 5: Manual Testing - Filtered Query (Scenario 3)

**Step 1: Query with language and framework**

Ask in Claude Code: "Query TMKB for Python Flask background job security"

**Step 2: Verify filters applied**

Check tool call includes:
```json
{
  "context": "...",
  "language": "python",
  "framework": "flask"
}
```

**Step 3: Verify filtering works**

Check returned patterns:
- All should be Python patterns
- Framework should be Flask or "any"

**Step 4: Document results**

Update `docs/testing/2026-02-06-mcp-integration-manual-tests.md`:

```markdown
## Scenario 3: Query with Filters

**Query**: "Query TMKB for Python Flask background job security"

**Status**: ✅ Pass / ❌ Fail

**Tool Call**:
```json
{
  "name": "tmkb_query",
  "arguments": {
    "context": "...",
    "language": "python",
    "framework": "flask"
  }
}
```

**Response**:
- [ ] Only Python patterns returned
- [ ] Only Flask/any framework patterns returned
- [ ] pattern_count reflects filtering

**Notes**:
[Verify all patterns match filters]
```

---

## Task 6: Manual Testing - Validation Errors (Scenarios 4-5)

**Step 1: Test invalid language**

Ask in Claude Code: "Query TMKB for Java security threats"

Expected: Error message "Invalid language 'java'. Supported languages: python"

**Step 2: Test empty context**

Try to invoke with empty/missing context (may require direct tool call if Claude Code prevents it).

Expected: Error message "context must be non-empty"

**Step 3: Verify server stability**

After errors, server should:
- Still be running
- Accept subsequent valid queries
- Not crash or hang

**Step 4: Document error handling**

Update `docs/testing/2026-02-06-mcp-integration-manual-tests.md`:

```markdown
## Scenario 4: Validation Error (Invalid Language)

**Query**: "Query TMKB for Java security threats"

**Status**: ✅ Pass / ❌ Fail

**Response**:
- [ ] Tool returned error (isError: true)
- [ ] Error message: "Invalid language 'java'. Supported languages: python"
- [ ] Server remained operational

**Notes**:
[Claude Code's handling of error]

## Scenario 5: Empty Context Error

**Status**: ✅ Pass / ❌ Fail

**Response**:
- [ ] Validation error caught
- [ ] Error message: "context must be non-empty"
- [ ] Server didn't crash

**Notes**:
[How error was triggered]
```

---

## Task 7: Manual Testing - Multiple Queries (Scenario 6)

**Step 1: Run multiple queries in one session**

In the same Claude Code chat, run:
1. "Query TMKB for multi-tenant isolation"
2. "Query TMKB for background job authorization"
3. "Query TMKB for admin panel security"
4. "Query TMKB for Java security" (invalid - should error)
5. "Query TMKB for Python API security" (valid - should succeed)

**Step 2: Verify stability**

Check:
- All valid queries return results
- Invalid query errors cleanly
- Error doesn't affect subsequent queries
- Server doesn't crash or hang

**Step 3: Document stability test**

Update `docs/testing/2026-02-06-mcp-integration-manual-tests.md`:

```markdown
## Scenario 6: Multiple Queries (Stability)

**Status**: ✅ Pass / ❌ Fail

**Queries Executed**:
1. Multi-tenant isolation - ✅ Pass / ❌ Fail
2. Background job authorization - ✅ Pass / ❌ Fail
3. Admin panel security - ✅ Pass / ❌ Fail
4. Java security (invalid) - ✅ Error / ❌ Fail
5. Python API security - ✅ Pass / ❌ Fail

**Results**:
- [ ] All valid queries succeeded
- [ ] Invalid query returned error
- [ ] Error didn't affect subsequent queries
- [ ] Server remained stable

**Notes**:
[Performance observations, any issues]
```

---

## Task 8: Manual Testing - Server Shutdown (Scenario 7)

**Step 1: Close Claude Code**

Completely quit Claude Code application.

**Step 2: Check server logs**

If you have access to server stderr output, check for:
- "MCP server shutdown" message
- No error messages
- Clean exit

**Step 3: Restart and verify clean startup**

Reopen Claude Code and verify:
- Server starts fresh
- No stale state
- New queries work

**Step 4: Document shutdown test**

Update `docs/testing/2026-02-06-mcp-integration-manual-tests.md`:

```markdown
## Scenario 7: Server Shutdown

**Status**: ✅ Pass / ❌ Fail

**Results**:
- [ ] Server received EOF on stdin
- [ ] Logged "MCP server shutdown"
- [ ] No error messages
- [ ] Clean exit
- [ ] Restart successful

**Notes**:
[Any observations about shutdown behavior]
```

---

## Task 9: Create README Documentation

**Files:**
- Modify: `README.md` (add MCP Integration section)

**Step 1: Find Installation section in README**

Read: `README.md`
Locate the "## Installation" section.

**Step 2: Add MCP Integration section after Installation**

Insert after Installation section:

```markdown
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
```

**Step 3: Commit README update**

```bash
git add README.md
git commit -m "docs: add MCP integration section to README

- Add Quick Start for Claude Code
- Link to detailed integration guide

Part of Issue #7

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 10: Create Detailed Integration Guide

**Files:**
- Create: `docs/mcp-integration.md`

**Step 1: Create comprehensive integration guide**

Create `docs/mcp-integration.md` with complete content:

```markdown
# MCP Integration Guide

## Overview

TMKB integrates with AI coding assistants via the Model Context Protocol (MCP).
The MCP server exposes the `tmkb_query` tool for querying authorization security threats.

**Supported Clients:**
- Claude Code (desktop app and CLI)
- Any MCP-compatible client supporting JSON-RPC 2.0

**What you get:**
- AI assistant can query TMKB during code generation
- Context-aware security recommendations
- Actionable threat patterns for authorization vulnerabilities

## Prerequisites

- Go 1.25 or later
- TMKB built from source
- Claude Code installed (or other MCP client)
- Threat patterns loaded (default: `./patterns` directory)

## Installation

### 1. Build TMKB

```bash
git clone https://github.com/mark-chris/tmkb.git
cd tmkb
go build -o tmkb cmd/tmkb/main.go
```

### 2. Verify Installation

```bash
./tmkb --version
./tmkb list  # Should show loaded patterns
```

## Configuration

### Claude Code Desktop

**Location**: `~/.claude/mcp_settings.json`

```json
{
  "mcpServers": {
    "tmkb": {
      "command": "/absolute/path/to/tmkb",
      "args": ["serve"],
      "env": {
        "TMKB_PATTERNS": "/absolute/path/to/patterns"
      }
    }
  }
}
```

**Important**:
- Use **absolute paths** (not relative or `~`)
- Set `TMKB_PATTERNS` if patterns aren't in default location
- Restart Claude Code after changing config

### Claude Code CLI

Coming soon.

### Configuration Options

| Field | Required | Description |
|-------|----------|-------------|
| `command` | Yes | Absolute path to tmkb binary |
| `args` | Yes | Must be `["serve"]` |
| `env.TMKB_PATTERNS` | No | Custom patterns directory (default: `./patterns`) |

## Usage

### Basic Query

In Claude Code chat:
> Query TMKB for background job authorization threats

Claude Code will invoke the `tmkb_query` tool and return relevant patterns.

### Query with Filters

> Query TMKB for Python Flask multi-tenant security

The tool automatically detects language/framework from your query.

### Example Tool Call

Claude Code sends:
```json
{
  "name": "tmkb_query",
  "arguments": {
    "context": "background job processing",
    "language": "python",
    "framework": "flask"
  }
}
```

TMKB returns:
```json
{
  "pattern_count": 3,
  "patterns_included": 3,
  "patterns": [
    {
      "id": "TMKB-AUTH-001",
      "severity": "high",
      "threat": "Background jobs bypass authorization",
      "check": "Verify jobs check user permissions",
      "fix": "Add authorization check to job handler"
    }
  ]
}
```

## Troubleshooting

### Server Won't Start

**Symptom**: Claude Code shows "Server failed to start"

**Causes & Solutions**:
1. **Binary not found**: Check `command` path is absolute and correct
2. **Permission denied**: Run `chmod +x /path/to/tmkb`
3. **Patterns not loaded**: Set `TMKB_PATTERNS` environment variable

**Verify manually**:
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | /path/to/tmkb serve
```

Should return initialize response.

### Connection Refused

**Symptom**: Claude Code can't connect to server

**Solutions**:
1. Check config file location: `~/.claude/mcp_settings.json`
2. Verify JSON syntax is valid
3. Restart Claude Code after config changes

### No Patterns Loaded

**Symptom**: Tool returns `pattern_count: 0` for all queries

**Solutions**:
1. Check patterns directory exists: `ls /path/to/patterns`
2. Set `TMKB_PATTERNS` in config
3. Verify patterns are valid YAML: `./tmkb validate`

### Tool Not Appearing

**Symptom**: Claude Code doesn't show tmkb_query tool

**Solutions**:
1. Restart Claude Code completely (quit and reopen)
2. Check Claude Code logs for MCP errors
3. Verify server starts manually (see "Server Won't Start")

### Query Returns Errors

**Error**: "Invalid language 'java'. Supported languages: python"
**Solution**: TMKB MVP only supports Python. Use language: "python" or omit.

**Error**: "context must be non-empty"
**Solution**: Provide a meaningful context describing what you're implementing.

**Error**: "Unknown tool: tmkb_query"
**Solution**: Server initialization failed. Check logs and restart.

## Technical Details

### MCP Protocol

- **Protocol Version**: 2025-11-25
- **Transport**: stdio (JSON-RPC 2.0 over stdin/stdout)
- **Capabilities**: Tools only (no resources, prompts, or sampling)

### Tool Schema

**Name**: `tmkb_query`

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "context": {
      "type": "string",
      "description": "What you're implementing",
      "minLength": 1
    },
    "language": {
      "type": "string",
      "enum": ["python"],
      "description": "Programming language"
    },
    "framework": {
      "type": "string",
      "enum": ["flask", "any"],
      "description": "Framework context"
    },
    "verbosity": {
      "type": "string",
      "enum": ["agent", "human"],
      "default": "agent",
      "description": "Output format"
    }
  },
  "required": ["context"],
  "additionalProperties": false
}
```

### Server Lifecycle

1. **Startup**: Claude Code spawns `tmkb serve` subprocess
2. **Initialize**: Handshake, capability negotiation
3. **Ready**: Server accepts tools/list and tools/call requests
4. **Shutdown**: Claude Code closes stdin, server exits cleanly

## Advanced Usage

### Custom Patterns

Set `TMKB_PATTERNS` to load custom pattern directories:

```json
{
  "mcpServers": {
    "tmkb": {
      "command": "/path/to/tmkb",
      "args": ["serve"],
      "env": {
        "TMKB_PATTERNS": "/custom/patterns"
      }
    }
  }
}
```

### Multiple Pattern Directories

Not currently supported. Use symlinks as workaround:
```bash
ln -s /other/patterns/* /main/patterns/
```

## References

- [MCP Specification](https://modelcontextprotocol.io/specification/2025-11-25)
- [Claude Code MCP Documentation](https://docs.anthropic.com/claude/docs/model-context-protocol)
- [TMKB GitHub](https://github.com/mark-chris/tmkb)
```

**Step 2: Commit integration guide**

```bash
git add docs/mcp-integration.md
git commit -m "docs: add comprehensive MCP integration guide

- Installation and configuration instructions
- Usage examples with Claude Code
- Troubleshooting section for common issues
- Technical details about protocol and tool schema
- Advanced usage for custom patterns

Part of Issue #7

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 11: Finalize Testing Documentation

**Files:**
- Create/Complete: `docs/testing/2026-02-06-mcp-integration-manual-tests.md`

**Step 1: Review all test results**

Ensure all 7 scenarios are documented with:
- ✅ Pass / ❌ Fail status
- Actual behavior observed
- Any issues or unexpected behavior

**Step 2: Add summary section**

Add to top of `docs/testing/2026-02-06-mcp-integration-manual-tests.md`:

```markdown
# MCP Integration Manual Testing Results

**Date**: 2026-02-06
**Issue**: #7
**Tester**: [Your name]
**Environment**: Claude Code [version]

## Summary

| Scenario | Status | Notes |
|----------|--------|-------|
| 1. Server Startup | ✅ Pass / ❌ Fail | |
| 2. Basic Query | ✅ Pass / ❌ Fail | |
| 3. Filtered Query | ✅ Pass / ❌ Fail | |
| 4. Invalid Language | ✅ Pass / ❌ Fail | |
| 5. Empty Context | ✅ Pass / ❌ Fail | |
| 6. Multiple Queries | ✅ Pass / ❌ Fail | |
| 7. Server Shutdown | ✅ Pass / ❌ Fail | |

**Overall**: ✅ All tests passed / ❌ N tests failed

**Issues Found**: [List any bugs or unexpected behavior]

**Performance**: [Response times, resource usage observations]

---
```

**Step 3: Commit testing documentation**

```bash
git add docs/testing/2026-02-06-mcp-integration-manual-tests.md
git commit -m "test: document MCP integration manual testing results

- Tested all 7 scenarios with Claude Code
- Verified server startup, queries, errors, stability, shutdown
- Documented configuration and observed behavior

Part of Issue #7

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Success Criteria

**Implementation:**
- ✅ Serve command starts MCP server via stdio
- ✅ No unused --port flag
- ✅ Proper logging to stderr
- ✅ Clean error handling

**Testing:**
- ✅ All 7 test scenarios executed
- ✅ Server stable across multiple queries
- ✅ Errors return actionable messages
- ✅ No crashes or hangs

**Documentation:**
- ✅ README mentions MCP integration
- ✅ Detailed guide in docs/mcp-integration.md
- ✅ Example configuration provided
- ✅ Troubleshooting section complete
- ✅ Testing results documented

## Notes

**Manual Testing:**
- Tasks 2-8 are manual testing tasks, not TDD
- Cannot automate without programmatic MCP client
- Document results thoroughly for validation
- Take screenshots if helpful

**Documentation:**
- Tasks 9-10 create user-facing documentation
- No tests needed for documentation
- Focus on clarity and completeness

**Execution Order:**
- Task 1 must complete first (implementation)
- Tasks 2-8 can be done in sequence (manual testing)
- Tasks 9-11 can be done in parallel or after testing
