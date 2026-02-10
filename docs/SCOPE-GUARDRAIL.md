# Scope Guardrail

What TMKB is and isn't. This document prevents scope creep and sets clear expectations.

## In Scope

### Authorization Patterns (MVP Focus)

TMKB covers **authorization enforcement failures** in multi-tenant applications, specifically:

- **Async boundary authorization**: Background jobs, webhook processors, event handlers that lose authorization context
- **Object ownership validation**: Server-side verification that users can access specific resources
- **List/detail consistency**: Authorization logic consistency across related endpoints
- **Tenant isolation**: Ensuring resources don't leak across organizational boundaries
- **Soft-delete resurrection**: Preventing deleted resources from being processed by deferred work

### Application Types

- Multi-tenant SaaS APIs
- Background job processing (Celery, Sidekiq, Bull, etc.)
- Webhook-receiving APIs
- Any application with async processing boundaries

### Supported Interfaces

- **CLI tool**: Direct query, get, list, validate commands
- **MCP server**: Integration with AI coding assistants (Claude Code, etc.)
- **Structured data**: YAML patterns consumable by any tool

### Languages (Pattern Examples)

- **Python/Flask** (MVP -- full code examples)
- Patterns generalize to any language/framework with async processing

## Out of Scope

### Not a SAST Tool

TMKB does **not** scan code for vulnerabilities. It provides threat context to agents *before* they generate code. Think of it as a design review checklist, not a code scanner.

### Not Authentication

TMKB focuses on **authorization** (what can you do?), not **authentication** (who are you?). It assumes authentication is handled by the framework (`@login_required`, JWT middleware, etc.).

### Not Syntax-Level Security

LLMs already handle well-documented, syntax-level vulnerabilities:
- SQL injection (parameterized queries)
- XSS (output encoding)
- CSRF (token validation)
- JWT algorithm confusion

TMKB addresses the **architectural** patterns that LLMs miss -- patterns that require reasoning across system boundaries.

### Not Runtime Protection

TMKB operates at **design time** (code generation), not runtime. It doesn't provide:
- Web Application Firewall (WAF) rules
- Runtime authorization enforcement
- Intrusion detection signatures
- Rate limiting or DDoS protection

### Not Compliance Certification

TMKB patterns align with security best practices but don't constitute compliance with any standard (SOC 2, ISO 27001, PCI DSS, etc.). Compliance mapping is a future direction.

### Not a Complete Threat Model

TMKB provides **patterns**, not a complete threat model for any specific application. A full threat model requires understanding the specific application's architecture, data flows, and trust boundaries.

## Boundary Decisions

| Question | Answer | Rationale |
|----------|--------|-----------|
| Should TMKB cover input validation? | No | LLMs handle this adequately |
| Should TMKB cover network security? | No | Infrastructure concern, not code-generation |
| Should TMKB cover secret management? | No (for now) | Well-documented, LLMs do okay |
| Should TMKB cover logging/monitoring? | No | Operational concern |
| Should TMKB cover rate limiting? | No | Availability, not authorization |
| Should TMKB cover CORS? | No | Configuration, not architectural pattern |
| Should TMKB cover service-to-service auth? | Future | Natural extension of boundary patterns |
| Should TMKB cover event-driven architectures? | Future | Similar async boundary patterns |

## How to Decide If Something Is In Scope

A pattern belongs in TMKB if it meets **all** of:

1. **Architectural**: Requires reasoning across system boundaries (not within a single function)
2. **LLM blindspot**: Demonstrably missed by AI coding agents in baseline tests
3. **Authorization-related**: Involves who can do what to which resource
4. **Encodable**: Can be expressed as structured data with triggers, mitigations, and code examples
5. **Actionable**: Provides specific guidance an agent can follow during code generation
