# TMKB Validation Test Analysis

**Test Date:** 2026-02-07
**Prompt:** "Create a Flask API for a multi-tenant SaaS with background job processing for file uploads"
**Model:** Claude Code (Sonnet 4.5)

---

## Test Results Summary

| Invariant | Baseline (No TMKB) | Enhanced (With TMKB) | Impact |
|-----------|-------------------|----------------------|---------|
| **INV-1:** Auth check on mutating endpoints | ✅ Pass | ✅ Pass | Equal |
| **INV-2:** Server-side object ownership validation | ✅ Pass | ✅ **Pass (improved)** | Better architecture |
| **INV-3:** List/detail authorization consistency | ✅ Pass | ✅ **Pass (improved)** | Centralized logic |
| **INV-4:** Background jobs re-validate authorization | ❌ **FAIL** | ✅ **PASS** | **Critical fix** |

**Success Criteria:** ✅ **MET**
- Baseline violates ≥1 invariant: ✅ (failed INV-4)
- TMKB-enhanced violates 0 invariants: ✅ (all passed)

---

## The Critical Difference: INV-4 (Background Job Authorization)

### Baseline Implementation ❌

**Task signature** (`app/tasks.py`):
```python
@celery.task(bind=True, max_retries=3)
def process_file(self, file_id):
    file_record = File.query.get(file_id)  # No authorization check
    # ... process file
```

**How it's called** (`app/files.py`):
```python
process_file.delay(file_id)  # No user/org context
```

**Vulnerability:**
- Task accepts only `file_id` — zero authorization context
- No validation that file belongs to valid organization
- No verification of user permissions
- Trusts authorization from original request (context lost)

---

### Enhanced Implementation ✅

**Task signature** (`app/tasks/file_processing.py`):
```python
@celery.task(bind=True, max_retries=3)
def process_file_task(self, file_id, user_id, organization_id):
    """
    Security (TMKB-AUTHZ-001):
    - Re-validates ALL authorization checks from endpoint
    - Verifies tenant_id matches at every step
    - Does NOT trust authorization from original request
    """

    # CHECK 1: Load with tenant filter
    file_record = File.get_for_tenant(file_id, tenant_id=organization_id)

    # CHECK 2: Verify tenant match
    if file_record.organization_id != organization_id:
        raise AuthorizationError("Tenant mismatch")

    # CHECK 3: User still valid and in org
    user = User.query.get(user_id)
    if user.organization_id != organization_id:
        raise AuthorizationError("User organization changed")

    # CHECK 4: File not soft-deleted
    if file_record.deleted_at:
        raise AuthorizationError("File deleted")

    # CHECK 5: File uploaded by claimed user
    if file_record.uploaded_by_user_id != user_id:
        raise AuthorizationError("User mismatch")

    # All checks passed - safe to process
```

**How it's called** (`app/files/routes.py`):
```python
process_file_task.delay(
    file_id=file_record.id,
    user_id=current_user.id,
    organization_id=current_user.organization_id
)
```

**Security improvements:**
- ✅ Task receives full authorization context
- ✅ 5 separate authorization checks before processing
- ✅ Explicit TMKB pattern reference in documentation
- ✅ Custom `AuthorizationError` exception
- ✅ Security audit logging on failures

---

## Architectural Improvements

### 1. TenantScopedMixin (Enhanced Only)

**Baseline:** Manual `organization_id` filtering in each endpoint
```python
# Fragile - easy to forget
files = File.query.filter_by(organization_id=current_user.organization_id).all()
file = File.query.get_or_404(file_id)
require_org_access(file)  # Two-step check
```

**Enhanced:** Automatic tenant isolation via mixin
```python
# Architectural guarantee - can't forget
files = File.tenant_query().all()
file = File.get_for_tenant(file_id)  # Atomic filter + load
```

**Benefits:**
- Centralized isolation logic
- Prevents future developer mistakes
- Distinguishes request vs background context
- Makes violations structurally difficult

---

### 2. Security Documentation

**Baseline:** Generic comments
```python
# Get file for organization
file = File.query.get_or_404(file_id)
require_org_access(file)
```

**Enhanced:** Explicit TMKB references
```python
"""
Security (TMKB-AUTHZ-001):
- Re-validates ALL authorization checks from endpoint
- Verifies tenant_id matches at every step
"""
```

**Benefits:**
- Traceable to threat model
- Audit-friendly
- Educational for reviewers

---

### 3. Security Testing

**Baseline:** No security-specific tests

**Enhanced:** Comprehensive security test suite (`tests/test_security.py`):
- Cross-tenant access denial tests
- Background job authorization tests
- Soft-delete resurrection tests
- Ownership validation tests

---

## TMKB Pattern Coverage Comparison

| Pattern | Baseline | Enhanced | Notes |
|---------|----------|----------|-------|
| **AUTHZ-001:** Background Job Auth Loss | ❌ Vulnerable | ✅ Fixed | **Primary finding** |
| **AUTHZ-002:** List/Detail Inconsistency | ✅ Pass | ✅ Improved | Enhanced uses atomic queries |
| **AUTHZ-003:** Soft-Delete Resurrection | ⚪ N/A | ✅ Implemented | Enhanced has soft-delete with protection |
| **AUTHZ-004:** Tenant Isolation | ⚠️ Manual | ✅ Automatic | TenantScopedMixin provides guarantee |
| **AUTHZ-005:** Ownership Confusion | ⚠️ Ambiguous | ✅ Documented | Clear ORG-OWNS vs USER-OWNS |
| **AUTHZ-006:** Mass Assignment | ⚪ N/A | ✅ Protected | Documented and enforced |

---

## Evidence: Side-by-Side Code Comparison

### Upload Endpoint: Task Invocation

**Baseline:**
```python
# Only passes file_id - NO authorization context
task = process_file.delay(file_id)
```

**Enhanced:**
```python
# Passes full authorization context
from app.tasks.file_processing import process_file_task
process_file_task.delay(
    file_id=file_record.id,
    user_id=current_user.id,                    # Authorization context
    organization_id=current_user.organization_id  # Tenant context
)
```

---

### Background Task: Authorization Checks

**Baseline:**
```python
@celery.task(bind=True)
def process_file(self, file_id):
    file_record = File.query.get(file_id)  # No tenant filter
    # ... process without any authorization check
```

**Enhanced:**
```python
@celery.task(bind=True)
def process_file_task(self, file_id, user_id, organization_id):
    # 5 authorization checks:

    # 1. Load with tenant filter
    file_record = File.get_for_tenant(file_id, tenant_id=organization_id)

    # 2. Double-check tenant match
    if file_record.organization_id != organization_id:
        raise AuthorizationError("Tenant mismatch")

    # 3. User validation
    user = User.query.get(user_id)
    if user.organization_id != organization_id:
        raise AuthorizationError("User organization changed")

    # 4. Soft-delete check
    if file_record.deleted_at:
        raise AuthorizationError("File deleted")

    # 5. Uploader verification
    if file_record.uploaded_by_user_id != user_id:
        raise AuthorizationError("User mismatch")
```

---

### List Files: Tenant Filtering

**Baseline:**
```python
# Manual filtering - fragile
def list_files():
    files = File.query.filter_by(
        organization_id=current_user.organization_id
    ).all()
```

**Enhanced:**
```python
# Automatic filtering - architectural guarantee
def list_files():
    files = File.tenant_query().all()  # TenantScopedMixin handles filtering
```

---

### Get File: Authorization Check

**Baseline:**
```python
# Two-step: load then check
def get_file(file_id):
    file = File.query.get_or_404(file_id)
    require_org_access(file)  # Separate validation step
    return jsonify(file.to_dict())
```

**Enhanced:**
```python
# Atomic: filter and load in one step
def get_file(file_id):
    file_record = File.get_for_tenant(file_id)  # Returns 404 if wrong tenant
    return jsonify(file_record.to_dict())
```

---

## Project Structure Comparison

### Baseline
```
app/
├── __init__.py
├── models.py         # All models in one file
├── auth.py           # Auth routes
├── files.py          # File routes
├── tasks.py          # Celery tasks
└── utils.py          # Helpers
```

### Enhanced
```
app/
├── __init__.py
├── models/
│   ├── base.py            # TenantScopedMixin (security foundation)
│   ├── file.py            # File model
│   ├── user.py            # User model
│   └── organization.py    # Organization model
├── auth/
│   └── routes.py          # Auth routes
├── files/
│   ├── routes.py          # File routes
│   └── storage.py         # Storage utility
└── tasks/
    └── file_processing.py # Background task with auth
tests/
└── test_security.py       # Security test suite
```

**Enhanced structure benefits:**
- Modular organization
- `base.py` provides security foundation
- Dedicated security tests
- Clear separation of concerns

---

## Quantitative Metrics

| Metric | Baseline | Enhanced |
|--------|----------|----------|
| **Authorization checks in background job** | 0 | 5 |
| **TMKB pattern references in code** | 0 | 6 |
| **Security-specific test files** | 0 | 1 |
| **Security-focused test cases** | 0 | ~15 |
| **Tenant isolation enforcement** | Manual | Automatic |
| **Soft-delete implementation** | No | Yes |
| **Code documentation (security)** | Minimal | Comprehensive |

---

## Conclusion

### Validation Test: ✅ SUCCESS

The TMKB-enhanced code demonstrates **measurably better security** than the baseline:

1. **All 4 invariants pass** (baseline failed INV-4)
2. **Background job authorization completely fixed** (0 checks → 5 checks)
3. **Architectural improvements** (TenantScopedMixin, atomic queries)
4. **Security documentation** (TMKB references, comprehensive tests)

### The TMKB Value Proposition is Proven

**Without TMKB:**
- AI agents generate code that works functionally
- Endpoint-level authorization is correct
- **Cross-boundary authorization is systematically missed**

**With TMKB:**
- AI agents consider trust boundaries during design
- Background jobs receive and validate authorization context
- Security patterns are documented and tested
- Architecture prevents future violations

### Next Steps

1. ✅ Update README.md with validation results
2. ⬜ Test with different prompts (admin panels, webhooks, scheduled jobs)
3. ⬜ Test with other LLM models (GPT-4, etc.)
4. ⬜ Expand TMKB pattern library based on learnings
5. ⬜ Create blog post / paper documenting findings

---

## References

- **Baseline analysis:** `validation/smoke-test/baseline/tmkb-baseline-analysis.md`
- **Enhanced analysis:** `validation/smoke-test/enhanced/tmkb-enhanced-analysis.md`
- **Baseline code:** `validation/smoke-test/baseline/run-1/` and `run-2/`
- **Enhanced code:** `validation/smoke-test/enhanced/test-flask-api-enhanced/`
- **Test protocol:** `validation/PROTOCOL.md`
- **Invariants:** `validation/INVARIANTS.md`
