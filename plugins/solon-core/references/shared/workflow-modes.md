# Workflow Modes

Standard mode vocabulary and auto-detection logic shared across orchestration skills.

## Standard Flags

| Flag | Meaning | Used By |
|------|---------|---------|
| `--fast` | Skip research, minimal workflow | plan, bootstrap, fix |
| (no flag / `--auto`) | Auto-detect complexity, standard workflow | all |
| `--deep` | Full research, validation, red-team | plan, bootstrap, fix |
| `--parallel` | Multi-agent parallel execution | plan, bootstrap, fix, ship |
| `--two` | Compare two approaches | plan only |

## Auto-Detection Signals

When no flag specified, analyze task and pick mode:

| Signal | Mode | Rationale |
|--------|------|-----------|
| "quick", "simple", clear scope, <3 files | fast | Skip research overhead |
| Complex task, unfamiliar domain, new tech | deep | Research needed |
| 3+ independent features/layers/modules | parallel | Enable concurrent agents |
| "compare", "evaluate", ambiguous approach | two | Compare alternatives |

Use `AskUserQuestion` if detection is uncertain.

## Mode Selection (Fix Skill)

For fix workflows, prompt user with `AskUserQuestion`:

| Option | Recommend When | Behavior |
|--------|----------------|----------|
| **Autonomous** (default) | Simple/moderate issues | Auto-approve if score >= 9.5 & 0 critical |
| **Human-in-the-loop** | Critical/production code | Pause for approval at each step |
| **Quick** | Type errors, lint, trivial bugs | Fast debug-fix-review cycle |

### Skip Mode Selection When

- Issue is clearly trivial (type error keyword detected) → default Quick
- User explicitly specified mode in prompt
- Previous context already established mode

## Mode Mapping Per Skill

### Plan Modes

| Flag | Research | Red Team | Validate | Ship Reminder |
|------|----------|----------|----------|---------------|
| `--fast` | Skip | Skip | Skip | `--auto` flag |
| `--deep` | 2 researchers | Yes | Optional | (none) |
| `--parallel` | 2 researchers | Yes | Optional | `--parallel` flag |
| `--two` | 2+ researchers | After selection | After selection | (none) |

### Bootstrap Modes

| Flag | Thinking | User Gates | Planning Skill | Ship Skill |
|------|----------|------------|----------------|------------|
| `--deep` | Ultrathink | Every phase | `--deep` | (interactive) |
| `--auto` | Ultrathink | Design only | `--auto` | `--auto` |
| `--fast` | Think hard | None | `--fast` | `--auto` |
| `--parallel` | Ultrathink | Design only | `--parallel` | `--parallel` |

### Fix Modes

| Complexity | Workflow |
|------------|----------|
| Simple | `workflow-quick.md` |
| Moderate | `workflow-standard.md` |
| Complex | `workflow-deep.md` |
| Parallel | Parallel `fullstack-developer` agents |

## Pre-Creation Check (Plan/Ship)

Check `## Plan Context` injected by hooks:
- **"Plan: {path}"** → Ask "Continue with existing plan? [Y/n]"
- **"Suggested: {path}"** → Ask if activate or create new
- **"Plan: none"** → Create new using `Plan dir:` from `## Naming`
