# Workflow Modes

## Auto-Detection (Default: `--auto`)

Analyze task and pick mode:

| Signal | Mode | Rationale |
|--------|------|-----------|
| "quick", "simple", clear scope, <3 files | fast | Skip research overhead |
| Complex task, unfamiliar domain, new tech | hard | Research needed |
| 3+ independent features/layers/modules | parallel | Enable concurrent agents |
| "compare", "evaluate", ambiguous approach | two | Compare alternatives |

Use `AskUserQuestion` if detection is uncertain.

## Fast Mode (`--fast`)

No research. Scout → Plan → Hydrate.

1. Read codebase docs (codebase-summary.md, code-standards.md, system-architecture.md)
2. `sc plan scaffold --slug <slug> --mode fast`
3. Fill plan.md and phase files
4. `sc task hydrate <plan-dir>`
5. **Cook reminder:** `Ready to implement. Run: /solon:cook --auto {absolute-plan-path}/plan.md`

**Why `--auto` cook flag?** Fast planning pairs with fast execution — skip review gates.

## Hard Mode (`--hard`)

Research → Scout → Plan → Red Team → Validate → Hydrate.

1. Spawn max 2 `researcher` agents in parallel (different aspects, max 3 searches each)
2. Read codebase docs; scout if stale/missing
3. `sc plan scaffold --slug <slug> --mode hard`
4. Fill plan files using research findings
5. `sc plan red-team <plan-dir>` → evaluate with adversarial reviewers
6. `sc plan validate <plan-dir>` → interview user
7. `sc task hydrate <plan-dir>`
8. **Cook reminder:** `Ready to implement. Run: /solon:cook {absolute-plan-path}/plan.md`

## Parallel Mode (`--parallel`)

Same as Hard + file ownership + dependency matrix.

1. Steps 1-4 from Hard mode
2. Each phase gets **exclusive file ownership** (no file in two phases)
3. Plan includes dependency matrix (concurrent vs sequential)
4. `sc plan red-team <plan-dir>`
5. `sc plan validate <plan-dir>`
6. `sc task hydrate <plan-dir>` — parallel phases have no `addBlockedBy`
7. **Cook reminder:** `Ready to implement. Run: /solon:cook --parallel {absolute-plan-path}/plan.md`

### Parallel Phase Requirements
- Each phase self-contained, no runtime deps on sibling phases
- Clear file boundaries — each file modified in ONE phase only
- Group by: architectural layer, feature domain, or tech stack

## Two-Approach Mode (`--two`)

Research → 2 approaches → User selects → Red Team → Validate → Hydrate.

1. Spawn 2+ `researcher` agents for different angles
2. Design 2 complete approaches with clear trade-offs
3. `AskUserQuestion` — user selects approach
4. `sc plan scaffold --slug <slug> --mode two`
5. Fill plan files for selected approach only
6. `sc plan red-team <plan-dir>`
7. `sc plan validate <plan-dir>`
8. `sc task hydrate <plan-dir>`
9. **Cook reminder:** `Ready to implement. Run: /solon:cook {absolute-plan-path}/plan.md`

## Task Hydration Per Mode

| Mode | Task Granularity | Dependency Pattern |
|------|------------------|--------------------|
| fast | Phase-level only | Sequential chain |
| hard | Phase + critical steps | Sequential + step deps |
| parallel | Phase + steps + ownership | Parallel groups + sequential deps |
| two | After user selects approach | Sequential chain |

## Cook Reminder (MANDATORY)

Always output after plan creation with **actual absolute path**:

| Mode | Cook Command |
|------|-------------|
| fast | `/solon:cook --auto {path}/plan.md` |
| hard | `/solon:cook {path}/plan.md` |
| parallel | `/solon:cook --parallel {path}/plan.md` |
| two | `/solon:cook {path}/plan.md` |

> Run `/clear` before implementing — fresh context helps Claude focus on implementation.
> **Why absolute path?** After `/clear`, new session loses context.

This reminder is **NON-NEGOTIABLE** — always output after presenting the plan.

## Pre-Creation Check

Check `## Plan Context` in injected context:
- **"Plan: {path}"** → Ask "Continue with existing plan? [Y/n]"
- **"Suggested: {path}"** → Ask if activate or create new
- **"Plan: none"** → Create new using `Plan dir:` from `## Naming`
