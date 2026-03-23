# Subagent Patterns

Common patterns for spawning and coordinating subagents across skills.

## Agent Tool Invocation

```
Agent(
  subagent_type: "<type>",
  description: "<3-5 word summary>",
  prompt: "<detailed task with context paths>"
)
```

Always include in subagent prompts:
- Project root path
- Reports path (from hook context or `plans/reports/`)
- Relevant file paths/ownership
- Specific deliverables expected

## Common Subagent Types

| Agent | Use When | Model |
|-------|----------|-------|
| `Explore` | Quick codebase search, file discovery | (inherited) |
| `researcher` | External research, docs, best practices | haiku |
| `planner` | Create implementation plans | opus |
| `fullstack-developer` | Implement code changes | opus |
| `tester` | Run tests, coverage analysis | sonnet |
| `code-reviewer` | Code quality review | sonnet |
| `debugger` | Root cause analysis, diagnostics | sonnet |
| `docs-manager` | Update project documentation | sonnet |
| `project-manager` | Plan sync-back, progress tracking | sonnet |
| `git-manager` | Commits, branches, PRs | haiku |

## Parallel Exploration

Spawn multiple `Explore` agents for independent scouting:

```
// Parallel — spawn in same message
Agent(subagent_type="Explore", description="Scout auth code", prompt="Find all auth-related files...")
Agent(subagent_type="Explore", description="Scout DB schema", prompt="Find database models...")
```

Use for: hypothesis verification, codebase mapping, multi-directory search.

## Parallel Implementation

Spawn `fullstack-developer` agents per phase with file ownership:

```
Agent(
  subagent_type="fullstack-developer",
  description="Implement phase 2",
  prompt: "Phase file: {path}. File ownership: src/api/*, src/models/*. ..."
)
```

Rules:
- Each agent owns distinct files (no overlap)
- Wait for all to complete before dependent phases
- Use `isolation: "worktree"` for git-isolated work if needed

## Research Delegation

Spawn max 2-3 `researcher` agents in parallel, each with focused scope:

```
Agent(subagent_type="researcher", description="Research auth patterns", prompt="Research [specific aspect]. Max 3 web searches. Write report to [reports-path].")
```

Rules:
- Max 3 web searches per researcher
- Each researcher has a distinct focus area
- Wait for all to complete before planning

## Finalization Pattern (MANDATORY)

Every orchestration skill must finalize with:

1. `project-manager` agent → sync plan status, update plan.md
2. `docs-manager` agent → update `./docs` if changes warrant
3. `TaskUpdate` → mark all Claude Tasks completed
4. Ask user about commit via `git-manager` agent

## Task-Agent Coordination

Assign tasks to agents via `owner`:
```
TaskUpdate(taskId=X, owner="agent-name")
```

Each agent:
1. Claims task: `TaskUpdate(status="in_progress")`
2. Does work
3. Completes: `TaskUpdate(status="completed")`
4. Blocked tasks auto-unblock when dependencies resolve
