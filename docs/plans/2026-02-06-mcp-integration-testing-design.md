# MCP Integration Testing Design

**Date**: 2026-02-06
**Issue**: #7
**Status**: Approved

## Overview

Complete the MCP integration loop by connecting the CLI serve command to the MCP protocol implementation and validating end-to-end functionality with Claude Code.

## Goals

1. Fix `serve` command to actually start the MCP server (currently has TODO)
2. Test MCP server integration with Claude Code as client
3. Verify tool calls work end-to-end with multiple scenarios
4. Document integration setup with step-by-step instructions
5. Create example MCP configuration for Claude Code

## Current State Assessment

### What Exists

**MCP Protocol Implementation** (PR #24, merged)
- ✅ Full JSON-RPC 2.0 protocol in `internal/mcp/`
- ✅ Server with stdio transport (`ServeStdio`)
- ✅ Initialize/initialized lifecycle
- ✅ Tools/list and tools/call handlers
- ✅ Strict input validation
- ✅ 86.3% test coverage

**CLI Serve Command** (`internal/cli/serve.go`)
- ✅ Command exists with help text
- ❌ Has TODO placeholder - doesn't actually start server
- ❌ Has unused `--port` flag (stdio doesn't use ports)
- ❌ Not connected to `internal/mcp` implementation

### What's Missing

- ❌ Serve command doesn't call MCP server
- ❌ No manual testing with real MCP client
- ❌ No integration documentation
- ❌ No example MCP configuration
- ❌ No troubleshooting guide

## Design Decisions

### Transport: stdio Only

**Decision**: Use stdio transport only, remove `--port` flag.

**Rationale**:
- stdio is the standard for Claude Code/Desktop integration
- MCP server implementation only supports stdio (HTTP/SSE not implemented)
- Simpler: no network configuration, no port conflicts
- Standard MCP pattern: client (Claude Code) spawns server and manages stdio pipes

**Alternative**: Add HTTP/SSE transport. Rejected because it adds significant complexity and isn't needed for the primary use case (Claude Code integration).

### Scope: Implementation + Testing + Documentation

**Decision**: Issue #7 includes serve command fix, manual testing, and documentation.

**Rationale**:
- Serve command fix is small (~10 lines)
- Can't test without working serve command
- Keeps work atomic: one PR fixes, tests, and documents
- Natural flow: implement → test → document

**Alternative**: Split into separate issues. Rejected because fixing serve command without testing creates incomplete work.

### Documentation: README + Detailed Doc

**Decision**: Brief overview in README with link to detailed `docs/mcp-integration.md`.

**Rationale**:
- README mention provides discoverability
- Detailed doc allows comprehensive instructions without cluttering README
- Users can jump directly to detailed guide
- Separate doc enables troubleshooting section

### Testing: Comprehensive Manual Testing

**Decision**: Test multiple scenarios including success and error cases.

**Rationale**:
- Validates full protocol works end-to-end
- Finds integration bugs before users encounter them
- Creates real-world examples for documentation
- Builds confidence in the integration

**Alternative**: Basic smoke test only. Rejected because it wouldn't catch edge cases or validate error handling.

## Architecture

### Three-Phase Approach

**Phase 1: Implementation**
- Fix serve command to call MCP server
- Remove unused `--port` flag
- Add proper error handling
- Keep changes minimal (~10 lines)

**Phase 2: Manual Testing**
- Configure Claude Code with MCP server
- Test full protocol flow
- Test multiple query scenarios
- Test error cases
- Document findings

**Phase 3: Documentation**
- Update README with brief overview
- Create detailed integration guide
- Provide example configuration
- Add troubleshooting section

## Implementation Details

### Changes to `internal/cli/serve.go`

**Remove:**
```go
var (
    servePort int  // DELETE
)

serveCmd.Flags().IntVar(&servePort, "port", 3000, "Port to listen on")  // DELETE
```

**Update help text:**
```go
Long: `Start a Model Context Protocol (MCP) server that AI agents can query.

The MCP server communicates via stdin/stdout using the JSON-RPC 2.0 protocol.
It is designed to be invoked by MCP clients like Claude Code.

Examples:
  # Start MCP server (typically invoked by Claude Code)
  tmkb serve`,
```

**Implement `runServe` function:**
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

**Key Implementation Details:**
- All logging goes to `stderr` to avoid interfering with JSON-RPC protocol on stdout
- `ServeStdio` blocks until stdin closes (normal for stdio servers)
- Clean error handling with context
- Simple: leverages existing `mcp.Server` implementation

### Import Requirements

Add to imports:
```go
import (
    "log"
    "os"

    "github.com/mark-chris/tmkb/internal/mcp"
)
```

## Manual Testing Plan

### Test Setup

**Prerequisites:**
1. Build TMKB: `go build -o tmkb cmd/tmkb/main.go`
2. Ensure patterns are loaded (default location or set `TMKB_PATTERNS`)
3. Have Claude Code installed
4. Have access to Claude Code MCP settings

**MCP Configuration:**

Create/update `~/.claude/mcp_settings.json`:
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

**Note**: Use absolute paths. Claude Code spawns the server process.

### Test Scenarios

#### Scenario 1: Server Startup and Connection

**Objective**: Verify server starts and Claude Code connects successfully.

**Steps:**
1. Configure MCP settings with tmkb server
2. Restart Claude Code
3. Open Claude Code chat
4. Check server logs (stderr output)

**Expected Results:**
- Server starts without errors
- Log shows "Starting MCP server with N patterns loaded"
- Claude Code recognizes the tmkb server
- Initialize handshake succeeds
- Tools/list returns tmkb_query tool

**Success Criteria:**
- No connection errors
- Tool appears in Claude Code's available tools

#### Scenario 2: Basic Query (Success Path)

**Objective**: Verify tool call works with valid parameters.

**Steps:**
1. In Claude Code chat, ask: "Query TMKB for background job authorization threats"
2. Observe tool call in chat
3. Check response format

**Expected Results:**
- Claude Code invokes tmkb_query tool
- Tool returns JSON with pattern_count > 0
- Patterns include relevant authorization threats
- Response format is valid (pattern_count, patterns array)

**Success Criteria:**
- Tool call succeeds
- Returns relevant patterns (e.g., TMKB-AUTH-001)
- JSON structure matches QueryResult schema

#### Scenario 3: Query with Filters

**Objective**: Verify language and framework filters work.

**Steps:**
1. Ask: "Query TMKB for Python Flask background job security"
2. Check returned patterns

**Expected Results:**
- Tool call includes language: "python", framework: "flask"
- Returned patterns are filtered to Python/Flask
- Pattern count reflects filtering

**Success Criteria:**
- Only Python patterns returned
- Only Flask or "any" framework patterns returned

#### Scenario 4: Validation Error (Invalid Language)

**Objective**: Verify validation errors return actionable messages.

**Steps:**
1. Ask: "Query TMKB for Java security threats"
2. Observe error response

**Expected Results:**
- Tool returns `isError: true`
- Error message: "Invalid language 'java'. Supported languages: python"
- Claude Code receives the error message
- LLM can potentially self-correct

**Success Criteria:**
- Tool execution error (not protocol error)
- Clear, actionable error message
- Server remains operational after error

#### Scenario 5: Empty Context Error

**Objective**: Verify required field validation.

**Steps:**
1. Try to invoke tool with empty or missing context
2. Observe error response

**Expected Results:**
- Tool returns error: "context must be non-empty"
- Server handles error gracefully

**Success Criteria:**
- Validation error caught
- Server doesn't crash

#### Scenario 6: Multiple Queries (Stability)

**Objective**: Verify server handles multiple requests in one session.

**Steps:**
1. Run 4-5 different queries in one chat session:
   - "Query TMKB for multi-tenant isolation"
   - "Query TMKB for background job authorization"
   - "Query TMKB for admin panel security"
   - Try an invalid query
   - Run another valid query

**Expected Results:**
- All queries succeed independently
- No state corruption between requests
- Server remains stable
- Error in one query doesn't affect subsequent queries

**Success Criteria:**
- All valid queries return results
- Server doesn't crash or hang
- Clean separation between requests

#### Scenario 7: Server Shutdown

**Objective**: Verify graceful shutdown.

**Steps:**
1. Close Claude Code or stop the chat session
2. Check server logs

**Expected Results:**
- Server receives EOF on stdin
- ServeStdio returns cleanly
- Log shows "MCP server shutdown"
- No error messages

**Success Criteria:**
- Clean shutdown without errors
- No resource leaks

### Test Results Documentation

For each scenario, document:
- ✅ Pass / ❌ Fail status
- Actual behavior observed
- Screenshots or terminal output (optional but helpful)
- Any unexpected behavior or bugs found
- Performance notes (response time)

## Documentation Structure

### README.md Addition

**Location**: After "Installation" section

**Content**:
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

### docs/mcp-integration.md

**Complete Structure:**

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
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{}}}' | /path/to/tmkb serve
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

## Success Criteria

### Implementation
- ✅ Serve command starts MCP server successfully
- ✅ No unused flags (--port removed)
- ✅ Proper error handling and logging
- ✅ Clean shutdown on EOF

### Testing
- ✅ All 7 test scenarios pass
- ✅ Server stable across multiple queries
- ✅ Error cases return actionable messages
- ✅ No crashes or hangs

### Documentation
- ✅ README mentions MCP integration
- ✅ Detailed guide in docs/mcp-integration.md
- ✅ Example configuration provided
- ✅ Troubleshooting section addresses common issues

## Trade-offs

### stdio vs HTTP Transport

**Chosen**: stdio only

**Pro**: Simple, standard for Claude Code, no network config
**Con**: Can't access remotely, no web UI

**Decision**: stdio meets the primary use case. HTTP can be added later if needed.

### Automated vs Manual Testing

**Chosen**: Manual testing with documented scenarios

**Pro**: Tests real integration, finds UX issues, creates examples
**Con**: Not repeatable, relies on human validation

**Decision**: Manual testing is appropriate for integration validation. Protocol-level behavior is already covered by unit tests.

### README vs Separate Doc

**Chosen**: Both (brief README + detailed doc)

**Pro**: Discoverability + detail without clutter
**Con**: Slight duplication between README and detailed doc

**Decision**: Benefits of discoverability outweigh minimal duplication.

## Future Enhancements

1. **Automated Integration Tests** - Programmatic MCP client for CI/CD
2. **HTTP/SSE Transport** - Web-based MCP clients
3. **More Languages** - Go, JavaScript, Java support
4. **Performance Metrics** - Track query latency, tool usage
5. **Claude Code CLI Config** - Documentation for CLI usage

## References

- Issue #7: MCP Integration Testing
- PR #24: MCP Protocol Implementation
- [MCP Specification (2025-11-25)](https://modelcontextprotocol.io/specification/2025-11-25)
- [Claude Code Documentation](https://docs.anthropic.com/claude/docs)
- Existing code: `internal/mcp/server.go`, `internal/cli/serve.go`
