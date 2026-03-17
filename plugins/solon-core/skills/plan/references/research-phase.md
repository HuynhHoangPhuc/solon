# Research Phase

**When to skip:** Fast mode, or if researcher reports already provided.

## Delegation Pattern

Spawn `researcher` agents via Agent tool in parallel. Max 3 agents per planning session.

Each agent investigates a distinct aspect — never overlap scope between agents.

### Agent Prompt Template

```
Research: {specific topic or technology}
Focus: {what to find — best practices / trade-offs / patterns}
Codebase: {project-root-path}
Max searches: 3 web searches
Write report to: {reports-path}/researcher-{date}-{slug}.md

Report format:
- Summary (2-3 sentences)
- Findings (by topic, cite sources)
- Trade-offs (pros/cons per approach)
- Recommendation (clear, justified)
- Unresolved Questions
```

### Aspect Split Examples

For a feature with API + DB + UI components:
- Agent 1: API design patterns, REST vs GraphQL, auth approaches
- Agent 2: Database schema design, indexing, migration strategies
- Agent 3: UI patterns, component library options, state management

For a pure backend task:
- Agent 1: Core implementation approach + libraries
- Agent 2: Security considerations + edge cases

## Token Efficiency Rules

- Max 3 web searches per agent — no open-ended browsing
- Read codebase docs (Glob/Grep) before web searches — codebase first
- Use `sl ast search` to find existing patterns before researching externally
- Focus: actionable insights, not exhaustive coverage
- Reports under 100 lines

## Synthesis

After all agents complete:
1. Collect report file paths
2. Read each report
3. Identify common findings and conflicts
4. Note which approach has most consensus
5. Pass combined findings to plan creation step

## Best Practices

- Research breadth before depth
- Document conflicting recommendations — note them as trade-offs
- Identify codebase constraints early (existing patterns to follow)
- Note security implications discovered during research
- Pass report paths to `sc plan scaffold` or planner for context
