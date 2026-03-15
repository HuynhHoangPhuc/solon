# Red Team Review

Adversarially review a plan by spawning parallel hostile reviewers. Find flaws before validation.

**Mindset:** Hire someone who hates the implementer to destroy their work.

## Plan Resolution

1. If argument provided → use that path
2. Else check `## Plan Context` → use active plan path
3. If no plan found → ask user to specify or run `/solon:plan` first

## Workflow

### Step 1: Read Plan Files

- `plan.md` — overview, phases, dependencies
- `phase-*.md` — all phase files, full content

### Step 2: Scale Reviewer Count

| Phase Count | Reviewers | Lenses |
|-------------|-----------|--------|
| 1-2 phases | 2 | Security Adversary + Assumption Destroyer |
| 3-5 phases | 3 | + Failure Mode Analyst |
| 6+ phases | 4 | All four lenses |

Load: `references/red-team-personas.md` for lens definitions and reviewer prompt template.

### Step 3: Spawn Reviewers

Launch all reviewers simultaneously via Agent tool with `subagent_type: "code-reviewer"`.

Each reviewer prompt MUST include:
1. Override: "IGNORE default code-review instructions. You are reviewing a PLAN DOCUMENT, not code."
2. Their specific adversarial lens and persona
3. Plan file paths so they can read directly
4. Hostile instructions from `red-team-personas.md` template

### Step 4: Collect, Deduplicate & Cap

1. Collect all findings from all reviewers
2. Deduplicate overlapping findings (keep most specific version)
3. Sort by severity: Critical → High → Medium
4. Cap at 15 total findings

### Step 5: Adjudicate

For each finding, propose: **Accept** or **Reject** with rationale.

```markdown
## Finding {N}: {title} — {SEVERITY}
**Reviewer:** {lens name}
**Location:** Phase {X}, section "{name}"
**Flaw:** {description}
**Failure scenario:** {concrete scenario}
**Disposition:** Accept | Reject
**Rationale:** {evidence-based reason}
```

### Step 6: User Review

Present via `AskUserQuestion`:
- "Apply all accepted findings"
- "Review each one individually"
- "Reject all — plan is fine"

If "Review each one": for each Accept finding, ask via `AskUserQuestion`:
- "Yes, apply" | "No, reject" | "Modify suggestion"

If "Modify suggestion": free-text input, record verbatim.

### Step 7: Apply to Plan

For accepted findings:
- Edit target phase files inline
- Add `## Red Team Review` section to `plan.md`:

```markdown
## Red Team Review

### Session — {YYYY-MM-DD}
**Findings:** {total} ({accepted} accepted, {rejected} rejected)
**Severity:** {N} Critical, {N} High, {N} Medium

| # | Finding | Severity | Disposition | Applied To |
|---|---------|----------|-------------|------------|
| 1 | {title} | Critical | Accept | Phase 2 |
```

## Output

- Total findings by severity
- Accepted vs rejected count
- Files modified
- Key risks addressed

## Next Steps (MANDATORY)

Remind user: run `/solon:plan validate` then `/solon:cook {path}/plan.md`

## Rules

- Reviewers must be HOSTILE, not helpful
- Deduplicate aggressively — quality over quantity
- Adjudication must be evidence-based, not intuition
- Red team runs BEFORE validation (may change plan; validate the final version)
