---
name: sl:cook
description: "Execute plan phases with progress tracking and quality gates. Use after /solon:plan to implement a full plan end-to-end."
argument-hint: "<plan-dir>/plan.md [--auto] [--parallel] [--no-test]"
---

# Cook — Implementation Orchestrator

Execute plan phases with progress tracking and quality gates.

## Usage

```
/solon:cook <plan-dir>/plan.md [--auto] [--parallel] [--no-test]
```

## Modes

| Flag | Behavior |
|------|----------|
| (default) | Sequential phases with review gate between each |
| `--auto` | Auto-approve gates when quality score ≥ 9.5 |
| `--parallel` | Concurrent phases where `blockedBy` allows |
| `--no-test` | Skip `/solon:test` after all phases complete |

## Workflow

### Step 1 — Resolve Plan

```bash
sc plan resolve
```

If argument provided, use that path directly. If no active plan found and no argument given, ask user via `AskUserQuestion`.

### Step 2 — Hydrate Tasks

```bash
sc task hydrate <plan-dir>
```

Parse phases into task list. Note `blockedBy` dependencies to determine execution order.

### Step 3 — Execute Phases

For each phase in dependency order:

1. Read phase file for implementation steps, file ownership, and success criteria
2. Create Claude Task via `TaskCreate` for tracking
3. Spawn `fullstack-developer` agent via Agent tool with:
   - Phase file path
   - Plan directory path
   - File ownership list from phase
   - Reports path
4. After agent completes, run compile/lint check:
   - Node: `npm run typecheck` or `npm run lint`
   - Rust: `cargo check`
   - Python: `python -m py_compile` or `ruff check`
5. Mark phase done:
   ```bash
   sc task sync <plan-dir> --completed <phase-num>
   ```
6. Update task: `TaskUpdate(status: "completed")`
7. **Review gate** (skip if `--auto` and quality ≥ 9.5):
   - Brief self-check: does output match phase success criteria?
   - If issues found, re-invoke agent to fix before proceeding

### Step 4 — Parallel Execution (if `--parallel`)

For phases with no `blockedBy` conflict, spawn multiple `fullstack-developer` agents simultaneously. Each agent owns distinct files (no overlap). Wait for all to complete before proceeding to dependent phases.

### Step 5 — Workflow Status

```bash
sc workflow status <plan-dir>
```

Print progress summary after all phases complete.

### Step 6 — Testing (unless `--no-test`)

Invoke `/solon:test --plan <plan-dir>`

Never skip failing tests. Fix failures before proceeding.

### Step 7 — Code Review

Invoke `/solon:review --plan <plan-dir>`

Address Critical-severity findings before finalizing.

### Step 8 — Finalize

1. Spawn `project-manager` agent to sync plan status
2. Spawn `docs-manager` agent to update `docs/` if implementation changed APIs, architecture, or public interfaces
3. Ask user via `AskUserQuestion`:
   - "Commit changes?" → Yes: conventional commit with focused message / No: leave staged

## Review Gate Criteria

After each phase, self-check:
- All files in ownership list were modified as expected
- Compile/lint passes
- Success criteria from phase file are met
- No files outside ownership list were modified

If any criterion fails: re-invoke agent with specific fix instructions before marking complete.

## Error Handling

- Compile error after phase → fix before marking complete, do not proceed
- Agent fails to complete → report to user, ask to retry or skip
- `sc` binary not found → instruct user to install Solon CLI

## Token Efficiency

- Read only the active phase file, not all phases at once
- Pass focused context to each agent (not entire plan)
- Review gate is a quick self-check, not a full re-read
