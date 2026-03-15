---
name: brainstorm
description: "Brainstorm solutions with trade-off analysis, research delegation, and structured debate. Use for ideation, architecture decisions, technical debates, feature exploration, feasibility assessment."
argument-hint: "[topic or problem]"
---

# Brainstorm — Structured Solution Discovery

End-to-end brainstorming with research delegation and structured debate.

## Core Principles

Always honoring **YAGNI**, **KISS**, **DRY**. Be honest, brutal, and concise. You DO NOT implement — only brainstorm and advise.

## Workflow (7 Phases)

### Phase 1 — Scout

Understand the codebase before asking questions.

- Read `docs/` directory (codebase-summary.md, code-standards.md, system-architecture.md)
- Use `Glob` to map project structure
- Use `Grep` to find patterns related to the topic
- Use `sl ast search` to find semantic code patterns
- Use `sl lsp diagnostics` for existing errors
- Goal: understand constraints and existing context

### Phase 2 — Discovery

Clarify with the user before researching.

Use `AskUserQuestion` tool with these questions:
1. What problem are you solving? What's the desired outcome?
2. What constraints exist? (timeline, tech stack, team size, budget)
3. What approaches have you already considered or ruled out?
4. What does success look like?

Group max 4 questions per `AskUserQuestion` call.

### Phase 3 — Research

Delegate focused research to agents. Max 3 `researcher` agents in parallel via Agent tool.

Each agent: max 3 web searches + codebase analysis. Focus areas:
- Industry best practices for the domain
- Technology evaluation (pros/cons of candidate approaches)
- Existing patterns in this codebase relevant to the topic

Prompt each researcher: "Research [specific aspect]. Max 3 web searches. Write report to [reports-path]."

Wait for all agents to complete before proceeding.

### Phase 4 — Analysis

Evaluate 2-3 distinct approaches with YAGNI/KISS/DRY lens.

For each approach:
- Brief description (1-2 sentences)
- Pros: what it does well
- Cons: risks, complexity, trade-offs
- Effort estimate (rough)
- Fit with existing codebase

Present as comparison table:

| Criterion | Option A | Option B | Option C |
|-----------|----------|----------|----------|
| Complexity | Low | Medium | High |
| Effort | 1 day | 3 days | 1 week |
| Maintainability | High | Medium | Low |
| Risk | Low | Medium | High |

Provide recommendation with rationale. Challenge over-engineered options.

### Phase 5 — Debate

Present options to user and challenge assumptions.

Use `AskUserQuestion` to:
1. Present recommended approach — ask for agreement or pushback
2. Challenge their instinct if they pick the complex option
3. Surface hidden constraints ("have you considered X?")
4. Confirm final direction

Be direct. Say "Option B is over-engineered for your timeline" when true.

### Phase 6 — Documentation

Write brainstorm report to the reports directory.

Get reports path:
```bash
sc plan resolve
```
If no active plan: use `plans/reports/` in project root.

Report filename: `brainstorm-{YYYYMMDD}-{HHMM}-{slug}.md`

Report format:
```markdown
# Brainstorm: {topic}

**Date:** {date}
**Outcome:** {chosen direction}

## Context
{problem statement, constraints, codebase context}

## Options Evaluated

### Option A: {name}
- **Summary:** ...
- **Pros:** ...
- **Cons:** ...
- **Effort:** ...

### Option B: {name}
...

## Comparison Table
{table from Analysis phase}

## Recommendation
{chosen approach + rationale}

## Implementation Considerations
{risks, dependencies, gotchas}

## Open Questions
- {unresolved question 1}
- {unresolved question 2}
```

### Phase 7 — Finalize

Use `AskUserQuestion` to ask:

"Ready to create a detailed implementation plan?"
- Yes → Run `/solon:plan` with brainstorm summary as context. Pass report path as argument.
- No → End session, remind user to run `/solon:plan` later with the report.

If Yes: invoke plan skill with context note "Brainstorm report: {report-path}"

## Report Output

Use naming pattern from `## Naming` section in hook context (includes full path + computed date). Fall back to `plans/reports/brainstorm-{date}-{slug}.md`.

## Token Efficiency

- Keep researcher prompts focused — no open-ended "research everything"
- Read scout reports before spawning researchers (avoid duplicate work)
- Comparison table replaces verbose prose
- Report under 100 lines
