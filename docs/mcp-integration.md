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

## Automatic Invocation

By default, `tmkb_query` is only called when the assistant *decides* to: it
reads your request, infers that authorization context is relevant, and invokes
the tool. That works, but it is non-deterministic — on a quick edit the model
may skip the query entirely.

If you want TMKB consulted *reliably* (before every code edit, for compliance
or team-wide consistency), Claude Code offers several mechanisms that invoke the
tool without depending on the model's judgement. They differ in how
deterministic they are and what they cost per use:

| Mechanism | Determinism | Triggered by | Best for | Trade-off |
|-----------|-------------|--------------|----------|-----------|
| [PreToolUse hook](#pretooluse-hook) | Highest — fires on every matching tool call | The harness, automatically | Guaranteeing a query before each code edit | A query (latency + tokens) on *every* matched edit; generic context |
| [Slash command](#slash-command) | High, but user-initiated | You typing `/tmkb` | On-demand, deliberate queries | Only runs when you remember to invoke it |
| [Skill / subagent hook](#skill-or-subagent-frontmatter-hook) | High, scoped | The harness, while that skill/agent is active | Limiting auto-queries to security-relevant work | Only fires inside that skill/agent's context |
| [Project instructions](#project-instructions-claudemd) | Low — guidance, not enforcement | The model, prompted | Steering deliberate queries with tailored context | The model can still skip it |

These are complementary, not mutually exclusive. A common setup pairs a
**PreToolUse hook** (a guaranteed backstop) with **CLAUDE.md guidance** (so the
model also queries *deliberately*, with a `context` tailored to the feature —
which a generic hook can't express).

> **Backstop, not a replacement.** An automatic hook fires a generic query; it
> can't describe the specific feature being built the way a deliberate query
> can. Treat it as a safety net that guarantees *something* relevant is in
> context, and still query TMKB intentionally during design.

### PreToolUse hook

A [hook](https://code.claude.com/docs/en/hooks) of type `mcp_tool` calls
`tmkb_query` automatically before the assistant writes or edits a file, and
injects the returned patterns into context. Add it to a Claude Code settings
file (see [Where hooks live](#where-hooks-live)):

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write|Edit|MultiEdit|NotebookEdit",
        "hooks": [
          {
            "type": "mcp_tool",
            "server": "tmkb",
            "tool": "tmkb_query",
            "input": {
              "context": "About to edit ${tool_input.file_path}; surface authorization and trust-boundary threats before generating code.",
              "language": "python",
              "framework": "flask",
              "verbosity": "agent"
            },
            "timeout": 15,
            "statusMessage": "Consulting TMKB…"
          }
        ]
      }
    ]
  }
}
```

- `matcher` filters on tool name. A bare `|`-separated list (e.g.
  `Write|Edit|MultiEdit`) is exact matching; include any other character and it
  is treated as a JavaScript regular expression (e.g. `mcp__.*`).
- `${tool_input.file_path}` is substituted with the path being edited, so the
  query reflects the file in play.
- `server` must match the name you registered under `mcpServers`
  (see [Configuration](#configuration)); `tool` is `tmkb_query`. This `hooks`
  block lives in a *settings* file, which is separate from the `mcpServers`
  registration.
- The tool's output is placed in the model's context next to the tool result.
  To inject it explicitly, have the hook return JSON containing
  `hookSpecificOutput.additionalContext`.
- `timeout` (seconds) bounds the call — keep it short, since it runs on every
  matched edit.

> First-run note: hooks that fire before the MCP server has connected (e.g.
> `SessionStart`) may no-op. `PreToolUse` runs once the server is up, so the
> query resolves.

### Where hooks live

Hook config goes in a Claude Code **settings** file — distinct from the
`mcpServers` registration covered under [Configuration](#configuration). The
same `hooks` block works in any of these; they differ in scope and whether they
are shared:

| File | Scope | Committed to the repo? |
|------|-------|------------------------|
| `.claude/settings.json` | This project, all contributors | Yes — shared with the team |
| `.claude/settings.local.json` | This project, just you | No — gitignored by default |
| `~/.claude/settings.json` | All your projects | No — lives in your home directory |

Pick based on who should get the behavior: commit to `.claude/settings.json` to
apply it for everyone working in the repo, or use `.claude/settings.local.json`
/ `~/.claude/settings.json` to opt yourself in without changing the repo. When
the same setting appears in more than one file, the more specific scope wins
(`settings.local.json` over `settings.json` over `~/.claude/settings.json`).

### Slash command

A custom slash command lets you fire a TMKB query on demand by typing `/tmkb`.
Create `.claude/commands/tmkb.md` (project) or `~/.claude/commands/tmkb.md`
(personal):

```markdown
---
description: Query TMKB for authorization threats relevant to the current work
---

Call the `mcp__tmkb__tmkb_query` tool with a `context` describing what we are
building right now, plus `language: "python"` and `framework: "flask"`. Apply
each returned pattern's `check` and `fix` before writing code.
```

This is deterministic about *what* runs, but you have to invoke it — useful when
you want one deliberate, well-scoped query rather than one on every edit.

### Skill or subagent frontmatter hook

Skills and subagents can declare their own `hooks` in YAML frontmatter. A
`PreToolUse` hook defined there fires only while that skill or subagent is
active, giving you the determinism of the hook approach scoped to
security-relevant work instead of every edit in the session. Use the same
`mcp_tool` hook block shown above, placed in the skill/subagent frontmatter
rather than in a settings file.

### Project instructions (CLAUDE.md)

The lowest-friction option is to instruct the assistant to query TMKB in your
project's `CLAUDE.md`. This is guidance, not enforcement — the model can still
skip it — but it lets you request a query with a `context` tailored to each
feature, which a generic hook can't. It pairs well with a PreToolUse hook
backstop:

```markdown
Before writing or modifying code that touches auth, multi-tenancy, background
jobs, file ownership, or trust boundaries, call `mcp__tmkb__tmkb_query` with a
`context` describing the feature, plus `language: "python"` and
`framework: "flask"`. Apply each returned pattern's `check` and `fix`.
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
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | /path/to/tmkb serve
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

- **Protocol Version**: negotiated — the client's requested version is echoed when supported (`2025-06-18`, `2025-03-26`, `2024-11-05`); otherwise the server falls back to `2025-06-18`
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

- [MCP Specification](https://modelcontextprotocol.io/specification/2025-06-18)
- [Claude Code MCP Documentation](https://docs.anthropic.com/claude/docs/model-context-protocol)
- [Claude Code hooks](https://code.claude.com/docs/en/hooks)
- [Claude Code settings](https://code.claude.com/docs/en/settings)
- [TMKB GitHub](https://github.com/mark-chris/tmkb)
