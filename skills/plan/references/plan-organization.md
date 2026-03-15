# Plan Creation & Organization

## Directory Structure

Use `Plan dir:` from `## Naming` section injected by hooks for the full computed path.

```
{plan-dir}/
├── research/
│   ├── researcher-01-report.md
│   └── ...
├── plan.md                          # Overview access point
├── phase-01-setup-environment.md
├── phase-02-implement-core.md
├── phase-03-implement-api.md
└── phase-04-write-tests.md
```

After `sc plan scaffold --slug <slug> --mode <mode>`, templates are created. Fill in content.

**ALWAYS** create plans in the CURRENT WORKING PROJECT DIRECTORY. Never in `~` or user home.

## plan.md Structure

All `plan.md` files MUST include YAML frontmatter:

```markdown
---
title: "Feature Implementation Plan"
description: "One-sentence summary for card preview"
status: pending
priority: P2
effort: 8h
branch: feat/feature-name
tags: [backend, api]
created: 2026-03-15
---

# Feature Implementation Plan

## Overview
Brief description of what this plan accomplishes.

## Phases

| # | Phase | Status | Effort | Link |
|---|-------|--------|--------|------|
| 1 | Setup | Pending | 1h | [phase-01](./phase-01-setup.md) |
| 2 | Core | Pending | 4h | [phase-02](./phase-02-core.md) |
| 3 | Tests | Pending | 2h | [phase-03](./phase-03-tests.md) |

## Dependencies
- List key external dependencies here
```

Keep plan.md under 80 lines. It's an index, not a spec.

## Phase File Structure

Each `phase-XX-name.md` contains:

```markdown
## Overview
- Priority: P1/P2/P3
- Status: Pending
- Brief description

## Key Insights
- Important findings from research
- Critical considerations

## Requirements
- Functional requirements
- Non-functional requirements

## Architecture
- System design, component interactions, data flow

## Related Code Files
- Files to modify (with action: modify/create/delete)
- Brief change description per file

## Implementation Steps
1. Step one (specific, actionable)
2. Step two
...

## Todo
- [ ] Task one
- [ ] Task two

## Success Criteria
- Definition of done
- Validation methods

## Risk Assessment
| Risk | Impact | Mitigation |
|------|--------|------------|

## Next Steps
- Dependencies unblocked after this phase
```

## Frontmatter Auto-Population

| Field | Source |
|-------|--------|
| title | Extract from task description |
| description | First sentence of Overview |
| status | Always `pending` for new plans |
| priority | From user request, default `P2` |
| effort | Sum of phase estimates |
| branch | `git branch --show-current` |
| tags | Infer from keywords (frontend, backend, api, auth) |
| created | Today's date YYYY-MM-DD |

## File Naming

Phase files: `phase-{NN}-{kebab-description}.md`
- `phase-01-setup-environment.md`
- `phase-02-implement-auth-api.md`
- `phase-03-add-ui-components.md`

Slug for plan dir: `{YYMMDD}-{HHMM}-{kebab-task-name}`
