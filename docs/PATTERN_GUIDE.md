# Pattern Creation Guide

How to create new threat patterns for TMKB.

## Pattern Structure

Every pattern is a YAML file in `patterns/<category>/<tier>/` with a `threat_pattern` root key. Here's the complete structure:

```yaml
threat_pattern:
  # === Required Metadata ===
  id: "TMKB-AUTHZ-NNN"          # Unique identifier
  name: "Human-Readable Name"     # Short, descriptive name
  tier: "A"                       # A (full depth) or B (essential)
  version: "1.0.0"                # Semantic version
  last_updated: "2026-02-10"      # ISO date

  # === Scope Tags ===
  category: "authorization"       # Primary category
  subcategory: "async-boundaries" # Specific area
  language: "python"              # Primary language for examples
  framework: "flask"              # Primary framework for examples

  severity: "high"                # critical, high, medium, low
  likelihood: "high"              # high, medium, low

  # === Generalization ===
  generalizes_to:                 # Other frameworks this applies to
    - "Django + Celery"
    - "Node.js + Bull"

  # === Provenance ===
  provenance:
    source_type: "generalized_observation"
    description: >
      Where this pattern comes from and why it matters.
    public_references:
      - cwe: "CWE-862"
        name: "Missing Authorization"
        url: "https://cwe.mitre.org/data/definitions/862.html"

  # === Triggers ===
  triggers:
    keywords:                     # Words that should trigger this pattern
      - "background job"
      - "celery"
    actions:                      # Actions that should trigger this pattern
      - "creating background job"
    file_patterns:                # File paths that indicate relevance
      - "**/tasks.py"

  # === Differentiation ===
  differentiation:
    llm_knowledge_state: >
      What LLMs already know about this topic.
    tmkb_value: >
      What TMKB adds beyond LLM knowledge.
    llm_blindspots:
      - "Specific blindspot 1"
      - "Specific blindspot 2"

  # === Core Content ===
  description: |
    Detailed description of the threat pattern.

  agent_summary:
    threat: "One-line threat description (<100 tokens total)"
    check: "What to verify"
    fix: "How to fix it"

  # === Attack Scenario ===
  attack_scenario:
    narrative: |
      Detailed attack narrative with setup, vulnerability, and attack paths.
    preconditions:
      - "Precondition 1"
    attack_steps:
      - step: 1
        action: "Action description"
        detail: "Detailed explanation"
    impact:
      confidentiality: "high"
      integrity: "high"
      availability: "low"
      scope: "Impact scope description"
      business_impact: |
        Business-level impact description.

  # === Mitigations ===
  mitigations:
    - id: "MIT-AUTHZ-NNNa"
      name: "Mitigation name"
      description: |
        Detailed mitigation description.
      effectiveness: "high"
      implementation_effort: "medium"
      tradeoffs:
        - "Tradeoff 1"
      code_examples:
        - language: "python"
          framework: "flask-celery"
          description: "Example description"
          vulnerable_code: |
            # Vulnerable code example
          secure_code: |
            # Secure code example

  # === Security Principles ===
  security_principles:
    - principle: "Principle name"
      explanation: >
        Why this principle matters.

  # === Related Patterns ===
  related_patterns:
    - id: "TMKB-AUTHZ-NNN"
      relationship: "extends"     # extends, related, prerequisite
      description: "How they relate"

  # === Testing ===
  testing:
    manual_verification:
      - step: "What to check"
        check: "How to check it"
    automated_checks:
      - type: "static_analysis"
        description: "What to scan for"
        pattern: "regex or tool config"

  # === Validation ===
  validation:
    baseline_test:
      prompt: "The prompt used to test this pattern"
      expected_failure: "What should fail without TMKB"
      observed: "What actually happened"
      date: "2026-02-10"
```

## Tier Guidelines

### Tier A (Full Depth)

Full-depth patterns include:
- Complete attack scenarios with multiple attack paths
- 3+ mitigations with code examples (vulnerable + secure)
- Baseline validation test results
- Security principles explanation
- Testing guidance (manual + automated)

**When to use Tier A:**
- High severity + high likelihood
- Core architectural patterns
- Patterns with strong empirical evidence (baseline test failures)

### Tier B (Essential Coverage)

Essential patterns include:
- Core description and agent summary
- At least 1 mitigation with code example
- Trigger keywords for agent matching
- Differentiation (what TMKB adds)

**When to use Tier B:**
- Medium severity or likelihood
- Supporting patterns that complement Tier A
- Patterns where baseline testing is pending

## Naming Convention

Pattern IDs follow the format: `TMKB-{CATEGORY}-{NUMBER}`

- `TMKB-AUTHZ-001` through `TMKB-AUTHZ-NNN` for authorization patterns
- Numbers are sequential within each category
- Tier A: 001-005 (MVP), Tier B: 006-012 (MVP)

File naming: `TMKB-AUTHZ-NNN.yaml` in `patterns/authorization/tier-{a,b}/`

## Writing Effective Patterns

### Agent Summary

The `agent_summary` is what AI agents see in compact mode. Keep it under 100 tokens total:

```yaml
agent_summary:
  threat: "Background jobs execute without authorization context"
  check: "Verify authorization is re-checked in the job"
  fix: "Pass user_id and tenant_id to job; re-validate before processing"
```

### Triggers

Think about what an agent might be working on when this pattern is relevant:

```yaml
triggers:
  keywords:          # Terms in user prompts or code
    - "celery"
    - "background job"
  actions:           # What the agent is doing
    - "creating background job"
  file_patterns:     # Files being created/modified
    - "**/tasks.py"
```

### Code Examples

Always include both vulnerable and secure code:

```yaml
code_examples:
  - language: "python"
    framework: "flask-celery"
    description: "Authorization re-check in Celery task"
    vulnerable_code: |
      # Clear comment: WHY this is vulnerable
      @celery.task
      def process_file(file_id):
          file = File.query.get(file_id)  # No auth check
    secure_code: |
      # Clear comment: WHAT security checks are added and WHY
      @celery.task
      def process_file(file_id, user_id, organization_id):
          # Re-validate authorization in worker context
          file = File.query.get(file_id)
          if file.organization_id != organization_id:
              raise AuthorizationError("Tenant mismatch")
```

### Differentiation

This is critical -- explain what LLMs already know vs. what TMKB adds:

```yaml
differentiation:
  llm_knowledge_state: >
    LLMs know about @login_required and RBAC at endpoints.
  tmkb_value: >
    TMKB flags that async execution is a trust boundary
    requiring re-authorization.
  llm_blindspots:
    - "Passes only resource ID to background job"
    - "Assumes endpoint check covers the entire operation"
```

## Validation

### Before Submitting

1. **Validate structure**: `./bin/tmkb validate --all`
2. **Test query matching**: `./bin/tmkb query --context "<relevant context>" --verbose`
3. **Verify the pattern loads**: `./bin/tmkb get TMKB-AUTHZ-NNN`

### Baseline Testing (Recommended for Tier A)

If possible, run a baseline test:

1. Use the prompt from `validation/PROTOCOL.md`
2. Verify the AI agent reproduces the vulnerability
3. Document the result in the pattern's `validation` section

## Example: Minimal Tier B Pattern

```yaml
threat_pattern:
  id: "TMKB-AUTHZ-NNN"
  name: "Pattern Name"
  tier: "B"
  version: "1.0.0"
  last_updated: "2026-02-10"

  category: "authorization"
  subcategory: "async-boundaries"
  language: "python"
  framework: "flask"

  severity: "medium"
  likelihood: "medium"

  provenance:
    source_type: "generalized_observation"
    description: >
      Brief description of where this pattern comes from.
    public_references:
      - cwe: "CWE-862"
        name: "Missing Authorization"
        url: "https://cwe.mitre.org/data/definitions/862.html"

  triggers:
    keywords:
      - "relevant keyword"
    actions:
      - "relevant action"

  differentiation:
    llm_knowledge_state: >
      What LLMs already know.
    tmkb_value: >
      What TMKB adds.
    llm_blindspots:
      - "Specific blindspot"

  description: |
    Detailed description of the threat.

  agent_summary:
    threat: "One-line threat"
    check: "What to verify"
    fix: "How to fix"

  mitigations:
    - id: "MIT-AUTHZ-NNNa"
      name: "Primary mitigation"
      description: |
        How to mitigate the threat.
      effectiveness: "high"
      implementation_effort: "medium"
      code_examples:
        - language: "python"
          framework: "flask"
          description: "Example"
          vulnerable_code: |
            # Vulnerable code
          secure_code: |
            # Secure code
```
