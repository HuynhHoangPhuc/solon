# Task Management Integration

## Session-Scoped Reality

Claude Tasks are **ephemeral** — they die when the session ends. Plan files (plan.md, phase-XX.md with checkboxes) are the **persistent** layer.

The **hydration pattern** bridges sessions:

```
Plan Files (persistent)   →[sc task hydrate]→   Claude Tasks (session)
[ ] Phase 1                                       ◼ pending
[ ] Phase 2                                       ◼ pending
                                ↓ work
Plan Files (updated)      ←[sync-back]←           Task Updates
[x] Phase 1                                       ✓ completed
[ ] Phase 2                                       ◼ in_progress
```

## When to Create Tasks

**Default:** On — auto-hydrate after plan files written.
**Skip with:** `--no-tasks` flag.
**3-Task Rule:** <3 phases → skip tasks (overhead exceeds benefit).

```bash
sc task hydrate <plan-dir>
```

## Task Creation Patterns

### Phase-Level Task

```
TaskCreate(
  subject: "Implement auth API endpoints",
  activeForm: "Implementing auth API endpoints",
  description: "Build login/logout/refresh endpoints. See phase-02-auth-api.md",
  metadata: {
    phase: 2,
    priority: "P1",
    effort: "3h",
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

## Naming Conventions

- **subject** (imperative, <60 chars): "Setup database migrations", "Implement OAuth2 flow"
- **activeForm** (continuous): "Setting up database", "Implementing OAuth2"
- **description**: 1-2 sentences, concrete deliverables, reference phase file

## Dependency Chains

```
Phase 1 (no blockers)                    ← start here
Phase 2 (addBlockedBy: [P1-id])          ← auto-unblocked when P1 completes
Phase 3 (addBlockedBy: [P2-id])
Step 3.4 (addBlockedBy: [P2-id])         ← critical steps share phase dependency
```

Parallel phases: no `addBlockedBy` — they start simultaneously.

## Ship Handoff

### Same Session (planning → ship immediately)
1. Hydrate tasks → tasks exist in session
2. Ship runs `TaskList` → finds existing tasks → starts implementation

### Cross Session (new session, resume plan)
1. User runs `/sl:ship {plan-dir}/plan.md`
2. Ship runs `TaskList` → empty (tasks died with session)
3. Ship reads plan files → re-hydrates from unchecked `[ ]` items
4. Already-checked `[x]` items = done, skipped

## Quality Checks After Hydration

Verify:
- No dependency cycles
- All phases have corresponding tasks
- Required metadata: phase, priority, effort, planDir, phaseFile
- Task count matches unchecked `[ ]` items

Output: `Hydrated [N] phase tasks + [M] critical step tasks`
