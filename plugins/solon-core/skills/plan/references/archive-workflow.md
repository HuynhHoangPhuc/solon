# Archive Workflow

Archive completed or abandoned plans to keep `plans/` clean.

## Plan Resolution

1. If argument provided → use that path
2. Else → read all plans in `plans/` directory

## Workflow

### Step 1: Read Plan Files

For each plan to archive:
- Read `plan.md` — title, status, dates, phase count
- Read first 20 lines of each `phase-XX-*.md` — understand progress

### Step 2: Journal Entry (Optional)

Use `AskUserQuestion`: "Document journal entries before archiving?"
- Yes → spawn `docs-manager` agent to write journal entry per plan
- No → skip to Step 3

Journal entries go to `./docs/journals/`. Format:
```markdown
# {plan title} — {date}

**Status:** {completed|cancelled|abandoned}
**Effort:** {actual effort}
**Outcome:** {what was achieved}

## Key Decisions
- {decision}: {rationale}

## Lessons Learned
- {lesson}
```

### Step 3: Confirm Archive Action

Use `AskUserQuestion`:
- "Archive all plans shown above?"
- "Let me pick specific plans"
- "Only completed plans"

Then: "Move to `plans/archive/` or delete permanently?"

### Step 4: Execute Archive

**Move to archive:**
```bash
mkdir -p plans/archive
mv plans/{plan-dir} plans/archive/{plan-dir}
```

**Delete permanently:**
```bash
rm -rf plans/{plan-dir}
```

### Step 5: Commit (Optional)

Use `AskUserQuestion`:
- "Stage and commit changes"
- "Skip, I'll commit later"

If commit: use conventional commit format `chore: archive {N} completed plans`

## Output Summary

After archiving:
- Count: plans archived, plans deleted
- Table: title | status | created | effort
- Table: journal entries created (if any)

## Notes

- Only archive plans with status `completed` or `cancelled`
- Never archive `in-progress` plans without confirming with user
- Sacrifice grammar for concision
