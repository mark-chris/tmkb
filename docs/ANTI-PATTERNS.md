# Top 10 Authorization Anti-Patterns AI Agents Make

These are the most common authorization mistakes observed in AI-generated code across multiple providers and models. Each anti-pattern maps to one or more TMKB patterns.

## 1. Fire-and-Forget Background Jobs

**What happens:** Agent generates a background job that accepts only a resource ID. No user context, no tenant context, no re-authorization.

**Example:**
```python
# Agent generates this 100% of the time without TMKB
@celery.task
def process_file(file_id):
    file = File.query.get(file_id)  # Any file, any tenant
    # ... process without authorization check
```

**Impact:** Cross-tenant data access via queue injection or TOCTOU attacks.

**TMKB Pattern:** [TMKB-AUTHZ-001](../patterns/authorization/tier-a/TMKB-AUTHZ-001.yaml) - Background Job Authorization Context Loss

**Observed in:** 6/6 baseline runs across Anthropic, OpenAI, and Google models.

---

## 2. Webhook Payload Blind Trust

**What happens:** Agent verifies webhook signatures at the HTTP endpoint but passes raw payload data to background workers without re-verification.

**Example:**
```python
@app.route('/webhooks/stripe', methods=['POST'])
def stripe_webhook():
    verify_stripe_signature(request)  # Verified here
    data = request.get_json()
    process_stripe_event.delay(data)  # Raw data, no proof of verification

@celery.task
def process_stripe_event(data):
    # Trusts data completely -- no re-verification
    event_type = data.get('type')
```

**Impact:** Workers process unverified payloads; queue injection bypasses signature verification.

**TMKB Pattern:** TMKB-AUTHZ-001 (async boundary generalization)

**Observed in:** Run-6 baseline (webhook pattern test).

---

## 3. Authenticated Equals Authorized

**What happens:** Agent adds `@login_required` to endpoints but doesn't check whether the authenticated user can access the *specific* resource.

**Example:**
```python
@app.route('/files/<int:file_id>')
@login_required
def get_file(file_id):
    file = File.query.get_or_404(file_id)  # Any user's file!
    return jsonify(file.to_dict())
```

**Impact:** Horizontal privilege escalation -- any authenticated user can access any resource by ID.

**TMKB Pattern:** [TMKB-AUTHZ-005](../patterns/authorization/tier-a/TMKB-AUTHZ-005.yaml) - User/Account/Resource Ownership Confusion

---

## 4. List/Detail Authorization Mismatch

**What happens:** List endpoint filters by tenant, but detail endpoint doesn't (or vice versa).

**Example:**
```python
@app.route('/files')
@login_required
def list_files():
    files = File.query.filter_by(
        organization_id=current_user.organization_id  # Filtered
    ).all()

@app.route('/files/<int:file_id>')
@login_required
def get_file(file_id):
    file = File.query.get_or_404(file_id)  # NOT filtered
    return jsonify(file.to_dict())
```

**Impact:** Resources invisible in list view are accessible via direct ID access.

**TMKB Pattern:** [TMKB-AUTHZ-002](../patterns/authorization/tier-a/TMKB-AUTHZ-002.yaml) - List/Detail Authorization Inconsistency

---

## 5. Client-Trusted Tenant ID

**What happens:** Agent accepts the organization/tenant ID from the request body instead of deriving it from the authenticated session.

**Example:**
```python
@app.route('/files', methods=['POST'])
@login_required
def upload_file():
    org_id = request.json.get('organization_id')  # Client-provided!
    file = File(organization_id=org_id, ...)
```

**Impact:** User can associate resources with any organization by providing a different tenant ID.

**TMKB Pattern:** [TMKB-AUTHZ-004](../patterns/authorization/tier-a/TMKB-AUTHZ-004.yaml) - Tenant Isolation via Application Logic

**Observed in:** Run-5 baseline (Gemini), where all endpoints accepted client-provided org IDs.

---

## 6. Soft-Delete Ignorance in Async Processing

**What happens:** Agent implements soft-delete (setting `deleted_at` timestamp) but background jobs don't check this flag.

**Example:**
```python
@celery.task
def process_file(file_id):
    file = File.query.get(file_id)
    # Doesn't check: if file.deleted_at: return
    file.status = 'processing'  # Resurrects deleted file
```

**Impact:** Deleted resources are processed, potentially leaking data or violating retention policies.

**TMKB Pattern:** [TMKB-AUTHZ-003](../patterns/authorization/tier-a/TMKB-AUTHZ-003.yaml) - Soft-Delete Resurrection Attack

---

## 7. Security as "Production Concern"

**What happens:** Agent generates placeholder comments instead of actual security implementations.

**Example:**
```python
def login():
    user = User.query.filter_by(username=data['username']).first()
    # In production: Verify password hash here
    session['user_id'] = user.id
```

**Impact:** No actual security -- authentication and authorization are deferred indefinitely.

**Observed in:** Run-5 baseline (Gemini), which had multiple "In production" comments with no implementation.

---

## 8. Endpoint-Only Authorization Thinking

**What happens:** Agent treats authorization as an HTTP concern and doesn't consider other entry points (CLI scripts, management commands, data migrations, scheduled jobs).

**Example:** Thorough endpoint authorization but a management command that processes all files regardless of tenant:
```python
# management/commands/reprocess.py
def handle(self):
    for file in File.query.filter_by(status='pending').all():
        process_file.delay(file.id)  # No tenant context
```

**Impact:** Non-HTTP entry points bypass all authorization checks.

**TMKB Pattern:** TMKB-AUTHZ-001 (generalized to all async boundaries)

---

## 9. Missing Object Ownership on Mutations

**What happens:** Agent adds ownership checks on read endpoints but not on update/delete operations.

**Example:**
```python
@app.route('/files/<int:file_id>')
@login_required
def get_file(file_id):
    file = File.query.filter_by(
        id=file_id, organization_id=current_user.org_id
    ).first_or_404()  # Ownership check on read

@app.route('/files/<int:file_id>', methods=['DELETE'])
@login_required
def delete_file(file_id):
    file = File.query.get_or_404(file_id)  # No ownership check on delete!
    db.session.delete(file)
```

**Impact:** Users can delete resources belonging to other tenants.

**TMKB Pattern:** TMKB-AUTHZ-002, TMKB-AUTHZ-005

---

## 10. Authorization Logic Duplication Without Abstraction

**What happens:** Agent copies authorization checks into each endpoint without a shared helper, leading to inconsistencies as endpoints are added.

**Example:**
```python
# First endpoint: correct check
file = File.query.filter_by(id=file_id, org_id=current_user.org_id).first()

# Later endpoint: subtly different check
file = File.query.filter_by(id=file_id).first()
if file.user_id != current_user.id:  # Different field, different logic!
    abort(403)
```

**Impact:** Inconsistent authorization logic across endpoints; some resources accessible through weaker checks.

**TMKB Pattern:** TMKB-AUTHZ-002, TMKB-AUTHZ-004

---

## Summary

| # | Anti-Pattern | Observed Rate | Severity |
|---|---|---|---|
| 1 | Fire-and-forget background jobs | 6/6 (100%) | Critical |
| 2 | Webhook payload blind trust | 1/1 (100%) | High |
| 3 | Authenticated = Authorized | 1/6 (17%) | High |
| 4 | List/detail auth mismatch | 1/6 (17%) | High |
| 5 | Client-trusted tenant ID | 1/6 (17%) | Critical |
| 6 | Soft-delete ignorance in async | 6/6 (100%) | Medium |
| 7 | Security as "production concern" | 1/6 (17%) | Critical |
| 8 | Endpoint-only auth thinking | 6/6 (100%) | High |
| 9 | Missing ownership on mutations | 1/6 (17%) | High |
| 10 | Auth logic duplication | 4/6 (67%) | Medium |

Anti-patterns 1, 2, 6, and 8 are **systematic** -- they appear across all providers and models. Anti-patterns 3, 5, and 7 were specific to weaker models (Gemini in Run-5).
