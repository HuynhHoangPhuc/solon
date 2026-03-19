---
name: sl:fix
description: "ALWAYS activate this skill before fixing ANY bug, error, test failure, CI/CD issue, type error, lint error, code problem."
argument-hint: "[issue] [--auto|--quick]"
---

# Fix — Structured Bug Fix Workflow

Diagnose before fixing. Never jump to code changes without understanding root cause.

## Usage

```
/sl:fix <issue description>
/sl:fix <issue> --quick    # fast path for trivial issues
/sl:fix <issue> --auto     # auto-approve if score >= 9.5
```

No args: use `AskUserQuestion` to get issue description and mode.

## Modes

| Mode | When | Behavior |
|------|------|----------|
| (default) | Moderate issues | Debug → fix → verify with review gates |
| `--quick` | Type errors, lint, trivial bugs | Fast debug → fix → verify |
| `--auto` | Any complexity | Auto-approve if quality score >= 9.5 |

## Workflow

### Step 1 — Diagnose

Invoke `/sl:debug` with the issue description. Wait for diagnosis report.

**MANDATORY:** Never skip this step. Never attempt a fix without diagnosis.

For `--quick` mode: run abbreviated diagnosis:
1. `sl lsp diagnostics <file>` — get error locations
2. `Grep` for error patterns
3. Brief hypothesis (no full report needed)

### Step 2 — Assess Complexity

Based on diagnosis, classify:

| Complexity | Criteria | Approach |
|------------|----------|----------|
| Simple | Single file, clear fix, <10 lines changed | Direct edit |
| Moderate | 2-5 files, clear root cause | Spawn `fullstack-developer` agent |
| Complex | 5+ files, architectural implications | Plan first, then implement |

### Step 3 — Implement Fix

**Simple:** Apply fix directly using `sl edit` (hashline) for precise edits.

**Moderate/Complex:** Spawn `fullstack-developer` agent via Agent tool:

```
Project root: <path>
Diagnosis report: <path or inline summary>
Root cause: <from debug report>
Files to modify: <list from diagnosis>
Reports path: <reports-dir>

Instructions:
1. Read diagnosis report for root cause and recommended fix
2. Implement fix using `sl edit` for precise line edits
3. Keep changes minimal — fix the bug, don't refactor
4. Run compile/lint check after edits
```

### Step 4 — Verify

1. **Compile check** — run language-appropriate compile/lint command
2. **LSP diagnostics** — `sl lsp diagnostics <file>` for each changed file → zero new errors
3. **Run tests** — invoke `/sl:test` if test suite exists
4. **Regression check** — verify fix doesn't break existing behavior

All checks must pass. If any fail, loop back to Step 3.

### Step 5 — Finalize

1. Report: files changed, root cause, fix applied
2. Ask user about commit via `/sl:git cm`

## Solon Tool Integration

| Step | Tool | Purpose |
|------|------|---------|
| Diagnose | `sl lsp diagnostics` | Locate errors |
| Diagnose | `sl ast search` | Find patterns |
| Diagnose | `/sl:scout` | Context gathering |
| Fix | `sl edit` | Precise hashline edits |
| Verify | `sl lsp diagnostics` | Confirm zero new errors |
| Test | `/sl:test` | Run test suite |

## Constraints

- NEVER attempt a fix without diagnosis first
- Keep changes minimal — fix the bug, don't refactor surrounding code
- Verify with LSP diagnostics after every edit
- Failed tests block finalization — fix them first
