# Output Standards & Quality

## Frontmatter Schema (Required for plan.md)

```yaml
---
title: "{Brief plan title}"
description: "{One-sentence summary}"
status: pending       # pending | in-progress | completed | cancelled
priority: P2          # P1 (High) | P2 (Medium) | P3 (Low)
effort: 4h            # Estimated total effort
branch: feat/name     # Current git branch
tags: [backend, api]  # Category tags
created: 2026-03-15   # YYYY-MM-DD
---
```

### Tag Vocabulary

- **Type:** `feature`, `bugfix`, `refactor`, `docs`, `infra`
- **Domain:** `frontend`, `backend`, `database`, `api`, `auth`
- **Scope:** `critical`, `tech-debt`, `experimental`

## Writing Style

**Sacrifice grammar for concision:**
- Bullets and lists over prose
- Short sentences, remove filler words
- Prioritize actionable information
- Imperative mood for steps ("Add validation" not "You should add validation")

## Quality Checklist

Before finalizing a plan:

- [ ] plan.md has YAML frontmatter with all required fields
- [ ] Each phase file has: Overview, Requirements, Implementation Steps, Todo, Success Criteria
- [ ] Implementation steps are specific enough for a junior developer
- [ ] File paths in "Related Code Files" are absolute or clearly relative to project root
- [ ] Risk Assessment covers at least 2 risks per phase
- [ ] Dependencies between phases are explicit
- [ ] Todo checkboxes `[ ]` are present in every phase (required for hydration)
- [ ] No placeholder text ("TBD", "TODO: fill this in") in plan files

## Task Breakdown Rules

- Each task independently executable with clear dependencies
- Prioritize by: risk > dependencies > business value
- Eliminate ambiguity — no "implement the thing"
- Specific file paths for all modifications
- Clear acceptance criteria per phase

## What Planners Do (and Don't)

**Do:**
- Create plan files with complete context
- Provide pseudocode/snippets to clarify intent
- Offer multiple options with trade-offs
- Reference existing patterns in the codebase

**Don't:**
- Implement code
- Leave phases vague
- Skip risk assessment
- Create plans in user home directory

## Unresolved Questions

Use `AskUserQuestion` for genuine decision points before finalizing:
- Technical choices requiring business input
- Unknowns that change implementation significantly
- Trade-offs requiring stakeholder decisions

Revise plan files based on answers before hydrating tasks.
