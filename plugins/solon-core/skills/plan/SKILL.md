---
name: sl:plan
description: "Plan implementations, design architectures, create technical roadmaps with detailed phases. Use for feature planning, system design, solution architecture, implementation strategy."
argument-hint: "[task description] [--fast|--hard|--parallel|--two]"
---

# Plan — Implementation Planning with Research + Validation

Create detailed implementation plans with configurable workflow modes.

## Usage

```
/sl:plan <task description> [--fast|--hard|--parallel|--two]
```

Default: `--auto` (analyze task complexity and pick mode automatically).

Add `--no-tasks` to skip task hydration in any mode.

## Default (No Arguments)

If invoked without arguments or unclear intent, use `AskUserQuestion`:

| Operation | Description |
|-----------|-------------|
| `(default)` | Create implementation plan for a task |
| `archive` | Write journal entry & archive plans |
| `red-team` | Adversarial plan review |
| `validate` | Critical questions interview |

## Mode Auto-Detection

When no flag specified, analyze task and pick mode:

| Signal | Mode |
|--------|------|
| "quick", "simple", clear scope, <3 files | fast |
| "parallel", 3+ independent features/layers | parallel |
| "compare", "evaluate", ambiguous approach | two |
| Complex, unfamiliar domain, new tech | hard |
| Default (uncertain) | hard |

Use `AskUserQuestion` if detection is uncertain.

Load: `references/workflow-modes.md` for auto-detection logic and per-mode workflows.

## Workflow Modes

| Flag | Research | Red Team | Validate | Ship Reminder |
|------|----------|----------|----------|---------------|
| `--fast` | Skip | Skip | Skip | `--auto` flag |
| `--hard` | 2 researchers | Yes | Optional | (none) |
| `--parallel` | 2 researchers | Yes | Optional | `--parallel` flag |
| `--two` | 2+ researchers | After selection | After selection | (none) |

### Fast Mode

No research. Scout → Plan → Hydrate.

1. Read `docs/` (codebase-summary.md, code-standards.md, system-architecture.md)
2. Run `sl plan scaffold --slug <slug> --mode fast` to create plan directory
3. Fill plan.md and phase files with content
4. Run `sl task hydrate <plan-dir>` to create tasks
5. Output ship reminder: `Ready to implement. Run: /sl:ship --auto {planDir}/plan.md`

### Hard Mode

Research → Scout → Plan → Red Team → Validate → Hydrate.

1. Spawn max 2 `researcher` agents in parallel (different aspects, max 3 searches each)
2. Read codebase docs; scout if stale/missing
3. Run `sl plan scaffold --slug <slug> --mode hard`
4. Fill plan.md and phase files using research findings
5. Run `sl plan red-team <plan-dir>` — evaluate prompt output with adversarial reviewers
6. Run `sl plan validate <plan-dir>` — interview user with critical questions
7. Run `sl task hydrate <plan-dir>`
8. Output ship reminder: `Ready to implement. Run: /sl:ship {planDir}/plan.md`

### Parallel Mode

Same as Hard + file ownership per phase + dependency matrix.

1. Same as Hard steps 1-4
2. Each phase in plan gets **exclusive file ownership** (no overlap between phases)
3. Plan includes dependency matrix (which phases run concurrently vs sequentially)
4. Run `sl plan red-team <plan-dir>`
5. Run `sl plan validate <plan-dir>`
6. Run `sl task hydrate <plan-dir>` — parallel groups have no `addBlockedBy`
7. Output ship reminder: `Ready to implement. Run: /sl:ship --parallel {planDir}/plan.md`

### Two-Approach Mode

Research → 2 approaches → User picks → Red Team → Validate → Hydrate.

1. Spawn 2+ `researcher` agents for different angles
2. Design 2 complete implementation approaches with trade-offs
3. Use `AskUserQuestion` to present approaches — user selects one
4. Run `sl plan scaffold --slug <slug> --mode two`
5. Fill plan files for selected approach only
6. Run `sl plan red-team <plan-dir>`
7. Run `sl plan validate <plan-dir>`
8. Run `sl task hydrate <plan-dir>`
9. Output ship reminder: `Ready to implement. Run: /sl:ship {planDir}/plan.md`

## Core Responsibilities

Always honoring **YAGNI**, **KISS**, **DRY**. Be honest and concise. DO NOT implement — only plan.

### Research & Analysis
Load: `references/research-phase.md`
**Skip if:** fast mode or provided with researcher reports

### Codebase Understanding
Load: `references/codebase-understanding.md`
**Skip if:** provided with scout reports

### Solution Design
Load: `references/solution-design.md`

### Plan Creation & Organization
Load: `references/plan-organization.md`

### Task Breakdown & Output Standards
Load: `references/output-standards.md`

## Workflow Process

1. **Pre-Creation Check** → Check Plan Context for active/suggested/none
2. **Mode Detection** → Auto-detect or use explicit flag
3. **Research Phase** → Spawn researchers (skip in fast mode)
4. **Codebase Analysis** → Read docs, scout if needed
5. **Plan Scaffolding** → `sl plan scaffold --slug <slug> --mode <mode>`
6. **Plan Documentation** → Fill content into scaffolded files
7. **Red Team Review** → `sl plan red-team <plan-dir>` (hard/parallel/two modes)
8. **Validation** → `sl plan validate <plan-dir>` (hard/parallel/two modes)
9. **Hydrate Tasks** → `sl task hydrate <plan-dir>` (unless `--no-tasks`)
10. **Ship Reminder** → Output command with absolute path (MANDATORY)

## Active Plan State

Check `## Plan Context` injected by hooks:
- **"Plan: {path}"** → Ask "Continue with existing plan? [Y/n]"
- **"Suggested: {path}"** → Ask if activate or create new
- **"Plan: none"** → Create new using `Plan dir:` from `## Naming`

After creating plan: run `sl workflow status <plan-dir>` to verify structure.

**IMPORTANT:** Always create plans in the CURRENT WORKING PROJECT DIRECTORY. Never in user home dir.

## Subcommands

| Subcommand | Reference | Purpose |
|------------|-----------|---------|
| `/sl:plan archive` | `references/archive-workflow.md` | Archive plans + write journal entries |
| `/sl:plan red-team` | `references/red-team-workflow.md` | Adversarial plan review |
| `/sl:plan validate` | `references/validate-workflow.md` | Validate plan with critical questions interview |

## Quality Standards

- Thorough and specific; consider long-term maintainability
- Detailed enough for junior developers
- Validate against existing codebase patterns
- Address security and performance concerns

**Remember:** Plan quality determines implementation success.
