# Clean Sonnet Re-run Setup (superpowers OFF, both sides)

**Goal:** produce a confound-free Sonnet 4.5 A/B to match the clean Opus 4.7 pair
(Run-8 baseline vs. Opus 4.7 enhanced). The existing Sonnet enhanced run had the
**superpowers** skill ON and its baseline did not, so that pair confounds TMKB
with a process framework. To fix it, **re-run *both* sides with superpowers OFF**
— swapping only the baseline is not enough, because it would still be compared
against a superpowers-ON enhanced run.

> Re-running only a new baseline does **not** decontaminate the pair. The
> contaminated side is the *enhanced* run (superpowers ON). Both halves must share
> the same superpowers state. We standardize on **OFF**.

---

## Invariants held constant across the pair

| Variable | Value (both runs) |
|----------|-------------------|
| Model | Claude Sonnet 4.5 (`claude-sonnet-4-5`) |
| Prompt | Verbatim from `validation/PROTOCOL.md` (do not paraphrase) |
| Extended thinking / effort | Default |
| Superpowers skill | **OFF** |
| Other skills/plugins | None loaded |
| Conversation | Fresh, no prior context |

**The only difference between the two runs is TMKB (off for baseline, on for enhanced).**

---

## 0. Turn superpowers OFF (applies to both runs)

Superpowers is a plugin loaded at session start (it announces itself via a
`SessionStart` hook / `using-superpowers` skill). For these runs it must not be
active. Pick whichever applies to your setup:

- **Disable the plugin** for the session/project so no `superpowers:*` skills are
  available, **or**
- Run in a **clean profile/sandbox** where the superpowers plugin is not installed.

**Verify before generating:** the session has **no** `superpowers:*` skills listed
and **no** "You have superpowers" / `using-superpowers` banner. Record the verified
state in the run's analysis doc (`Superpowers skill: OFF`).

---

## 1. Baseline run (TMKB OFF)

1. Fresh directory, e.g. `validation/smoke-test/baseline/run-9-sonnet-clean/`.
2. **No** `.mcp.json`, **no** `tmkb` server, **no** `PreToolUse` hook, and **no**
   TMKB guidance in `CLAUDE.md` (omit `CLAUDE.md` entirely).
3. Confirm `mcp__tmkb__tmkb_query` is **not** available in the session.
4. Paste the PROTOCOL.md prompt verbatim into a fresh conversation.
5. Let the model generate the full codebase. Save everything (tarball it).
6. Analyze against the four invariants; expect INV-4 to FAIL (consistent with Runs 1–8).

---

## 2. Enhanced run (TMKB ON) — mirror the Opus 4.7 enhanced wiring

Use the **same TMKB wiring** the Opus 4.7 enhanced run used, so the only model
delta is Sonnet-vs-Opus and the TMKB setup is identical.

1. Fresh directory, e.g. `validation/smoke-test/enhanced/flask-file-upload-sonnet-clean/`.
2. Start the TMKB server:
   ```bash
   TMKB_PATTERNS=/home/mark/Projects/tmkb/patterns /home/mark/Projects/tmkb/tmkb serve
   ```
3. `.claude/settings.local.json` — wire the tool + the auto-query hook (mirror of
   the Opus 4.7 run):
   ```json
   {
     "permissions": { "allow": ["mcp__tmkb__tmkb_query"] },
     "hooks": {
       "PreToolUse": [
         {
           "matcher": "Write|Edit|MultiEdit|NotebookEdit",
           "hooks": [
             {
               "type": "mcp_tool",
               "server": "tmkb",
               "tool": "tmkb_query",
               "input": {
                 "context": "About to write or edit code (${tool_input.file_path}) in a multi-tenant Flask file-upload service. Surface authorization and trust-boundary threats before generating code: background-job auth context loss, tenant isolation, IDOR, soft-delete resurrection, ownership confusion.",
                 "language": "python",
                 "framework": "flask",
                 "verbosity": "agent"
               },
               "timeout": 15,
               "statusMessage": "Consulting TMKB for authorization threats…"
             }
           ]
         }
       ]
     }
   }
   ```
4. `CLAUDE.md` — include the same TMKB design guidance used in the Opus 4.7 run
   (consult `mcp__tmkb__tmkb_query` during design; authorization failures occur at
   boundaries, not functions).
5. Paste the **same** PROTOCOL.md prompt verbatim into a fresh conversation.
6. Confirm the agent calls `tmkb_query` (and the hook fires). Save everything.
7. Analyze against the four invariants; expect all 4 to PASS.

---

## 3. Record and pair

- Fill in the **Experimental Conditions** table (PROTOCOL.md) for both runs,
  including `Superpowers skill: OFF`.
- Write a paired analysis like `tmkb-enhanced-opus-4-7-analysis.md`, framed as a
  same-model A/B.
- Expectation: a **second** clean A/B (Sonnet) confirming TMKB flips INV-4 with no
  superpowers and no model confound — giving a two-model generalization claim
  (Sonnet **and** Opus, both unaugmented).
- **Do not** credit any generated test suite to TMKB. With superpowers OFF, expect
  no security test suite; if one appears, note it but keep the test-coverage metric
  out of the TMKB delta.
</content>
