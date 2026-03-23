# Skill Activation Matrix

When to activate each skill and tool during fixing workflows.

## Always Activate

| Skill/Tool | Reason |
|------------|--------|
| `sl:debug` | Core to all fix workflows - find root cause first |

## Task Orchestration (Moderate+ Only)

| Tool | Activate When |
|------|---------------|
| `TaskCreate` | After complexity assessment, create all phase tasks upfront |
| `TaskUpdate` | At start/completion of each phase |
| `TaskList` | Check available unblocked work, coordinate parallel agents |
| `TaskGet` | Retrieve full task details before starting work |

Skip Tasks for Quick workflow (< 3 steps). See `references/task-orchestration.md`.

## Conditional Activation

| Skill | Activate When |
|-------|---------------|
| `sl:problem-solving` | Stuck on approach, multiple failed attempts |
| `sl:sequential-thinking` | Complex logic chain, multi-step reasoning needed |
| `sl:brainstorm` | Multiple valid approaches, architecture decision |
| `sl:ai-multimodal` | UI issues, screenshots provided, visual bugs |

## Subagent Usage

| Subagent | Activate When |
|----------|---------------|
| `debugger` | Root cause unclear, need deep investigation |
| `Explore` (parallel) | Scout multiple areas simultaneously |
| `Bash` (parallel) | Verify implementation (typecheck, lint, build) |
| `researcher` | External docs needed, latest best practices |
| `planner` | Complex fix needs breakdown, multiple phases |
| `tester` | After implementation, verify fix works |
| `sl:review` | After fix, verify quality and security |
| `git-manager` | After approval, commit changes |
| `docs-manager` | API/behavior changes need doc updates |
| `project-manager` | Major fix impacts roadmap/plan status |
| `fullstack-developer` | Parallel independent issues (each gets own agent) |

## Parallel Patterns

See `references/parallel-exploration.md` for detailed patterns.

| When | Parallel Strategy |
|------|-------------------|
| Root cause unclear | 2-3 `Explore` agents on different areas |
| Multi-module fix | `Explore` each module in parallel |
| After implementation | `Bash` agents: typecheck + lint + build |
| Before commit | `Bash` agents: test + build + lint |
| 2+ independent issues | Task trees + `fullstack-developer` agents per issue |

## Workflow → Skills Map

| Workflow | Skills Activated |
|----------|------------------|
| Quick | `sl:debug`, `sl:review`, parallel `Bash` verification |
| Standard | Above + Tasks, `sl:problem-solving`, `sl:sequential-thinking`, `tester`, parallel `Explore` |
| Deep | All above + `sl:brainstorm`, `researcher`, `planner` |
| Parallel | Per-issue Task trees + `fullstack-developer` agents + coordination via `TaskList` |

## Detection Triggers

| Keyword/Pattern | Skill to Consider |
|-----------------|-------------------|
| "stuck", "tried everything" | `sl:problem-solving` |
| "complex", "multi-step" | `sl:sequential-thinking` |
| "which approach", "options" | `sl:brainstorm` |
| "latest docs", "best practice" | `researcher` subagent |
| Screenshot attached | `sl:ai-multimodal` |
