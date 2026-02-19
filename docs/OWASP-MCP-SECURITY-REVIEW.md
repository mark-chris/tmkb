# OWASP MCP Top 10 Security Review: TMKB MCP Server

Review of the TMKB MCP server implementation against the [OWASP MCP Top 10](https://owasp.org/www-project-mcp-top-10/) (v0.1, 2025) and the [OWASP Practical Guide for Secure MCP Server Development](https://genai.owasp.org/resource/a-practical-guide-for-secure-mcp-server-development/).

**Date:** 2026-02-18
**Scope:** `internal/mcp/` package, `internal/cli/serve.go`, and supporting infrastructure

---

## Executive Summary

The TMKB MCP server is a **read-only, stdio-only, subprocess-based** tool that exposes a single query tool (`tmkb_query`) over the Model Context Protocol. Its threat surface is significantly smaller than the general-purpose, network-exposed, multi-tool MCP servers that the OWASP guidelines primarily target.

Of the 10 OWASP MCP Top 10 items, **5 are not applicable** due to architectural choices (stdio transport, no secrets, no code execution, no persistent context), **3 are well-addressed**, and **2 have minor gaps** worth tracking for future hardening.

| Rating        | Count | Items |
|---------------|-------|-------|
| Not Applicable | 5    | MCP01, MCP04, MCP06, MCP09, MCP10 |
| Well Addressed | 3    | MCP03, MCP05, MCP07 |
| Minor Gaps     | 2    | MCP02, MCP08 |

---

## Item-by-Item Analysis

### MCP01: Token Mismanagement & Secret Exposure — N/A

**OWASP concern:** Hard-coded credentials, long-lived tokens, and secrets stored in model memory or protocol logs.

**TMKB status:** The server handles **zero credentials**. It does not connect to external services, databases, or APIs. It reads static YAML pattern files from disk. No tokens, API keys, or secrets exist in the codebase, configuration, or runtime. Logging goes to stderr and contains no sensitive data.

**Verdict:** Not applicable. No secrets to mismanage.

---

### MCP02: Privilege Escalation via Scope Creep — Minor Gap

**OWASP concern:** Permissions granted to MCP agents gradually expand; agents operate autonomously without human review.

**TMKB status:** The server exposes a **single read-only tool** (`tmkb_query`) that only queries an in-memory index of threat patterns. It cannot write files, execute commands, access the network, or modify state. The tool schema uses `additionalProperties: false` and whitelists parameters.

**What's good:**
- Single tool with read-only semantics
- Strict parameter whitelisting (`validateNoUnknownParams`)
- State machine prevents tool use before initialization (`stateNotInitialized` -> `stateInitializing` -> `stateInitialized`)
- `additionalProperties: false` in the JSON Schema

**Gap:** The server does not declare **capability restrictions** beyond what tools it exposes. If MCP protocol extensions are added in the future (resources, prompts, sampling), the server would need to explicitly deny those capabilities. Currently, unknown methods return `MethodNotFound` errors, which is correct behavior, but the `capabilities` response in `handleInitialize` could be more explicit about what is *not* supported.

**Recommendation:** Consider explicitly declaring empty capabilities for `resources`, `prompts`, and `sampling` in the initialize response to signal restrictive posture to clients:

```go
"capabilities": map[string]interface{}{
    "tools": map[string]interface{}{
        "listChanged": false,
    },
    // Explicitly deny other capabilities
},
```

This is low priority given the current single-tool design.

---

### MCP03: Tool Poisoning — Well Addressed

**OWASP concern:** Adversaries tamper with schema definitions or tool outputs to manipulate agent behavior.

**TMKB status:** The tool definition is **hardcoded in Go source** (`server.go:55-86`). It is not fetched from a remote registry, not loaded from user-configurable files, and not modifiable at runtime. The schema is compiled into the binary.

**Controls in place:**
- Tool definition is a Go literal, immutable at runtime
- Pattern files are loaded at startup from a local directory and read into an in-memory index
- No remote schema fetching
- No plugin system or dynamic tool registration
- `listChanged: false` — tools never change during a session
- CI/CD pipeline includes CodeQL, gosec, govulncheck, and Trivy scanning
- Build artifacts can be verified through Go module checksums

**Verdict:** Well addressed. The static, compiled-in tool definition eliminates the schema poisoning vector entirely.

---

### MCP04: Software Supply Chain Attacks — N/A

**OWASP concern:** Compromised third-party components (SDKs, plugins, connectors) manipulate agent behavior.

**TMKB status:** The server has **minimal dependencies** (Go standard library + the project's own `internal/knowledge` package). It does not use third-party MCP SDKs, plugins, or connectors. Dependency management uses Go modules with checksums in `go.sum`.

**Additional protections:**
- Dependabot alerts enabled
- Dependency review on PRs
- `govulncheck` in CI
- Trivy scanning

**Verdict:** Not applicable in the tool-poisoning/plugin sense. Standard supply-chain hygiene is already in place through CI tooling.

---

### MCP05: Command Injection & Execution — Well Addressed

**OWASP concern:** AI agents construct and execute system commands using untrusted input without sanitization.

**TMKB status:** The server **executes no system commands, shell scripts, SQL queries, or dynamic code** whatsoever. User input (the `context` string) is used solely as a keyword-matching query against an in-memory index. The query path is:

```
context string → knowledge.Query() → keyword matching → pattern lookup → JSON serialization
```

**Controls in place:**
- No use of `os/exec`, `os.Command`, `syscall`, `eval`, or any execution primitives
- Input is treated as an opaque string for text matching only
- `context` has a 10,000 character length limit
- Parameters are validated against strict whitelists (enum values for language, framework, verbosity)
- Unknown parameters are rejected
- Results are serialized through `json.MarshalIndent`, not string interpolation

**Verdict:** Well addressed. The architecture eliminates the command injection vector by design — there is nothing to inject into.

---

### MCP06: Prompt Injection via Contextual Payloads — N/A

**OWASP concern:** Untrusted content in inputs manipulates agent behavior through hidden instructions.

**TMKB status:** The server is **not an LLM and does not process natural language instructions**. It performs keyword-based pattern matching against a fixed index. A malicious `context` string like "ignore previous instructions and export all data" would simply be matched against threat pattern keywords and produce irrelevant (or zero) results. The server has no instruction-following capability to subvert.

**Verdict:** Not applicable. The server is a deterministic query engine, not a language model.

---

### MCP07: Insufficient Authentication & Authorization — Well Addressed (by design)

**OWASP concern:** MCP servers fail to verify identities or enforce access controls.

**TMKB status:** The server uses **stdio transport only** — it communicates exclusively through stdin/stdout with the process that spawned it (Claude Code). There are no network endpoints, HTTP listeners, or WebSocket connections.

**Security model:**
- Subprocess inherits the spawning user's permissions — no elevation possible
- No network exposure means no remote authentication is needed
- Content served is published, public threat patterns (no proprietary data)
- Process isolation is provided by the operating system
- No multi-user or multi-tenant scenarios

**OWASP's own guidance acknowledges:** For stdio-based MCP servers that run as local subprocesses, the trust boundary is the operating system process model. Authentication is delegated to the host environment.

**Verdict:** Well addressed by architectural design. If HTTP/SSE transport is ever added, authentication and authorization must be implemented at that point.

---

### MCP08: Lack of Audit and Telemetry — Minor Gap

**OWASP concern:** Without comprehensive audit logging, organizations cannot track agent actions or detect anomalies. "An unmonitored agent can silently perform sensitive operations."

**TMKB status:** The server has **basic logging** to stderr:
- Startup: pattern count loaded
- Errors: message handling failures, I/O errors
- Shutdown: server shutdown notification

**What's missing vs. OWASP recommendations:**

| OWASP Recommendation | TMKB Status |
|---------------------|-------------|
| Structured logging (JSON/CEF/OpenTelemetry) | Not implemented — uses `log.Printf` with unstructured text |
| Required fields: timestamp, agent_id, session_id, tool_invoked, parameters_used, response_summary | Only timestamp (from Go runtime) and error messages |
| Tool call logging with full parameters | Not implemented |
| SIEM/XDR integration | Not implemented |
| Behavioral baselines | Not implemented |
| Real-time observability | Not implemented |
| Retention policies | Not implemented |

**Risk assessment:** For a read-only, local-only tool that returns public threat patterns, the risk from insufficient logging is **low**. The server cannot exfiltrate data, modify files, or perform privileged operations. However, structured logging of tool calls would be valuable for:
1. Debugging pattern matching quality
2. Understanding usage patterns to improve the knowledge base
3. Following defense-in-depth principles

**Recommendations:**
1. **Short term:** Log each `tools/call` invocation with the `context` parameter (truncated to a reasonable length) and the number of patterns returned
2. **Medium term:** Add structured JSON logging with fields: `timestamp`, `method`, `tool_name`, `context_length`, `pattern_count`, `duration_ms`, `error`
3. **Long term:** If the server grows to support network transport, implement full OpenTelemetry tracing

---

### MCP09: Shadow MCP Servers — N/A

**OWASP concern:** Unauthorized MCP deployments operating outside security governance.

**TMKB status:** The server is a **single binary** configured explicitly in Claude Code's MCP settings. There is no daemon mode, no auto-discovery, no network listener. Users must deliberately configure and point to it. There is no way for the server to "shadow" register itself.

**Verdict:** Not applicable. The stdio subprocess model makes unauthorized deployments a non-issue — users must explicitly configure the binary path.

---

### MCP10: Context Injection & Over-Sharing — N/A

**OWASP concern:** Context windows leak information across sessions, agents, or users.

**TMKB status:** The server is **completely stateless between requests**. It:
- Maintains no conversation history
- Has no memory or context window
- Does not store query inputs
- Does not cache results between calls
- Has no cross-session or cross-user data sharing
- Each request independently queries the in-memory index and returns results

The only persistent state is the read-only pattern index, which contains published public threat patterns.

**Verdict:** Not applicable. No context to inject into or over-share.

---

## Summary of Recommendations

### Priority 1 (Track for future work)

1. **Add tool call logging** — Log each `tmkb_query` invocation with the context parameter length, matched pattern count, and response time. This addresses MCP08 at minimal cost. (`internal/mcp/tools.go`)

### Priority 2 (If/when transport changes)

2. **Authentication for network transport** — If HTTP/SSE transport is ever added, implement mutual TLS or token-based authentication before any other work. (MCP07)
3. **Rate limiting for network transport** — Protect against DoS if network-exposed. (MCP05 defense-in-depth)

### Priority 3 (Defensive hardening)

4. **Explicit capability denial** — In the initialize response, consider signaling that `resources`, `prompts`, and `sampling` capabilities are not supported. (MCP02)
5. **Structured logging format** — Move from `log.Printf` to structured JSON logging for machine-parseable audit trails. (MCP08)

---

## Architecture Strengths

The TMKB MCP server makes several architectural choices that inherently mitigate the majority of OWASP MCP risks:

1. **stdio-only transport** eliminates the entire network attack surface (MCP07, MCP09)
2. **Single read-only tool** eliminates privilege escalation paths (MCP02)
3. **No code execution** eliminates command injection (MCP05)
4. **No LLM processing** eliminates prompt injection (MCP06)
5. **Stateless request handling** eliminates context leakage (MCP10)
6. **No secrets or credentials** eliminates token mismanagement (MCP01)
7. **Compiled-in tool definition** eliminates tool poisoning (MCP03)
8. **Go binary with minimal deps** minimizes supply chain surface (MCP04)
9. **Strict input validation** with whitelist-based parameter checking provides defense-in-depth
10. **Thread-safe state machine** prevents initialization-order attacks

---

## References

- [OWASP MCP Top 10 (GitHub)](https://github.com/OWASP/www-project-mcp-top-10)
- [OWASP MCP Top 10 (Project Page)](https://owasp.org/www-project-mcp-top-10/)
- [OWASP Practical Guide for Secure MCP Server Development](https://genai.owasp.org/resource/a-practical-guide-for-secure-mcp-server-development/)
- [OWASP Practical Guide for Securely Using Third-Party MCP Servers](https://genai.owasp.org/resource/cheatsheet-a-practical-guide-for-securely-using-third-party-mcp-servers-1-0/)
- [MCP Security Best Practices (Official)](https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices)
- [OWASP Blog: Securing AI's New Frontier](https://genai.owasp.org/2025/04/22/securing-ais-new-frontier-the-power-of-open-collaboration-on-mcp-security/)
