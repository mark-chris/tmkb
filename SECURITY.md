# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please use GitHub's private vulnerability reporting feature:

1. Navigate to the [Security tab](https://github.com/mark-chris/tmkb/security)
2. Click **"Report a vulnerability"**
3. Fill out the advisory form with:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

### What to Expect

- **Response Time:** You should receive an initial response within 48 hours
- **Private Discussion:** We'll collaborate through the private advisory
- **Coordinated Disclosure:** If confirmed, we'll work together to develop and release a fix before public disclosure
- **Credit:** You'll be credited in the security advisory (unless you prefer to remain anonymous)

### Why Private Reporting?

- **Protects Users:** Gives time to fix before attackers learn about the issue
- **Secure Collaboration:** Private workspace to discuss and develop fixes
- **Professional Process:** Follows responsible disclosure best practices
- **CVE Assignment:** Can request CVE if the vulnerability warrants it

## Security Features

This project uses multiple layers of security scanning:

### Automated Security Scanning
- **CodeQL:** Static analysis for security vulnerabilities
- **gosec:** Go-specific security scanning
- **Trivy:** Vulnerability and misconfiguration scanning
- **govulncheck:** Go vulnerability detection
- **Secret Scanning:** Detects accidentally committed secrets

### Dependency Management
- **Dependency Review:** Scans PRs for vulnerable dependencies
- **Dependabot Alerts:** Notifies of vulnerable dependencies

### Process Security
- **Required Status Checks:** All code must pass security scans before merge
- **SARIF Integration:** Security results visible in GitHub Security tab
- **Automated Updates:** Security patches applied promptly

## Security Best Practices

This project follows security best practices including:

- **Least Privilege:** Minimal permissions in CI/CD
- **Input Validation:** All external inputs validated
- **Secure Defaults:** Security features enabled by default
- **Regular Updates:** Dependencies kept current
- **Code Review:** All changes reviewed for security implications

## Scope

### In Scope
- Security vulnerabilities in TMKB code
- Vulnerabilities in direct dependencies
- Security issues in documentation that could mislead users
- Authentication/authorization bypasses
- Code injection vulnerabilities
- Path traversal issues
- Data exposure issues

### Out of Scope
- Vulnerabilities in dependencies of dependencies (report upstream)
- Issues in unsupported versions
- Social engineering attacks
- Physical security
- Denial of service through resource exhaustion (rate limiting is a feature)

## Security Contact

For security questions that don't require private reporting:
- Open a GitHub Discussion in the Security category
- Tag issues with the `security` label for public, non-sensitive security improvements

## Acknowledgments

We appreciate the security research community's efforts to responsibly disclose vulnerabilities. Contributors who report valid security issues will be acknowledged in:
- The security advisory
- Release notes for the fix
- This SECURITY.md file (if they wish)

Thank you for helping keep TMKB and its users safe!
