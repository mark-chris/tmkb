# Data Provenance and Disclaimer

## Pattern Sources

TMKB patterns are derived from **generalized security observations**, not from proprietary threat intelligence or specific incident data.

### Source Types

| Source Type | Description | Example |
|-------------|-------------|---------|
| Generalized observation | Common architectural pattern observed across many applications | Background jobs losing auth context |
| Public references | Published CWE, OWASP, and security research | CWE-862, API1:2023 |
| AI baseline testing | Empirical testing of AI-generated code | 6 baseline runs across 3 providers |

### What TMKB Patterns Are

- **Architectural patterns**: Common ways authorization fails at system boundaries
- **Generalized from public knowledge**: Based on well-known security principles (defense in depth, least privilege, zero trust at boundaries)
- **Empirically validated**: Each pattern includes baseline test results showing AI agents reproduce the vulnerability

### What TMKB Patterns Are Not

- **Not proprietary threat intelligence**: No pattern is based on a specific company's internal incidents
- **Not vulnerability disclosures**: TMKB doesn't disclose new vulnerabilities -- it documents well-known patterns
- **Not derived from private data**: All references are public (CWE, OWASP, published research)
- **Not penetration test results**: Patterns are architectural, not derived from testing specific systems

## Public References

All TMKB patterns reference public standards:

### CWE (Common Weakness Enumeration)

- [CWE-862: Missing Authorization](https://cwe.mitre.org/data/definitions/862.html)
- [CWE-863: Incorrect Authorization](https://cwe.mitre.org/data/definitions/863.html)

### OWASP

- [API1:2023 - Broken Object Level Authorization](https://owasp.org/API-Security/editions/2023/en/0xa1-broken-object-level-authorization/)

### Security Principles

Patterns encode established security principles:
- **Authorization at point of action**: Check permissions where the action happens, not just at the entry point
- **Async boundaries are trust boundaries**: Context changes when crossing process boundaries
- **Never trust parameters across process boundaries**: Re-validate, don't assume
- **Defense in depth**: Layer multiple controls

## Validation Data

### Baseline Test Data

All baseline test results are generated from AI coding agents using standard prompts. The generated code is:

- **Ephemeral**: Created for testing purposes only, not deployed
- **Anonymized**: No real user data, organizations, or credentials
- **Archived**: Stored as zip files in `validation/smoke-test/baseline/`
- **Reproducible**: Anyone can reproduce using the documented prompts and protocols

### Models Tested

| Provider | Model | Date Tested |
|----------|-------|-------------|
| Anthropic | Claude Sonnet 4.5 | Feb 3-8, 2026 |
| Anthropic | Claude Opus 4.6 | Feb 7, 2026 |
| OpenAI | GPT-5.2 | Feb 8, 2026 |
| Google | Gemini | Feb 8, 2026 |

Test results reflect model behavior at the time of testing. Model behavior may change with updates.

## Disclaimer

### No Guarantee of Completeness

TMKB covers a focused subset of authorization patterns. Using TMKB does not guarantee that generated code is free of all security vulnerabilities. Always conduct security reviews and testing appropriate for your application's risk profile.

### No Guarantee of Accuracy

While patterns are based on established security principles and empirically validated, they may not apply to every application architecture. Patterns should be interpreted in the context of your specific system design.

### AI-Generated Code Requires Review

Even with TMKB, AI-generated code should be reviewed by qualified developers before deployment. TMKB improves the baseline but doesn't replace human security judgment.

### Model Behavior Changes

Baseline test results reflect AI model behavior at specific points in time. Model updates may change which invariants pass or fail. The structural blindspot (async boundary authorization) has been consistent across all tested models and providers as of February 2026.

## License

TMKB is released under the MIT License. See [LICENSE](../LICENSE) for details.

Pattern data, validation results, and documentation may be used, modified, and distributed under the same license.
