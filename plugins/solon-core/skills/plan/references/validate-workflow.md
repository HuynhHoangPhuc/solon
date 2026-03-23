# Validate Workflow

Interview the user with critical questions to validate assumptions and surface issues before coding begins.

## Plan Resolution

1. If argument provided → use that path
2. Else check `## Plan Context` → use active plan path
3. If no plan found → ask user to specify or run `/sl:plan --deep` first

## Configuration

Check `## Plan Context` for validation settings from `.sl.json`:
- `mode` — `prompt` | `auto` | `off`
- `questions` — range like `3-8` (min-max)

| Mode | Behavior |
|------|----------|
| `prompt` | Ask user "Validate this plan?" → Yes (Recommended) / No |
| `auto` | Run validation automatically without asking |
| `off` | Skip validation entirely |

## Workflow

### Step 1: Read Plan Files

- `plan.md` — overview, phases, dependencies
- `phase-*.md` — all phase files
- Look for: decision points, assumptions, risks, trade-offs, scope boundaries

### Step 2: Extract Question Topics

Load: `references/validate-question-framework.md` for category keywords and question format rules.

Scan plan files for keywords that indicate unvalidated assumptions or open decisions.

### Step 3: Generate Questions

For each detected topic:
- Formulate a concrete question with 2-4 options
- Mark recommended option with "(Recommended)" suffix
- "Other" option is implicit — always available

Use question count from `## Plan Context` validation settings (e.g., `3-8` → aim for 5).

### Step 4: Interview User

Use `AskUserQuestion` tool:
- Group related questions (max 4 per call)
- Focus on: assumptions, risks, trade-offs, architecture choices
- Only ask about genuine decision points — not obvious facts

### Step 5: Document Answers

Add or append `## Validation Log` section in `plan.md`:

```markdown
## Validation Log

### Session 1 — {YYYY-MM-DD}
**Trigger:** {what prompted this validation}
**Questions asked:** {count}

#### Questions & Answers

1. **[Architecture]** How should validation results be persisted?
   - Options: Save to plan.md frontmatter | Create validation-answers.md | Don't persist
   - **Answer:** Save to plan.md frontmatter (Recommended)
   - **Rationale:** Single source of truth, no extra files

#### Confirmed Decisions
- Persistence: plan.md frontmatter — simplest approach

#### Action Items
- [ ] Update phase-02 to write frontmatter after validation

#### Impact on Phases
- Phase 2: add frontmatter write step
```

### Step 6: Propagate Changes

Auto-update affected phase files based on validation answers.
Add marker: `<!-- Updated: Validation Session {N} - {change} -->`

## Output

- Number of questions asked
- Key decisions confirmed
- Phase files updated
- Recommendation: proceed or revise further

## Next Steps (MANDATORY)

After validation, output with absolute path:

```
Best Practice: Run /clear before implementing — fresh context helps focus.
Then run: /sl:ship --auto {ABSOLUTE_PATH}/plan.md

Why --auto? Plan was validated — safe to skip review gates.
Why absolute path? After /clear, new session loses context.
```

## Rules

- Only ask about genuine decision points — not rhetorical questions
- If plan is simple, fewer than min questions is fine
- Prioritize questions that could significantly change implementation
- Never ask the same question twice across sessions (check existing Validation Log)
