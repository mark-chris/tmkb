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
