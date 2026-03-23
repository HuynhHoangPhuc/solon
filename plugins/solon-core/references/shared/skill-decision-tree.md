# Skill Decision Tree

Route user intent to the correct solon-core skill.

## Primary Routing

| User Intent | Skill | Notes |
|-------------|-------|-------|
| "Build/implement X from scratch" | `/sl:plan` → `/sl:ship` | Plan first, then implement |
| "Fix bug/error/test failure" | `/sl:fix` | Unified fix for any issue type |
| "Why is X happening? Investigate" | `/sl:fix` | Investigates before fixing |
| "Evaluate approach, compare options" | `/sl:brainstorm` | Optionally chains to `/sl:plan` |
| "Quick technical question" | `/sl:ask` | Architecture consultation, no implementation |
| "Start new project" | `/sl:bootstrap` | End-to-end: research → plan → ship |
| "Run tests" | `/sl:test` | Delegates to tester agent |
| "Review code quality" | `/sl:review` | Delegates to code-reviewer agent |
| "Update/create docs" | `/sl:docs` | init / update / summarize |
| "Wrap up session" | `/sl:watzup` | Summarize recent changes |
| "Cut a release" | `/sl:release` | Version bump + changelog + tag |

## Internal Delegation Map

Skills that invoke other skills during their workflow:

```
brainstorm ──consensus──→ plan (if user agrees)
bootstrap  ──planning──→ plan ──implementation──→ ship
ship       ──quality───→ test ──quality──→ review
fix        ──complex───→ debug workflow (internal, not a separate skill)
fix        ──parallel──→ fullstack-developer agents
plan       ──validate──→ plan red-team / plan validate (subcommands)
```

## Disambiguation Guide

### plan vs brainstorm
- **plan**: You already know WHAT to build, need HOW (detailed phases, file ownership, tasks)
- **brainstorm**: You're unsure WHAT approach to take, need to explore options first

### fix vs plan → ship
- **fix**: Something is broken, need to diagnose and repair
- **plan → ship**: Building something new, not fixing a broken thing

### ask vs brainstorm
- **ask**: Quick question, want an answer now (no research delegation)
- **brainstorm**: Complex decision, want structured multi-option analysis with research

### bootstrap vs plan → ship
- **bootstrap**: Starting from zero (no project yet), need full scaffolding
- **plan → ship**: Project exists, adding a new feature or capability

### test vs fix
- **test**: Want to run the test suite and see results
- **fix**: Tests are failing and you want them diagnosed and fixed

### docs vs watzup
- **docs**: Create or update project documentation in `./docs`
- **watzup**: Quick summary of recent changes for session wrap-up
