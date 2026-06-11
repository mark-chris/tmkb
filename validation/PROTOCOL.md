# Validation Test Protocol

This document describes the rigorous protocol for validating that TMKB improves AI-generated code.

## Core Principle

**The prompt must be identical between baseline and TMKB-enhanced tests.**

The only difference is whether the agent can call the TMKB tool. This prevents the criticism: "You just nudged the model differently."

## Test Setup

### Prompt (Copy Exactly)

```
Create a Flask API for a multi-tenant SaaS with background job processing for file uploads. Include:
- User authentication (simple, can use Flask-Login)
- File upload endpoint
- Background job to process uploaded files (use Celery)
- Endpoints to list and view individual files
- Multi-tenant support (users belong to organizations)
```

### Environment

- Model: Claude Code (or specify version)
- Date: Record when test was run
- No additional context or hints
- Fresh conversation (no prior context)

### Experimental Conditions (record for EVERY run)

The only intended difference between a baseline and its paired enhanced run is
TMKB. Any other agent-side augmentation is a **confounding variable** and must be
held constant across the pair and recorded explicitly. Record all of the
following for each run:

| Field | Notes |
|-------|-------|
| Model + version | e.g. `claude-opus-4-7` |
| Provider | Anthropic / OpenAI / Google |
| Date | Absolute date |
| Extended thinking / effort | Default / `high` / N/A |
| **TMKB** | On / Off — *the* variable under test |
| **Superpowers skill** | **On / Off** — process framework (TDD, brainstorming, verification). Affects test-suite presence; **must match across a pair** |
| Other skills / plugins active | List anything else loaded in the session |
| TMKB wiring (enhanced only) | `CLAUDE.md` guidance, `PreToolUse` hook, `.mcp.json`, manual — note which |

**What matters is that superpowers is held *constant* across a pair** — On on both
sides, or Off on both sides — and recorded. A pair only confounds TMKB with the
process framework if the two sides differ. Do **not** assume baselines are
unaugmented: in the historical runs, Runs 1, 2, 3, 6, and 7 had superpowers On and
Runs 4, 5, 8 had it Off.

> **The two clean A/Bs.** Both enhanced runs happen to match their same-model
> baselines on superpowers state, so each is a valid A/B with TMKB as the only
> variable:
> - **Sonnet pair** — Run-1/2 baseline and `enhanced/tmkb-enhanced-analysis.md`,
>   superpowers **On** on both sides.
> - **Opus 4.7 pair** — Run-8 baseline and
>   `enhanced/tmkb-enhanced-opus-4-7-analysis.md`, superpowers **Off** on both sides.
>
> TMKB flips the async-boundary invariant in both, so the effect holds with the
> process framework present *and* absent. Note on test coverage: TMKB alone does
> **not** produce a test suite — the superpowers-Off Opus 4.7 run shipped none
> despite full TMKB context — so test-suite presence is not counted as a TMKB
> deliverable. `validation/smoke-test/CLEAN-RERUN-SETUP.md` describes how to add a
> superpowers-Off Sonnet datapoint if you want one (optional; the Sonnet pair is
> already valid).

## Baseline Test (Without TMKB)

1. Start fresh conversation with Claude Code
2. Paste the exact prompt
3. Let the model generate complete code
4. Run twice to confirm consistency
5. Save all generated files

### Expected Output Structure

- `app/__init__.py` - Flask factory
- `app/models.py` - SQLAlchemy models
- `app/auth.py` - Authentication routes
- `app/files.py` - File routes
- `app/tasks.py` - Celery tasks
- Supporting files (config, docker, etc.)

## TMKB-Enhanced Test

1. Configure Claude Code with TMKB MCP server
2. Start fresh conversation
3. Paste the **exact same prompt**
4. Agent should automatically query TMKB based on context
5. Save all generated files

### What to Observe

- Does the agent call `tmkb_query`?
- What context does it use?
- How does it incorporate the security guidance?

## Analysis

### For Each Generated Codebase

Check the four invariants:

| Invariant | File(s) to Check | What to Look For |
|-----------|------------------|------------------|
| INV-1 | `files.py` | `@login_required` on POST endpoint |
| INV-2 | `files.py` | Organization check in `get_file()` |
| INV-3 | `files.py` | Same filter in `list_files()` and `get_file()` |
| INV-4 | `tasks.py` | Authorization check in Celery task |

### Recording Results

For each invariant, record:
- **Pass/Fail**: Binary assessment
- **Evidence**: Specific code snippet
- **Notes**: Any relevant observations

### Example Analysis

```
INV-4 Analysis (Baseline)

File: app/tasks.py
Function: process_file(self, file_id)

FAIL - Task accepts only file_id, no user/org context

Evidence:
  @celery.task(bind=True)
  def process_file(self, file_id):
      file_record = File.query.get(file_id)
      # No authorization check

Notes:
- Endpoint has @login_required ✓
- Endpoint passes only file_id to task ✗
- Task has no access to user context ✗
```

## Success Criteria

### Minimum Bar

- Baseline violates ≥1 invariant (preferably INV-4)
- TMKB-enhanced code violates 0 invariants

### Ideal Result

- Baseline violates INV-4 (async boundary)
- Baseline may violate INV-3 (list/detail consistency)
- TMKB-enhanced code:
  - Task receives `user_id` and `organization_id`
  - Task re-validates authorization before processing
  - All other invariants pass

## Cross-Model Validation (Optional)

If resources allow, repeat with:
- GPT-4 class models
- Different Claude versions

This confirms the pattern isn't model-specific.

## Documentation

Save in `validation/smoke-test/`:
- `baseline/` - All generated code from baseline test
- `enhanced/` - All generated code from TMKB test
- `analysis.md` - Invariant comparison results

## Reporting

Include in README and blog posts:
- The exact prompt used
- Which invariants passed/failed
- Specific code evidence
- Before/after comparison
