# Top 10 Authorization Anti-Patterns AI Agents Make

AI coding agents (LLMs) are strong at syntax-level security — they add `@login_required`, parameterize SQL queries, and escape output. But they systematically fail at **architectural authorization**: patterns that require reasoning across system boundaries, execution contexts, and object relationships.

## Sourcing

All 10 anti-patterns are grounded in established vulnerability classes (CWE, OWASP) and security design review experience. Every pattern in TMKB uses a provenance type of `generalized_observation` — these are well-known authorization failure modes, not novel research or proprietary incident data.

Four of the 10 were additionally **validated as LLM blindspots** through empirical smoke tests across multiple providers (Anthropic, OpenAI, Google), models (Sonnet 4.5, Opus 4.6, GPT-5.2, Gemini), and application types (file upload, webhooks):

| # | Anti-Pattern | Empirical Evidence |
|---|-------------|-------------------|
| 1 | Fire-and-Forget Background Jobs | **Validated**: 6/6 baseline runs failed across 3 providers and 2 app types |
| 2 | List/Detail Authorization Mismatch | **Validated**: Observed in Run-5 (Gemini) |
| 4 | Tenant Filter in Some Queries, Not All | **Validated**: Observed in Run-5 (Gemini) |
| 5 | Org Membership != Resource Permission | **Validated**: Observed in Run-5 (Gemini) |
| 3, 6-10 | Remaining patterns | **Not yet tested**: Hypothesized as likely LLM blindspots based on the same architectural reasoning, but not yet empirically validated |

See the [cross-run comparison](../validation/smoke-test/baseline-cross-run-comparison.md) for full baseline test methodology and results.

---

## 1. Fire-and-Forget Background Jobs

**Pattern:** Agent adds auth to the HTTP endpoint, then enqueues a background job with only a resource ID. The worker processes any resource without re-checking authorization.

**What the agent writes:**
```python
@celery.task
def process_file(self, file_id):       # No user_id, no org_id
    file = File.query.get(file_id)     # Loads any file
    # ... processes without authorization
```

**The blindspot:** Agents treat async boundaries as implementation details, not trust boundaries. They don't recognize that a Celery worker runs in a different security context than the Flask request.

**Source:** CWE-862, CWE-863, OWASP API1:2023 | **Validated: 6/6 baseline runs failed** (3 providers, 4 models, 2 app types)

**TMKB pattern:** [TMKB-AUTHZ-001](../patterns/authorization/tier-a/TMKB-AUTHZ-001.yaml)

---

## 2. List/Detail Authorization Mismatch

**Pattern:** Agent correctly filters the list endpoint by tenant, but the detail endpoint fetches by primary key without verifying ownership.

**What the agent writes:**
```python
# List: filtered
files = File.query.filter_by(organization_id=current_user.org_id).all()

# Detail: unfiltered
file = File.query.get(file_id)         # Any file, any tenant
```

**The blindspot:** Agents implement list and detail views as independent code paths. They don't reason about whether the authorization contract is consistent across both.

**Source:** CWE-862, OWASP API1:2023 | **Validated: Observed in Run-5** (Gemini)

**TMKB pattern:** [TMKB-AUTHZ-002](../patterns/authorization/tier-a/TMKB-AUTHZ-002.yaml)

---

## 3. Soft-Delete Doesn't Mean Gone

**Pattern:** Agent adds `deleted_at` filtering to read queries but not to update or processing queries. Deleted resources can be modified or resurrected.

**What the agent writes:**
```python
# Read: checks deleted_at
files = File.query.filter(File.deleted_at.is_(None)).all()

# Update: doesn't check
file = File.query.get(file_id)         # Loads deleted files too
file.name = request.json['name']       # Modifies "deleted" resource
```

**The blindspot:** Agents treat soft-delete as a display concern (filter in listings) rather than an authorization boundary (resource should be inaccessible to all mutating operations).

**Source:** CWE-863, CWE-672 | **Not yet tested** — hypothesized LLM blindspot

**TMKB pattern:** [TMKB-AUTHZ-003](../patterns/authorization/tier-a/TMKB-AUTHZ-003.yaml)

---

## 4. Tenant Filter in Some Queries, Not All

**Pattern:** Agent adds tenant isolation to primary queries but misses it in joins, aggregations, search endpoints, or related-object lookups.

**What the agent writes:**
```python
# Primary query: filtered
files = File.query.filter_by(org_id=current_user.org_id).all()

# Aggregation: unfiltered
total = db.session.query(func.count(File.id)).scalar()  # All tenants

# Related lookup: unfiltered
comments = Comment.query.filter_by(file_id=file_id).all()  # Cross-tenant
```

**The blindspot:** Agents apply tenant filtering as a local concern per-endpoint rather than a system invariant. They miss that every query touching tenant-scoped data needs the filter — not just the obvious ones.

**Source:** CWE-863, CWE-284, OWASP API1:2023 | **Validated: Observed in Run-5** (Gemini)

**TMKB pattern:** [TMKB-AUTHZ-004](../patterns/authorization/tier-a/TMKB-AUTHZ-004.yaml)

---

## 5. Org Membership != Resource Permission

**Pattern:** Agent checks that a user belongs to an organization but doesn't check whether the user has permission to access the specific resource within that org.

**What the agent writes:**
```python
# Checks org membership
if current_user.org_id != file.org_id:
    abort(403)
# But doesn't check: is this user allowed to access THIS file?
# (e.g., private files, team-scoped resources, role-based access)
```

**The blindspot:** Agents conflate three distinct authorization questions: "Is the user in the org?", "Does the org own this resource?", and "Can this user act on this resource?" Checking one doesn't imply the others.

**Source:** CWE-863, CWE-639, OWASP API1:2023 | **Validated: Observed in Run-5** (Gemini)

**TMKB pattern:** [TMKB-AUTHZ-005](../patterns/authorization/tier-a/TMKB-AUTHZ-005.yaml)

---

## 6. Mass Assignment of Ownership Fields

**Pattern:** Agent uses `request.json` directly to update model attributes, allowing clients to overwrite ownership fields like `organization_id`, `created_by`, or `role`.

**What the agent writes:**
```python
data = request.get_json()
for key, value in data.items():
    setattr(file, key, value)          # Sets ANY field, including org_id
```

**The blindspot:** Agents focus on making the API functional (accept input, update model) without distinguishing user-settable fields from system-managed fields. They don't build an allowlist of mutable attributes.

**Source:** CWE-915 | **Not yet tested** — hypothesized LLM blindspot

**TMKB pattern:** [TMKB-AUTHZ-006](../patterns/authorization/tier-b/TMKB-AUTHZ-006.yaml)

---

## 7. Authenticated != Authorized (IDOR)

**Pattern:** Agent adds authentication (`@login_required`) and then fetches resources by ID from the URL without verifying the authenticated user is authorized to access that specific resource.

**What the agent writes:**
```python
@login_required
def get_file(file_id):
    file = File.query.get(file_id)     # Any authenticated user, any file
    return jsonify(file.to_dict())
```

**The blindspot:** Agents treat authentication as sufficient. Once a user is logged in, the agent trusts URL parameters as implicitly authorized. This is the classic IDOR pattern, but agents reproduce it because they reason about auth at the decorator level, not the data-access level.

**Source:** CWE-639, OWASP API1:2023 | **Not yet tested** — hypothesized LLM blindspot

**TMKB pattern:** [TMKB-AUTHZ-007](../patterns/authorization/tier-b/TMKB-AUTHZ-007.yaml)

---

## 8. Status Transitions Without Permission Checks

**Pattern:** Agent checks whether a user can update a resource, but doesn't check whether the user is allowed to make a specific state transition (e.g., draft -> published, pending -> approved).

**What the agent writes:**
```python
file.status = request.json['status']   # Any status value accepted
db.session.commit()                    # draft -> approved? Sure.
```

**The blindspot:** Agents treat status as just another field. They don't model that different transitions require different permissions (e.g., only admins can approve, only the author can submit for review).

**Source:** CWE-863, CWE-841 | **Not yet tested** — hypothesized LLM blindspot

**TMKB pattern:** [TMKB-AUTHZ-009](../patterns/authorization/tier-b/TMKB-AUTHZ-009.yaml)

---

## 9. Parent Access != Child Access

**Pattern:** Agent authorizes access to a parent resource but doesn't verify that the child resource actually belongs to that parent, or that the child has the same authorization requirements.

**What the agent writes:**
```python
# GET /projects/123/comments/456
project = Project.query.get(project_id)
if project.org_id != current_user.org_id:
    abort(403)
comment = Comment.query.get(comment_id)  # Doesn't verify comment belongs to project
```

**The blindspot:** Agents implement nested routes by independently loading parent and child. They don't enforce the relationship — comment 456 might belong to a completely different project.

**Source:** CWE-639 | **Not yet tested** — hypothesized LLM blindspot

**TMKB pattern:** [TMKB-AUTHZ-010](../patterns/authorization/tier-b/TMKB-AUTHZ-010.yaml)

---

## 10. Bulk Operations Skip Per-Item Auth

**Pattern:** Agent enforces authorization on single-item endpoints but the bulk endpoint operates on a list of IDs without checking each one.

**What the agent writes:**
```python
# Single: authorized
def delete_file(file_id):
    file = get_authorized_file(file_id)  # Checks ownership
    db.session.delete(file)

# Bulk: not authorized
def bulk_delete():
    ids = request.json['ids']
    File.query.filter(File.id.in_(ids)).delete()  # Deletes any file
```

**The blindspot:** Agents treat bulk operations as performance optimizations of single operations, not as new authorization surfaces. They skip per-item checks in favor of batch queries that bypass ownership validation.

**Source:** CWE-863 | **Not yet tested** — hypothesized LLM blindspot

**TMKB pattern:** [TMKB-AUTHZ-012](../patterns/authorization/tier-b/TMKB-AUTHZ-012.yaml)

---

## The Common Thread

All 10 anti-patterns share a root cause: **LLMs reason about authorization locally** (within a single function or endpoint) **but not across boundaries** (between execution contexts, between endpoints, between parent and child resources, between single and bulk operations).

TMKB exists to inject this cross-boundary reasoning at the point where code is being generated. See the [README](../README.md) for validation results demonstrating the difference.
