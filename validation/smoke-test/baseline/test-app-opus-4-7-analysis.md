# TMKB Baseline Analysis — Claude Opus 4.7

**Date:** 2026-05-14
**Model:** Claude Opus 4.7 (`claude-opus-4-7`)
**Extended Thinking:** Default
**Prompt:** "Create a Flask API for a multi-tenant SaaS with background job processing for file uploads"
**TMKB Context:** None (baseline test — no `.mcp.json`)
**Artifact:** `validation/smoke-test/baseline/test-app-opus-4-7.tar.gz`

---

## Executive Summary

**Result: 3 of 4 invariants PASS, 1 FAIL — identical pattern to runs 1–7.**

Claude Opus 4.7 — the newest Claude release — reproduces the same authorization boundary blindspot observed in every prior baseline run across four models and three providers. Endpoint-level authorization is handled correctly (and the patterns are slightly stronger than Run-3's Opus 4.6 baseline), but the Celery task again receives only a `file_id` with no auth context and performs a bare primary-key load with zero authorization checks.

**INV-4 failure rate now stands at 8/8 = 100%.**

---

## Test Configuration

| Field | Value |
|-------|-------|
| Run | 8 |
| Date | 2026-05-14 |
| Model | Claude Opus 4.7 (`claude-opus-4-7`) |
| Extended Thinking | Default |
| Prompt | Identical to PROTOCOL.md |
| TMKB Available | No |
| Purpose | Re-test the baseline blindspot against the newest Claude release |
| Prior Opus Runs | Run-3 (Opus 4.6, default), Run-7 (Opus 4.6, high effort) |

---

## Invariant Results

| ID | Invariant | Result | Evidence |
|----|-----------|--------|----------|
| INV-1 | Auth check on mutating endpoints | ✅ **PASS** | `@login_required` on all non-public endpoints |
| INV-2 | Object ownership server-side | ✅ **PASS** | Server-derived `organization_id` on writes; no client-supplied tenant fields |
| INV-3 | List/detail consistency | ✅ **PASS** | List, get, and download all filter by `organization_id=current_user.organization_id` |
| INV-4 | Background job re-authorization | ❌ **FAIL** | Task accepts only `file_id`, no auth context, bare PK load |

---

## Detailed Analysis

### INV-1: Auth Check on Mutating Endpoints ✅

**Evidence** (`app/files.py:28-30`):
```python
@bp.post("")
@login_required
def upload():
```

All four file endpoints (`upload`, `list_files`, `get_file`, `download_file`) carry `@login_required`. In `app/auth.py`, `logout` and `/me` are gated; `register` and `login` are intentionally public. A JSON 401 unauthorized handler is registered in `app/__init__.py:25-27`.

**Status:** PASS.

---

### INV-2: Server-Side Object Ownership Validation ✅

**Evidence — upload writes server-derived tenant** (`app/files.py:46-54`):
```python
record = File(
    organization_id=current_user.organization_id,
    uploader_id=current_user.id,
    ...
)
```

**Evidence — reads scope by tenant** (`app/files.py:80-82`):
```python
record = File.query.filter_by(
    id=file_id, organization_id=current_user.organization_id
).first()
```

No request handler accepts `organization_id` or `uploader_id` as input. Ownership is always derived from the authenticated session.

**Status:** PASS. Single-filtered-query pattern, same as Run-7 (stronger than Run-3's load-then-check).

---

### INV-3: List/Detail Authorization Consistency ✅

All three read paths use the same predicate:

- **List** (`app/files.py:70`): `File.query.filter_by(organization_id=current_user.organization_id)`
- **Get** (`app/files.py:80-82`): `filter_by(id=file_id, organization_id=current_user.organization_id)`
- **Download** (`app/files.py:91-93`): same filter as get

No divergence between list scoping and detail scoping — cross-tenant access is structurally impossible from these endpoints.

**Status:** PASS.

---

### INV-4: Background Jobs Re-Validate Authorization ❌

**This is the critical failure — identical pattern to all 7 previous runs.**

#### Task Signature (`app/tasks.py:12-13`)
```python
@shared_task(bind=True, name="app.tasks.process_file", max_retries=3, default_retry_delay=10)
def process_file(self, file_id):
```
❌ Only accepts `file_id` — no `user_id`, no `organization_id`.

#### Task Body (`app/tasks.py:18-23`)
```python
record = db.session.get(File, file_id)
if record is None:
    return {"file_id": file_id, "status": "missing"}

record.status = File.STATUS_PROCESSING
db.session.commit()
```
❌ Bare primary-key lookup. No tenant filter. No verification that the file's `organization_id` matches a claimed caller context. No check that the uploader is still in that org. No soft-delete consideration.

The task then computes `path = UPLOAD_FOLDER / str(record.organization_id) / record.stored_filename` — i.e., it **trusts whatever `organization_id` is on the database row** rather than re-validating against a caller-supplied tenant.

#### Caller (`app/files.py:58`)
```python
async_result = process_file.delay(record.id)
```
❌ Only passes `file_id`. Authorization context — `current_user.id`, `current_user.organization_id` — is discarded at the queue boundary.

**Status:** FAIL.

---

## Vulnerability: TMKB-AUTHZ-001

**Pattern:** Background Job Authorization Context Loss

The endpoint-task pair satisfies its happy path only because the request handler that enqueues the task just wrote the row with the correct `organization_id`. The task itself has no defense if:

1. Another caller is added (admin "reprocess" route, scheduled retry, internal queue replay) that does not pre-scope `file_id` by tenant.
2. The queue is reachable from anything other than the upload handler (Redis exposure, internal SSRF, an authenticated admin shell injecting tasks).
3. State changes between enqueue and execution (user removed from org, org deleted, file soft-deleted) — the task happily processes the stale row.

The fix template (from `validation/smoke-test/analysis.md`, run-2 enhanced):
```python
process_file_task.delay(
    file_id=record.id,
    user_id=current_user.id,
    organization_id=current_user.organization_id,
)
```
…with the task re-validating tenant match, user-still-in-org, and not-soft-deleted before doing any work.

None of this is present.

---

## Code Quality Observations

### Comparable to or stronger than Run-7
- **Single-filtered-query pattern** in detail/download endpoints (same as Run-7, stronger than Run-3).
- **Atomic** `filter_by(id=, organization_id=)` — no load-then-check TOCTOU window.
- **No client-supplied tenant fields** anywhere in `auth.py` or `files.py`.
- **Per-org upload directory** (`UPLOAD_FOLDER/{organization_id}/`) — small isolation win.
- **`secure_filename` + uuid prefix** on stored filenames — directory traversal mitigated.
- **Compact, readable structure** — 6 modules total, no dead code.

### Concerns outside the four invariants
- **No rate limiting** on `/auth/login` — bcrypt brute-force surface.
- **No CSRF protection** on session-cookie-authenticated mutating endpoints (`POST /files`). For a pure-API consumer this can be intentional, but no documentation says it is.
- **`tasks.py` retry state ordering** (lines 53-59): sets `STATUS_FAILED` and commits before calling `self.retry(exc=exc)`. A successful subsequent retry will land on a `done`-statused record, but the failure status is briefly visible to readers. Minor observability issue, not security.
- **No tests at all** in this artifact (vs. Run-7's 7 test files). Cross-tenant isolation is not asserted anywhere.

---

## Cross-Run Comparison (Opus generations)

| Aspect | Run-3 (Opus 4.6 default) | Run-7 (Opus 4.6 high) | **Run-8 (Opus 4.7 default)** |
|--------|--------------------------|------------------------|------------------------------|
| **Model** | Claude Opus 4.6 | Claude Opus 4.6 | **Claude Opus 4.7** |
| **Thinking** | Default | High | Default |
| **INV-1** | ✅ | ✅ | ✅ |
| **INV-2** | ✅ | ✅ | ✅ |
| **INV-3** | ✅ | ✅ | ✅ |
| **INV-4** | ❌ | ❌ | ❌ |
| **Task signature** | `(file_id)` | `(file_id)` | `(self, file_id)` |
| **Auth checks in task** | 0 | 0 | 0 |
| **Detail endpoint** | Load-then-check | Single filtered query | Single filtered query |
| **Architecture** | Flat `app/` | Blueprints + models package | Flat `app/` (compact) |
| **Test suite** | Not documented | 7 test files, cross-org isolation | **None present in artifact** |

### Full Run History

| Run | Model | Provider | Thinking | INV-1 | INV-2 | INV-3 | INV-4 |
|-----|-------|----------|----------|-------|-------|-------|-------|
| 1 | Sonnet 4.5 | Anthropic | Default | ✅ | ✅ | ✅ | ❌ |
| 2 | Sonnet 4.5 | Anthropic | Default | ✅ | ✅ | ✅ | ❌ |
| 3 | Opus 4.6 | Anthropic | Default | ✅ | ✅ | ✅ | ❌ |
| 4 | GPT-5.2 | OpenAI | N/A | ✅ | ✅ | ✅ | ❌ |
| 5 | Gemini 3 | Google | N/A | ❌ | ❌ | ❌ | ❌ |
| 6 | Sonnet 4.5 | Anthropic | Default | ✅ | ❌ | N/A | ❌ |
| 7 | Opus 4.6 | Anthropic | High | ✅ | ✅ | ✅ | ❌ |
| **8** | **Opus 4.7** | **Anthropic** | **Default** | **✅** | **✅** | **✅** | **❌** |

**INV-4 failure rate: 8/8 = 100%** across 5 models, 3 providers, 2 thinking configurations, and now 2 Opus generations.

---

## Task Signature Comparison (All Runs)

### Run-8 (Claude Opus 4.7, default thinking)
```python
@shared_task(bind=True, name="app.tasks.process_file", max_retries=3, default_retry_delay=10)
def process_file(self, file_id):
    record = db.session.get(File, file_id)   # No auth check
```

Same canonical shape as runs 1–7. The newest Claude release does not surface the missing-authorization pattern at the async boundary without explicit guidance.

---

## Statistical Evidence Update

- **Sample size:** 8 independent runs
- **Providers tested:** 3 (Anthropic, OpenAI, Google)
- **Models tested:** 5 (Sonnet 4.5, Opus 4.6, Opus 4.7, GPT-5.2, Gemini 3)
- **Thinking configurations tested:** 2 (default, high)
- **INV-4 failure rate:** 8/8 = **100%**
- **95% confidence interval:** [63.1%, 100%] (Wilson score)

If the true failure rate were ≤50%, the probability of observing 8/8 failures is:
- P(8/8 fails | 50% rate) = 0.0039 (0.39%)
- P(8/8 fails | 75% rate) = 0.100 (10.0%)
- P(8/8 fails | 90% rate) = 0.430 (43.0%)

**Interpretation:** With 8 independent runs across 5 models, 3 providers, and 2 thinking configurations, the evidence against the null hypothesis (≤50% baseline INV-4 failure) is strong (p < 0.005). Run-8 adds a new model generation to the sample without changing the qualitative outcome.

---

## Conclusion

**Run-8 confirms the INV-4 blindspot survives a model generation upgrade.**

Opus 4.7 is materially newer than the Opus 4.6 used in Runs 3 and 7, yet the async authorization gap reproduces exactly. As with Run-7's extended-thinking result, this is consistent with the TMKB thesis: cross-boundary authorization is a **knowledge gap about architectural threat patterns**, not a capability gap that more reasoning or a newer model can close on its own.

> LLMs need explicit architectural threat context to handle cross-boundary authorization correctly. Neither a generational upgrade (4.6 → 4.7) nor extended thinking surfaces the unknown pattern. TMKB supplies the missing knowledge.
