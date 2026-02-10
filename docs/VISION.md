# Security Context Plane: Vision

> Structured security knowledge that AI agents consume at code-generation time, preventing architectural vulnerabilities before they exist.

## The Gap

AI coding agents have transformed software development. They generate entire applications from natural language prompts. But they have a systematic blindspot: **architectural security patterns that require reasoning across system boundaries**.

LLMs know OWASP Top 10. They add `@login_required` decorators. They implement RBAC when asked. But they consistently fail to propagate authorization context across async boundaries, maintain consistency between related endpoints, or model business object ownership in multi-tenant systems.

This isn't a training data problem that will be solved by the next model release. TMKB's validation demonstrates the same failure across 3 providers (Anthropic, OpenAI, Google), 4 models, and 2 application types. The blindspot is structural.

## The Vision: Security Context Plane

The Security Context Plane is an infrastructure layer that provides security knowledge to AI agents during code generation. Just as a data plane carries application data and a control plane manages routing, the Security Context Plane carries threat models, authorization patterns, and security constraints.

```
┌─────────────────────────────────────────────────────────┐
│                   AI Coding Agent                        │
│           (Claude, GPT, Gemini, etc.)                   │
├────────────────────────┬────────────────────────────────┤
│                        │                                │
│   Code Generation      │   Security Context Plane       │
│   ┌──────────────┐     │   ┌──────────────────────┐     │
│   │ User Prompt  │     │   │ Threat Patterns      │     │
│   │              │────►│   │ (TMKB)               │     │
│   │ "Build a     │     │   ├──────────────────────┤     │
│   │  multi-tenant│     │   │ Policy Constraints   │     │
│   │  API with    │     │   │ (future)             │     │
│   │  background  │     │   ├──────────────────────┤     │
│   │  jobs"       │     │   │ Compliance Rules     │     │
│   └──────────────┘     │   │ (future)             │     │
│                        │   └──────────────────────┘     │
│         │              │            │                    │
│         ▼              │            ▼                    │
│   ┌──────────────┐     │   ┌──────────────────────┐     │
│   │ Generated    │◄────│───│ Security-Aware Code  │     │
│   │ Code         │     │   │ (auth in jobs, tenant│     │
│   │              │     │   │  isolation, etc.)    │     │
│   └──────────────┘     │   └──────────────────────┘     │
│                        │                                │
└────────────────────────┴────────────────────────────────┘
```

## TMKB: The First Component

TMKB (Threat Model Knowledge Base) is the first implementation of the Security Context Plane. It focuses on authorization enforcement in multi-tenant applications because:

1. **Highest impact**: Authorization failures lead to cross-tenant data breaches
2. **Consistent LLM failure**: 100% baseline failure rate on async boundary auth (6/6 runs, 3 providers)
3. **Measurable improvement**: 0 to 5 authorization checks in background jobs with TMKB context
4. **Clear boundary**: Authorization patterns are well-defined enough to encode as structured data

## Design Principles

### 1. Agent-First

TMKB is designed for consumption by AI agents, not just humans. Patterns include:
- Token-limited agent summaries (<100 tokens)
- Trigger keywords and file patterns for automatic relevance detection
- JSON output mode optimized for agent context windows
- MCP server integration for seamless tool use

### 2. Encode Judgment, Not Rules

TMKB doesn't encode "always add authentication" (LLMs already know that). It encodes the judgment that "async execution boundaries are trust boundaries requiring re-authorization" -- the kind of insight that comes from security design reviews, not documentation.

### 3. Validate Empirically

Every pattern includes baseline test results showing that LLMs fail without the pattern and succeed with it. This isn't theoretical -- it's measured.

### 4. Minimal and Focused

The MVP covers authorization patterns in multi-tenant applications. It doesn't try to cover all of security. Depth over breadth.

## Future Directions

### Near-Term (Post-MVP)

- **Language expansion**: Node.js, Go, Ruby, Java pattern examples
- **Pattern expansion**: Service-to-service auth, API gateway patterns, event-driven architectures
- **Integration testing**: Automated baseline/enhanced comparison pipeline

### Medium-Term

- **Policy integration**: Organization-specific authorization policies as structured data
- **Compliance mapping**: Map patterns to SOC 2, ISO 27001, and GDPR requirements
- **IDE integration**: VS Code extension for inline threat pattern warnings

### Long-Term

- **Multi-source context plane**: Combine threat models, API contracts, infrastructure topology, and compliance rules into a unified security context
- **Feedback loops**: Agent-generated code feeds back into pattern validation
- **Community patterns**: Open contribution model for domain-specific threat patterns

## Why This Matters

Every AI-generated application that handles background jobs without re-authorization is a potential data breach. Every webhook processor that trusts payloads without re-verification is a potential attack vector. These aren't edge cases -- they're the default output of every major AI coding assistant.

The Security Context Plane makes the secure path the default path. TMKB is the proof of concept.
