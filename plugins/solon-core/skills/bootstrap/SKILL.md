---
name: sl:bootstrap
description: "Bootstrap new projects with research, tech stack, design, planning, and implementation. Modes: deep (interactive), auto (default), fast (skip research), parallel (multi-agent)."
license: MIT
argument-hint: "[requirements] [--deep|--auto|--fast|--parallel]"
---

# Bootstrap - New Project Scaffolding

End-to-end project bootstrapping from idea to running code.

**See also:** `../../references/shared/skill-decision-tree.md` for when to use `/sl:bootstrap` vs other skills.

**Principles:** YAGNI, KISS, DRY | Token efficiency | Concise reports

## Usage

```
/bootstrap <user-requirements>
```

**Flags** (optional, default `--auto`):

| Flag | Mode | Thinking | User Gates | Planning Skill | Ship Skill |
|------|------|----------|------------|----------------|------------|
| `--deep` | Full interactive | Ultrathink | Every phase | `--deep` | (interactive) |
| `--auto` | Automatic | Ultrathink | Design only | `--auto` | `--auto` |
| `--fast` | Quick | Think hard | None | `--fast` | `--auto` |
| `--parallel` | Multi-agent | Ultrathink | Design only | `--parallel` | `--parallel` |

**Example:**
```
/bootstrap "Build a SaaS dashboard with auth" --fast
/bootstrap "E-commerce platform with Stripe" --parallel
```

## Workflow Overview

```
[Git Init] → [Research?] → [Tech Stack?] → [Design?] → [Planning] → [Implementation] → [Test] → [Review] → [Docs] → [Onboard] → [Final]
```

Each mode loads a specific workflow reference + shared phases.

## Mode Detection

If no flag provided, default to `--auto`.

Load the appropriate workflow reference:
- `--deep`: Load `references/workflow-full.md`
- `--auto`: Load `references/workflow-auto.md`
- `--fast`: Load `references/workflow-fast.md`
- `--parallel`: Load `references/workflow-parallel.md`

All modes share: Load `references/shared-phases.md` for implementation through final report.

## Step 0: Git Init (ALL modes)

Check if Git initialized. If not:
- `--deep`: Ask user if they want to init → `git-manager` subagent (`main` branch)
- Others: Auto-init via `git-manager` subagent (`main` branch)

## Skill Triggers (MANDATORY)

After early phases (research, tech stack, design), trigger downstream skills:

### Planning Phase
Activate **sl:plan** skill with mode-appropriate flag:
- `--deep` → `/sl:plan --deep <requirements>` (thorough research + validation)
- `--auto` → `/sl:plan --auto <requirements>` (auto-detect complexity)
- `--fast` → `/sl:plan --fast <requirements>` (skip research)
- `--parallel` → `/sl:plan --parallel <requirements>` (file ownership + dependency graph)

Planning skill outputs a plan path. Pass this to ship.

### Implementation Phase
Activate **sl:ship** skill with the plan path and mode-appropriate flag:
- `--deep` → `/sl:ship <plan-path>` (interactive review gates)
- `--auto` → `/sl:ship --auto <plan-path>` (skip review gates)
- `--fast` → `/sl:ship --auto <plan-path>` (skip review gates)
- `--parallel` → `/sl:ship --parallel <plan-path>` (multi-agent execution)

## Role

Elite software engineering expert specializing in system architecture and technical decisions. Brutally honest about feasibility and trade-offs.

## Critical Rules

- Activate relevant skills from catalog during the process
- Keep all research reports ≤150 lines
- All docs written to `./docs` directory
- Plans written to `./plans` directory using naming from `## Naming` section
- DO NOT implement code directly — delegate through planning + ship skills
- Sacrifice grammar for concision in reports
- List unresolved questions at end of reports

## References

- `references/workflow-full.md` - Full interactive workflow
- `references/workflow-auto.md` - Auto workflow (default)
- `references/workflow-fast.md` - Fast workflow
- `references/workflow-parallel.md` - Parallel workflow
- `references/shared-phases.md` - Common phases (implementation → final report)
- `../../references/shared/workflow-modes.md` - Standard mode vocabulary and detection
- `../../references/shared/subagent-patterns.md` - Agent spawning patterns
