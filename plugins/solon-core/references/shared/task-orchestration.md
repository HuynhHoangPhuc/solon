# Task Orchestration

Shared patterns for Claude Task creation, tracking, and coordination across skills.

## Session-Scoped Reality

Claude Tasks are **ephemeral** — they die when the session ends. Plan files (plan.md, phase-XX.md with checkboxes) are the **persistent** layer.

The **hydration pattern** bridges sessions:

```
Plan Files (persistent)   →[sl task hydrate]→   Claude Tasks (session)
[ ] Phase 1                                       pending
[ ] Phase 2                                       pending
                                ↓ work
Plan Files (updated)      ←[sync-back]←           Task Updates
[x] Phase 1                                       completed
[ ] Phase 2                                       in_progress
```

## When to Use Tasks

| Complexity | Use Tasks? | Reason |
|-----------|-----------|--------|
| Simple (< 3 steps) | No | Overhead exceeds benefit |
| Moderate (3-6 steps) | Yes | Multi-step coordination |
| Complex (6+ steps) | Yes | Dependency chains, parallel agents |
| Parallel (2+ issues) | Yes | Independent task trees |

## Task Tools

- `TaskCreate(subject, description, activeForm, metadata)` — create task
- `TaskUpdate(taskId, status, addBlockedBy, addBlocks)` — update status/deps
- `TaskGet(taskId)` — get full details
- `TaskList()` — list all with status

**Lifecycle:** `pending` → `in_progress` → `completed`

## TaskCreate Patterns

### Phase-Level Task

```
TaskCreate(
  subject: "Implement auth API endpoints",
  activeForm: "Implementing auth API endpoints",
  description: "Build login/logout/refresh endpoints. See phase-02-auth-api.md",
  metadata: {
    phase: 2, priority: "P1", effort: "3h",
    planDir: "plans/260315-auth/",
    phaseFile: "phase-02-auth-api.md"
  }
)
```

### Critical Step Task (high-risk steps within a phase)

```
TaskCreate(
  subject: "Implement token refresh with rotation",
  activeForm: "Implementing token refresh",
  description: "Handle expiry, refresh flow, error recovery. Step 2.4 in phase-02.",
  metadata: {
    phase: 2, step: "2.4", priority: "P1", effort: "1h",
    planDir: "plans/260315-auth/", phaseFile: "phase-02-auth-api.md",
    critical: true, riskLevel: "high"
  },
  addBlockedBy: ["{phase-1-task-id}"]
)
```

### Standard Workflow Tasks (fix/ship)

```
TaskCreate(subject="Debug & investigate",  metadata={step: 1})
TaskCreate(subject="Scout related code",   metadata={step: 2})
TaskCreate(subject="Implement fix",        metadata={step: 3}, addBlockedBy=[step1, step2])
TaskCreate(subject="Run tests",            metadata={step: 4}, addBlockedBy=[step3])
TaskCreate(subject="Code review",          metadata={step: 5}, addBlockedBy=[step4])
TaskCreate(subject="Finalize",             metadata={step: 6}, addBlockedBy=[step5])
```

## Dependency Chains

```
Phase 1 (no blockers)                    ← start here
Phase 2 (addBlockedBy: [P1-id])          ← auto-unblocked when P1 completes
Phase 3 (addBlockedBy: [P2-id])
Step 3.4 (addBlockedBy: [P2-id])         ← critical steps share phase dependency
```

Parallel phases: no `addBlockedBy` — they start simultaneously.

## Parallel Issue Coordination

For 2+ independent issues, create separate task trees:

```
// Issue A tree
TaskCreate(subject="[Issue A] Debug",   metadata={issue: "A", step: 1})
TaskCreate(subject="[Issue A] Fix",     metadata={issue: "A", step: 2}, addBlockedBy=[A-1])
TaskCreate(subject="[Issue A] Verify",  metadata={issue: "A", step: 3}, addBlockedBy=[A-2])

// Issue B tree (same pattern)

// Final shared task
TaskCreate(subject="Integration verify", addBlockedBy=[A-3, B-3])
```

## Subagent Task Assignment

Assign tasks via `owner` field:
```
TaskUpdate(taskId=taskA, owner="agent-debug")
TaskUpdate(taskId=taskB, owner="agent-fix")
```

Check available work: `TaskList()` → filter by `status=pending`, `blockedBy=[]`, `owner=null`

## Hydration & Cross-Session

**Hydrate:** `sl task hydrate <plan-dir>` — creates tasks from unchecked `[ ]` items.

| Mode | Task Granularity | Dependency Pattern |
|------|------------------|--------------------|
| fast | Phase-level only | Sequential chain |
| deep | Phase + critical steps | Sequential + step deps |
| parallel | Phase + steps + ownership | Parallel groups + sequential deps |

**Cross-session resume:**
1. Ship reads plan files → re-hydrates from unchecked `[ ]` items
2. Already-checked `[x]` items = done, skipped

## Rules

- Create tasks BEFORE starting work (upfront planning)
- Only 1 task `in_progress` per agent at a time
- Mark complete IMMEDIATELY after finishing (don't batch)
- Use `metadata` for filtering: `{step, phase, issue, severity}`
- If task fails → keep `in_progress`, create subtask for blocker
- Skip Tasks entirely for simple workflows (< 3 steps)

## Naming Conventions

- **subject** (imperative, <60 chars): "Setup database migrations", "Implement OAuth2 flow"
- **activeForm** (continuous): "Setting up database", "Implementing OAuth2"
- **description**: 1-2 sentences, concrete deliverables, reference phase file
