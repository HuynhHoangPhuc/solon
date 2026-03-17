---
name: sl:review
description: "Delegate code review to code-reviewer agent with plan-aware context. Use after /solon:cook or standalone for code quality review."
argument-hint: "[--plan <plan-dir>]"
---

# Review — Code Review Orchestrator

Delegate code review to `code-reviewer` agent with plan-aware context.

## Usage

```
/solon:review [--plan <plan-dir>]
```

## Severity Levels

| Level | Label | Action Required |
|-------|-------|-----------------|
| 🔴 | Critical | Must fix before merge |
| 🟡 | Warning | Should fix, explain if skipping |
| 🔵 | Suggestion | Nice to have, no action required |

## Workflow

### Step 1 — Resolve Plan Context

If `--plan <plan-dir>` provided, use that path.
Otherwise run:
```bash
sc plan resolve
```

If active plan found, extract requirements and success criteria from phase files for review context.

### Step 2 — Collect Changed Files

```bash
git diff --name-only HEAD
git diff --name-only --cached
```

If no git diff (new project or initial commit), review all source files in scope.

### Step 3 — Delegate to Code-Reviewer Agent

Spawn `code-reviewer` agent via Agent tool with this context:

```
Project root: <path>
Changed files: <list from git diff>
Plan requirements: <extracted from plan phases, if plan active>
Reports path: <plan-reports-dir or plans/reports/>

Review focus areas:
1. Correctness — logic errors, edge cases, off-by-one, null handling
2. Security — injection, auth gaps, exposed secrets, unsafe deserialization
3. Performance — N+1 queries, blocking calls, unnecessary allocations
4. Maintainability — readability, naming, modularity, DRY violations
5. Architecture — adherence to project patterns in docs/system-architecture.md
6. Plan compliance — do changes match plan requirements (if plan active)?

Use severity levels: Critical / Warning / Suggestion
Write review report to reports path.
```

### Step 4 — Write Review Report

Report filename: `review-{YYYYMMDD}-{HHMM}-{slug}.md`

Report format:
```markdown
# Code Review Report: {project or feature}

**Date:** {date}
**Reviewer:** code-reviewer agent
**Files Reviewed:** {count}
**Plan:** {plan-dir or none}

## Summary

| Severity | Count |
|----------|-------|
| Critical | N |
| Warning | N |
| Suggestion | N |

## Findings

### 🔴 Critical: {title}
- **File:** {path}:{line}
- **Issue:** {description}
- **Fix:** {specific recommendation}

### 🟡 Warning: {title}
- **File:** {path}:{line}
- **Issue:** {description}
- **Fix:** {recommendation}

### 🔵 Suggestion: {title}
- **File:** {path}:{line}
- **Note:** {improvement idea}

## Plan Compliance
{match/mismatch with plan requirements, if plan active}

## Overall Assessment
{1-2 sentence verdict}
```

### Step 5 — Report Summary

Output to user:
- Count per severity level
- List of Critical findings (titles only)
- Path to full report

If Critical findings exist: surface them clearly. When invoked from `/solon:cook`, Critical findings block finalize step and re-invoke `fullstack-developer` agent to fix.

## Integration with Cook

When invoked by `/solon:cook`:
- Critical findings → fix loop: re-invoke `fullstack-developer` agent with findings, then re-review
- Warning findings → present to user, ask to fix or document reason for skipping
- Suggestion findings → logged in report, no blocking

## Report Output

Use naming pattern from `## Naming` section in hook context. Fall back to `plans/reports/review-{date}-{slug}.md`.
