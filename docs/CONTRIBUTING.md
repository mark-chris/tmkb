# Contributing to TMKB

Thank you for your interest in contributing to the Threat Model Knowledge Base. This guide will help you get started.

## Getting Started

### Prerequisites

- **Go 1.25+**: [Download Go](https://go.dev/dl/)
- **Git**: For version control
- **Task** (optional): [Task runner](https://taskfile.dev/) for build automation

### Setup

```bash
# Clone the repository
git clone https://github.com/mark-chris/tmkb.git
cd tmkb

# Install dependencies
go mod download

# Build
go build -o bin/tmkb ./cmd/tmkb

# Run tests
go test ./...

# Validate patterns
./bin/tmkb validate --all
```

Or using Task:
```bash
task setup
task build
task test
task validate
```

## Types of Contributions

### 1. New Threat Patterns

The most impactful contribution. See [PATTERN_GUIDE.md](PATTERN_GUIDE.md) for how to create patterns.

**Before creating a pattern:**
- Check existing patterns to avoid duplication
- Verify the pattern represents an **architectural** security concern (not syntax-level)
- Confirm it's a demonstrable LLM blindspot (ideally with a baseline test)

### 2. Language/Framework Examples

Existing patterns have Python/Flask examples. Adding examples for other languages and frameworks increases TMKB's usefulness:

- Node.js (Express + Bull/BullMQ)
- Go (net/http + Temporal/Asynq)
- Ruby (Rails + Sidekiq/ActiveJob)
- Java (Spring + @Async)

### 3. Validation Runs

Running baseline tests with different AI models strengthens the statistical evidence:

1. Follow the protocol in [PROTOCOL.md](../validation/PROTOCOL.md)
2. Use the standard prompt (no modifications)
3. Record the model, provider, version, and date
4. Analyze against the four invariants
5. Save all generated code
6. Submit analysis as a PR

### 4. Bug Fixes and Code Improvements

Standard code contributions:

- Fix bugs in the CLI, query engine, or MCP server
- Improve test coverage
- Optimize performance
- Fix documentation errors

### 5. Documentation

- Improve clarity of existing docs
- Add usage examples
- Fix broken links or outdated information

## Development Workflow

### Branch Naming

- `feature/description` -- New features
- `fix/description` -- Bug fixes
- `docs/description` -- Documentation only
- `patterns/description` -- New or updated patterns

### Making Changes

1. **Fork** the repository
2. **Create a branch** from `main`
3. **Make changes** with clear, focused commits
4. **Run tests**: `go test ./...`
5. **Run linter**: `golangci-lint run ./...`
6. **Validate patterns**: `./bin/tmkb validate --all`
7. **Submit a PR** with a clear description

### Commit Messages

Follow conventional commit style:

```
feat: add Node.js code examples for TMKB-AUTHZ-001
fix: correct Wilson score calculation in scoring.go
docs: update validation results with Run-7
patterns: add TMKB-AUTHZ-006 tier-b essential pattern
test: add integration tests for MCP query handler
```

### Pull Request Guidelines

- **Title**: Short, descriptive (under 70 characters)
- **Description**: Explain what and why (not just how)
- **Testing**: Describe how you tested the changes
- **Patterns**: If adding/modifying patterns, include `validate --all` output

## Code Style

### Go Code

- Follow standard Go conventions (`gofmt`, `goimports`)
- The project uses [golangci-lint](https://golangci-lint.run/) with 13 linters enabled
- See `.golangci.yml` for the linter configuration
- Run `golangci-lint run ./...` before submitting

### Pattern YAML

- Follow the structure in existing patterns (see [PATTERN_GUIDE.md](PATTERN_GUIDE.md))
- Use `validate --all` to check pattern structure
- Include both vulnerable and secure code examples

### Documentation

- Use Markdown
- Keep lines readable
- Include code examples where helpful
- Link to related files and patterns

## Testing

### Running Tests

```bash
# All tests
go test ./...

# With verbose output
go test -v ./...

# With race detection
go test -race ./...

# Specific package
go test ./internal/knowledge/...

# Pattern validation
./bin/tmkb validate --all
```

### Writing Tests

- Add tests for new functionality
- Use table-driven tests where appropriate
- Test both success and error paths
- Integration tests go in `*_integration_test.go` files

## Questions?

- Open a [GitHub Issue](https://github.com/mark-chris/tmkb/issues) for bugs or feature requests
- Use [GitHub Discussions](https://github.com/mark-chris/tmkb/discussions) for questions and ideas

## Code of Conduct

Be respectful, constructive, and professional. We're building security infrastructure -- accuracy and rigor matter more than speed.
