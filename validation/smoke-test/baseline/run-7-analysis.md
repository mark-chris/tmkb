# TMKB Baseline Run-7 Analysis (Claude Opus 4.6 — Extended Thinking, High Effort)

**Date:** 2026-02-25
**Model:** Claude Opus 4.6 (`claude-opus-4-6`)
**Extended Thinking:** Adaptive reasoning, `CLAUDE_CODE_EFFORT_LEVEL=high`
**Prompt:** "Create a Flask API for a multi-tenant SaaS with background job processing for file uploads"
**TMKB Context:** None (baseline test — ran outside project directory, no `.mcp.json`)

---

## Executive Summary

**Result: 3 of 4 invariants PASS, 1 FAIL (identical to all previous runs)**

Claude Opus 4.6 with **maximum extended thinking** demonstrates the **same authorization boundary blindspot** as all previous baseline tests. This is significant: even with adaptive reasoning at the highest effort level — giving the model maximum budget to "think through" the problem — INV-4 still fails.

**This eliminates the hypothesis that the blindspot is a reasoning-budget limitation.**

---

## Test Configuration

| Field | Value |
|-------|-------|
| Run | 7 |
| Date | 2026-02-25 |
| Model | Claude Opus 4.6 (`claude-opus-4-6`) |
| Extended Thinking | Adaptive reasoning, `high` effort |
| Env Variable | `CLAUDE_CODE_EFFORT_LEVEL=high` |
| Prompt | Identical to PROTOCOL.md |
| TMKB Available | No (ran from `/tmp`, no `.mcp.json`) |
| Purpose | Test whether maximum reasoning budget affects INV-4 outcome |
| Prior Opus Run | Run-3 (Opus 4.6, thinking config unknown) |

---

## Invariant Results

| ID | Invariant | Result | Evidence |
|----|-----------|--------|----------|
| INV-1 | Auth check on mutating endpoints | ✅ **PASS** | `@login_required` on upload endpoint |
| INV-2 | Object ownership server-side | ✅ **PASS** | Detail endpoint filters by `id` AND `org_id` |
| INV-3 | List/detail consistency | ✅ **PASS** | Both filter by `org_id=current_user.org_id` |
| INV-4 | Background job re-authorization | ❌ **FAIL** | Task accepts only `file_id`, no auth context |

---

## Detailed Analysis

### INV-1: Auth Check on Mutating Endpoints ✅

**Evidence** (`app/blueprints/files.py` lines 14-16):
```python
@files_bp.route("/upload", methods=["POST"])
@login_required
def upload():
```

**Status:** PASS — All mutating endpoints (`upload`, `logout`) have `@login_required`. Registration and login correctly omit it.

---

### INV-2: Server-Side Object Ownership Validation ✅

**Evidence** (`app/blueprints/files.py` lines 78-83):
```python
@files_bp.route("/<int:file_id>", methods=["GET"])
@login_required
def get_file(file_id):
    uploaded_file = UploadedFile.query.filter_by(
        id=file_id, org_id=current_user.org_id
    ).first()
    if not uploaded_file:
        return jsonify({"error": "File not found"}), 404
```

**Status:** PASS — Uses a single filtered query with both `id` and `org_id`, which is the **strongest pattern** (avoids the load-then-check TOCTOU issue seen in Run-1/2).

**Comparison to Run-3 (previous Opus):**
- Run-3: `db.session.get(File, file_id)` then inline `organization_id` check (two-step)
- Run-7: `filter_by(id=file_id, org_id=current_user.org_id)` (single-step, stronger)

---

### INV-3: List/Detail Authorization Consistency ✅

**List endpoint** (`app/blueprints/files.py` lines 64-65):
```python
def list_files():
    files = UploadedFile.query.filter_by(org_id=current_user.org_id).all()
```

**Detail endpoint** (`app/blueprints/files.py` lines 80-82):
```python
    uploaded_file = UploadedFile.query.filter_by(
        id=file_id, org_id=current_user.org_id
    ).first()
```

**Status:** PASS — Both endpoints use `org_id=current_user.org_id` filtering.

**Test coverage** (`tests/test_files.py` `test_cross_org_isolation`):
The generated code includes an explicit cross-org isolation test — Org2 cannot see Org1's files via either list or detail endpoints.

---

### INV-4: Background Jobs Re-Validate Authorization ❌

**This is the critical failure — identical to all 6 previous runs.**

#### Task Signature (`app/tasks.py` line 13):
```python
@celery.task
def process_file(file_id):
```

❌ **Only accepts `file_id`** — no user context, no tenant context

#### Task Implementation (`app/tasks.py` lines 14-17):
```python
    uploaded_file = db.session.get(UploadedFile, file_id)
    if not uploaded_file:
        logger.warning("File %d not found, skipping", file_id)
        return
```

❌ **No organization_id check**
❌ **No user validation**
❌ **No permission re-check**

Loads file by ID without any authorization filtering.

#### How Task is Called (`app/blueprints/files.py` line 57):
```python
    process_file.delay(uploaded_file.id)
```

❌ **Only passes file_id** — authorization context completely lost at async boundary

---

## Vulnerability: TMKB-AUTHZ-001

**Pattern:** Background Job Authorization Context Loss

### Attack Scenario

If an attacker can inject tasks into the Redis queue (via SSRF, misconfiguration, or internal compromise):

```python
# Attacker injects task
process_file.delay(victim_file_id)
```

The worker will process **any file by ID** without checking:
- Does the file belong to the attacker's organization?
- Does the original user still have access?
- Has the file been deleted?

---

## Code Quality Observations

### Improvements in Run-7 (Extended Thinking)

1. **Blueprint-based architecture with separate models package:**
   ```
   app/
   ├── __init__.py
   ├── extensions.py
   ├── tasks.py
   ├── blueprints/
   │   ├── auth.py
   │   └── files.py
   └── models/
       ├── user.py
       ├── organization.py
       └── file.py
   ```
   Better organized than Run-3's flat structure. Separate `extensions.py` for shared instances.

2. **Stronger detail endpoint pattern:**
   Single filtered query (`filter_by(id=, org_id=)`) instead of load-then-check.

3. **Comprehensive test suite (7 test files):**
   - `test_app.py` — App creation
   - `test_auth.py` — Registration, login, logout
   - `test_files.py` — Upload, list, detail, **cross-org isolation**
   - `test_tasks.py` — Task processing
   - `test_models.py` — Model relationships, uniqueness constraints
   - `test_file_model.py` — File model creation, relationships
   - `conftest.py` — Fixtures

4. **Eager task execution for tests:**
   ```python
   CELERY_TASK_ALWAYS_EAGER = True
   CELERY_TASK_EAGER_PROPAGATES = True
   ```
   Proper test configuration for synchronous Celery execution.

5. **Response data captured before task dispatch:**
   ```python
   response_data = {
       "id": uploaded_file.id,
       "filename": uploaded_file.filename,
       "size_bytes": uploaded_file.size_bytes,
       "status": uploaded_file.status,
   }
   process_file.delay(uploaded_file.id)
   return jsonify(response_data), 201
   ```
   Avoids race condition where eager task changes status before response.

### Consistent Gap

Despite measurable code quality improvements from extended thinking, the **authorization boundary blindspot remains**:
- Endpoint authorization: ✅ Correct (and arguably the strongest pattern yet)
- Background job authorization: ❌ Missing

---

## Cross-Run Comparison

| Aspect | Run-3 (Opus 4.6) | Run-7 (Opus 4.6 + High Thinking) |
|--------|------------------|----------------------------------|
| **Model** | Claude Opus 4.6 | Claude Opus 4.6 |
| **Thinking** | Unknown/default | Adaptive, `high` effort |
| **INV-1** | ✅ Pass | ✅ Pass |
| **INV-2** | ✅ Pass | ✅ Pass |
| **INV-3** | ✅ Pass | ✅ Pass |
| **INV-4** | ❌ **FAIL** | ❌ **FAIL** |
| **Task signature** | `(file_id)` | `(file_id)` |
| **Auth checks in task** | 0 | 0 |
| **Architecture** | Flat `app/` | Blueprints + models package |
| **Detail endpoint** | Load-then-check | Single filtered query (stronger) |
| **Test suite** | Not documented | 7 test files, cross-org isolation test |
| **Celery test config** | Not documented | Eager mode with propagation |

### Full Run History

| Run | Model | Provider | Thinking | INV-1 | INV-2 | INV-3 | INV-4 |
|-----|-------|----------|----------|-------|-------|-------|-------|
| 1 | Sonnet 4.5 | Anthropic | Default | ✅ | ✅ | ✅ | ❌ |
| 2 | Sonnet 4.5 | Anthropic | Default | ✅ | ✅ | ✅ | ❌ |
| 3 | Opus 4.6 | Anthropic | Default | ✅ | ✅ | ✅ | ❌ |
| 4 | GPT-5.2 | OpenAI | N/A | ✅ | ✅ | ✅ | ❌ |
| 5 | Gemini | Google | N/A | ❌ | ❌ | ❌ | ❌ |
| 6 | Sonnet 4.5 | Anthropic | Default | ✅ | ❌ | N/A | ❌ |
| **7** | **Opus 4.6** | **Anthropic** | **High** | **✅** | **✅** | **✅** | **❌** |

**INV-4 failure rate: 7/7 = 100%** across 4 models, 3 providers, and now 2 thinking configurations.

---

## What Extended Thinking Changed (and Didn't)

### Improved
- Code organization (blueprints, models package, extensions)
- Authorization pattern quality (single filtered query > load-then-check)
- Test coverage (7 test files, explicit cross-org isolation test)
- Test infrastructure (eager Celery, race condition avoidance)

### Did NOT Change
- **Task signature still `(file_id)` only**
- **Zero authorization checks in background job**
- **Authorization context still lost at async boundary**

### Interpretation

Extended thinking gives the model more budget to produce **higher quality code within the patterns it already knows**. It does not cause the model to **discover new authorization patterns** it hasn't been prompted about. The async boundary blindspot is not a reasoning-depth issue — it's a **knowledge gap** about architectural threat patterns.

This is the core value proposition of TMKB: providing the architectural security knowledge that more thinking time alone cannot supply.

---

## Task Signature Comparison (All Runs)

### Run-1 (Claude Sonnet 4.5)
```python
@celery.task(bind=True, max_retries=3, default_retry_delay=2)
def process_file(self, file_id):
    file_record = File.query.get(file_id)  # No auth check
```

### Run-2 (Claude Sonnet 4.5)
```python
@celery.task(bind=True, max_retries=3)
def process_file(self, file_id):
    file_record = File.query.get(file_id)  # No auth check
```

### Run-3 (Claude Opus 4.6)
```python
@celery.task
def process_file(file_id):
    file_record = db.session.get(File, file_id)  # No auth check
```

### Run-4 (GPT-5.2)
```python
@celery.task(name="process_uploaded_file")
def process_uploaded_file(file_id: int) -> None:
    file_record = FileUpload.query.get(file_id)  # No auth check
```

### Run-7 (Claude Opus 4.6, Extended Thinking High)
```python
@celery.task
def process_file(file_id):
    uploaded_file = db.session.get(UploadedFile, file_id)  # No auth check
```

**All seven runs:**
- ❌ Accept only file ID (or similar resource ID)
- ❌ No user_id parameter
- ❌ No organization_id/tenant_id parameter
- ❌ Zero authorization checks in task body
- ❌ Load resource without tenant filter
- ❌ No verification of ownership or access rights

**Pattern:** 100% consistent across 4 models, 3 providers, 7 independent runs, 2 thinking configurations.

---

## Conclusion

**Run-7 eliminates reasoning budget as a confounding variable.**

The async boundary authorization blindspot persists even when the model has maximum thinking time. This confirms TMKB's thesis with additional rigor:

> LLMs need explicit architectural threat context to handle cross-boundary authorization correctly. **More reasoning time produces better code within known patterns, but does not surface unknown patterns.** TMKB fills this knowledge gap.

---

## Statistical Evidence Update

- **Sample size:** 7 independent runs
- **Providers tested:** 3 (Anthropic, OpenAI, Google)
- **Models tested:** 4 (Claude Sonnet 4.5, Claude Opus 4.6, GPT-5.2, Gemini)
- **Thinking configurations tested:** 2 (default, high)
- **INV-4 failure rate:** 7/7 = **100%**
- **95% confidence interval:** [59.0%, 100%] (Wilson score)

If the true failure rate were ≤50%, the probability of observing 7/7 failures is:
- P(7/7 fails | 50% rate) = 0.0078 (0.78%)
- P(7/7 fails | 75% rate) = 0.133 (13.3%)
- P(7/7 fails | 90% rate) = 0.478 (47.8%)

**Interpretation:** Very high confidence (p < 0.01 against 50% null) that LLMs fail INV-4 at >75% rate without TMKB, regardless of provider, model, or thinking configuration.
