# TMKB Enhanced Analysis — Claude Opus 4.7 (clean same-model A/B)

**Date:** 2026-06-10
**Model:** Claude Opus 4.7 (`claude-opus-4-7`)
**Prompt:** Identical to `validation/PROTOCOL.md`
**TMKB Context:** Enabled — `mcp__tmkb__tmkb_query` wired via `CLAUDE.md` guidance **and** a `PreToolUse` hook that auto-runs `tmkb_query` before every code edit
**Superpowers skill:** **OFF**
**Artifact:** `validation/smoke-test/enhanced/flask-file-upload-enhanced.tar.gz`

---

## Why this run matters: it is the first clean A/B

Every prior enhanced result compared a *Sonnet 4.5 enhanced* run against a *Sonnet baseline*. That comparison was also confounded by a second variable — the **superpowers** skill framework (TDD, brainstorming, verification-before-completion) was ON for that enhanced run but not described for the baseline.

This run removes the confound. It pairs directly against **Run-8** — the Claude Opus 4.7 *baseline* (`baseline/test-app-opus-4-7-analysis.md`):

| Dimension | Run-8 (baseline) | This run (enhanced) |
|-----------|------------------|---------------------|
| Model | Claude Opus 4.7 | Claude Opus 4.7 |
| Prompt | PROTOCOL.md | PROTOCOL.md (identical) |
| Superpowers skill | OFF | OFF |
| **TMKB** | **OFF** | **ON** |

**TMKB is the only variable that differs.** Any difference in the four invariants is therefore attributable to TMKB, not to the model generation, the reasoning budget, or a process framework.

---

## Executive Summary

**Result: ALL 4 INVARIANTS PASS ✅ — Run-8 (same model, no TMKB) fails INV-4.**

| Invariant | Run-8 (Opus 4.7, no TMKB) | This run (Opus 4.7, TMKB) | Evidence |
|-----------|---------------------------|----------------------------|----------|
| INV-1 Auth on mutating endpoints | ✅ PASS | ✅ PASS | `@login_required` on upload/delete; `create_user` admin-gated |
| INV-2 Object ownership server-side | ✅ PASS | ✅ PASS | `File.get_for_tenant()` → `None` → 404 |
| INV-3 List/detail consistency | ✅ PASS | ✅ PASS | Both paths go through `TenantScopedMixin` |
| **INV-4 Background job re-auth** | ❌ **FAIL** | ✅ **PASS** | Task takes `(file_id, user_id, organization_id)` + 4 re-validation checks |

The discriminator (INV-4) flips FAIL → PASS on the **identical model**.

---

## Invariant Evidence

### INV-1: Auth on mutating endpoints ✅

`app/files/routes.py:22-24` — upload requires auth:
```python
@bp.post("")
@login_required
def upload_file():
```
`delete_file` (`routes.py:102-104`) is also `@login_required`. The admin-only `create_user` endpoint (`app/auth/routes.py:89`) gates on `current_user.is_admin` and returns 403 otherwise.

### INV-2: Server-side object ownership ✅

`app/files/routes.py:76-84` — detail loads through the tenant-scoped accessor and 404s on a miss, never revealing cross-tenant existence:
```python
record = File.get_for_tenant(file_id)
if record is None:
    abort(404)  # 404, not 403 (TMKB-AUTHZ-007)
```
On write, `organization_id` and `uploaded_by_id` are derived from the session (`routes.py:45-52`), never read from the request body.

### INV-3: List/detail consistency ✅

Both endpoints share the same `TenantScopedMixin` path:
- List (`routes.py:72`): `File.tenant_query().order_by(...).all()`
- Detail (`routes.py:79`): `File.get_for_tenant(file_id)`

`tenant_query()` and `get_for_tenant()` apply the same org filter **and** soft-delete exclusion (`app/models/base.py:74-111`), so divergence is architecturally impossible.

### INV-4: Background job re-authorization ✅ (the discriminator)

Task signature (`app/tasks.py:27`):
```python
def process_file(self, file_id, user_id, organization_id):
```
vs. Run-8 baseline's `def process_file(self, file_id):` with a bare `db.session.get(File, file_id)` and zero checks.

The enhanced task re-validates before touching the resource (`tasks.py:28-51`):
1. **Tenant-filtered load** — `File.query.filter_by(id=file_id, organization_id=organization_id).first()`, explicitly *not* `File.query.get()`.
2. **Tenant match** — defense-in-depth re-check of `organization_id` even after the filtered load.
3. **User still exists** — `db.session.get(User, user_id)` is not `None`.
4. **User still in org** — `user.organization_id == organization_id`.
5. **Not soft-deleted** — `file_record.is_deleted` short-circuits processing (TMKB-AUTHZ-003).

---

## Attribution: TMKB vs. superpowers

This run is the evidence that **TMKB, not superpowers, produces the security architecture.** With the superpowers framework OFF, TMKB alone still produced:

- **INV-4 fix** — full auth context into the task plus re-validation.
- **`TenantScopedMixin`** that *raises* outside a request context (`base.py:82-86`), forcing background jobs onto the explicit-tenant path — INV-4 becomes hard to regress, not just fixed once.
- **Boundary mental model and `TMKB-AUTHZ-*` references** throughout the code.
- **Non-invariant hardening:** per-org upload dirs + `secure_filename` + uuid prefix (traversal), user-enumeration-safe login, mass-assignment defense on `create_user`, and an ORM-level `before_update` guard blocking mutation of soft-deleted rows.

What did **not** appear this run: a **security test suite**. The Sonnet enhanced run shipped `tests/test_security.py`; this run shipped no tests at all — matching its superpowers-OFF condition (and matching the Run-8 baseline, which also had none). This strongly indicates the test suite in the earlier run was a **superpowers/TDD artifact, not a TMKB outcome.** See the correction note in `tmkb-enhanced-analysis.md`. The takeaway is a cleaner value statement: **TMKB buys architecture; it does not, on its own, buy tests.**

---

## Gaps and caveats (not a rubber stamp)

These do not change any invariant result but should be recorded:

1. **No test suite in the artifact.** Cross-tenant isolation is asserted nowhere. Expected for a superpowers-OFF run, but it means "comprehensive security tests" is *not* a claim this run supports.
2. **The task bypasses its own helper.** `tasks.py:29` uses raw `File.query.filter_by(id=, organization_id=)` instead of the purpose-built `File.get_for_tenant(file_id, tenant_id=organization_id)`. Functionally fine (it adds an explicit `is_deleted` check), but the centralized path it built is the more consistent choice, and the redundant tenant re-check at `tasks.py:40` is dead code given the filtered load.
3. **No uploader-match check in the task.** Unlike the Sonnet enhanced run (which had a 5th "uploaded_by" check), this task validates tenant + user-in-org but not that the file was uploaded by `user_id`. Acceptable for an org-owned resource, but worth noting the check count is 4, not 5.
4. **Traceability typo.** `base.py:51` references `MIT-AUTHZ-003b` instead of `TMKB-AUTHZ-003`, breaking the code→threat-model grep the analyses tout.
5. **Residual non-invariant gaps, same as baseline.** No login rate-limiting, no CSRF on cookie-auth mutating endpoints, and `SECRET_KEY` defaults to `dev-secret-change-me` (`config.py:7`). Arguably outside TMKB's authz-boundary scope, but they were not closed — do not overclaim "secure across the board."

---

## Bottom line

This is the highest-signal datapoint in the suite: **Opus 4.7, identical prompt, superpowers OFF on both sides, TMKB the only difference — INV-4 goes from 0 authorization checks to 4, and the run passes all four invariants.** Combined with Run-8, the suite can now state a same-model, confound-free A/B rather than a cross-model, superpowers-confounded one.

> A generational upgrade (4.6 → 4.7) and extended thinking both failed to surface the async-boundary pattern in the baseline. TMKB supplies the missing knowledge — and does so without help from a process framework.

---

## References

- **Paired baseline:** `validation/smoke-test/baseline/test-app-opus-4-7-analysis.md` (Run-8)
- **Sonnet enhanced (superpowers ON):** `validation/smoke-test/enhanced/tmkb-enhanced-analysis.md`
- **Invariants:** `validation/INVARIANTS.md`
- **Protocol & experimental conditions:** `validation/PROTOCOL.md`
- **Clean Sonnet re-run setup:** `validation/smoke-test/CLEAN-RERUN-SETUP.md`
</content>
</invoke>
