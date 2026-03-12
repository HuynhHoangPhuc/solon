# Skill Activation Matrix

When to activate each skill and tool during fixing workflows.

## Always Activate

| Skill/Tool | Reason |
|------------|--------|
| `solon:debug` | Core to all fix workflows - find root cause first |

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
| `solon:problem-solving` | Stuck on approach, multiple failed attempts |
| `solon:sequential-thinking` | Complex logic chain, multi-step reasoning needed |
| `solon:brainstorm` | Multiple valid approaches, architecture decision |
| `solon:context-engineering` | Fixing AI/LLM/agent code, context window issues |
| `solon:ai-multimodal` | UI issues, screenshots provided, visual bugs |

## Subagent Usage

| Subagent | Activate When |
|----------|---------------|
| `debugger` | Root cause unclear, need deep investigation |
| `Explore` (parallel) | Scout multiple areas simultaneously |
| `Bash` (parallel) | Verify implementation (typecheck, lint, build) |
| `researcher` | External docs needed, latest best practices |
| `planner` | Complex fix needs breakdown, multiple phases |
| `tester` | After implementation, verify fix works |
| `solon:code-review` | After fix, verify quality and security |
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
| Quick | `debug`, `solon:code-review`, parallel `Bash` verification |
| Standard | Above + Tasks, `solon:problem-solving`, `solon:sequential-thinking`, `tester`, parallel `Explore` |
| Deep | All above + `solon:brainstorm`, `solon:context-engineering`, `researcher`, `planner` |
| Parallel | Per-issue Task trees + `fullstack-developer` agents + coordination via `TaskList` |

## Detection Triggers

| Keyword/Pattern | Skill to Consider |
|-----------------|-------------------|
| "AI", "LLM", "agent", "context" | `solon:context-engineering` |
| "stuck", "tried everything" | `solon:problem-solving` |
| "complex", "multi-step" | `solon:sequential-thinking` |
| "which approach", "options" | `solon:brainstorm` |
| "latest docs", "best practice" | `researcher` subagent |
| Screenshot attached | `solon:ai-multimodal` |
