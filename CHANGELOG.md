# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-02-10

### Added

- **Core query engine** with relevance scoring, keyword matching, and context analysis
- **CLI tool** with `query`, `get`, `list`, `validate`, `serve`, and `version` commands
- **MCP server** for integration with AI coding assistants (Claude Code, etc.)
- **12 authorization threat patterns** (5 Tier A, 7 Tier B) covering:
  - Background job authorization context loss (TMKB-AUTHZ-001)
  - List/detail authorization inconsistency (TMKB-AUTHZ-002)
  - Soft-delete resurrection attack (TMKB-AUTHZ-003)
  - Tenant isolation via application logic (TMKB-AUTHZ-004)
  - User/account/resource ownership confusion (TMKB-AUTHZ-005)
  - Additional essential patterns (TMKB-AUTHZ-006 through 012)
- **Agent mode output**: Token-limited (<500 tokens), max 3 patterns, JSON format
- **Verbose mode output**: Unlimited tokens, max 10 patterns, comprehensive details
- **Pattern validation**: Structural and content validation for all YAML patterns
- **Baseline validation**: 6 independent tests across 3 providers (Anthropic, OpenAI, Google)
- **Enhanced validation**: 1 test demonstrating 100% improvement with TMKB context
- **CI/CD pipeline**: Tests with race detection, CodeQL scanning, dependency review
- **Documentation suite**: README, API docs, MCP integration guide, validation methodology
- **Security infrastructure**: CodeQL, golangci-lint, security policy

[0.1.0]: https://github.com/mark-chris/tmkb/releases/tag/v0.1.0
