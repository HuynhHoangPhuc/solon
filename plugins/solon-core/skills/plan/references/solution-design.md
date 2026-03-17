# Solution Design

## Core Principles

- **YAGNI** — Don't add functionality until necessary
- **KISS** — Prefer simple solutions over complex ones
- **DRY** — Avoid code and concept duplication

## Decision Criteria

When choosing between approaches, evaluate:

| Factor | Questions |
|--------|-----------|
| Complexity | How many moving parts? How much new tech? |
| Effort | Days to implement? Days to maintain? |
| Risk | What can go wrong? How recoverable? |
| Fit | Does it match existing codebase patterns? |
| Scope | Is this MVP or over-engineered? |

Always prefer the simpler option unless complexity buys clear value.

## Technical Trade-off Analysis

- Evaluate 2-3 approaches for each significant requirement
- Pros/cons per approach — be specific and concrete
- Short-term vs long-term implications
- Development effort vs operational benefit
- Recommend one option; explain why others are worse

## Architecture Patterns

Choose based on actual need, not hype:
- **Monolith first** — start simple; split only when pain is real
- **API contracts** — define interfaces before implementation
- **State management** — use simplest that works (local → lifted → global)
- **Data flow** — unidirectional is easier to debug

## Security Assessment (Required)

Identify during design, not after:
- Auth/authorization requirements
- Input validation surfaces
- Data exposure risks (PII, secrets)
- OWASP Top 10 applicability
- API security (rate limiting, CORS)

## Performance Considerations

Identify bottlenecks early:
- Database query patterns (N+1, missing indexes)
- Caching opportunities (what, where, TTL)
- Async processing candidates
- Memory and CPU profile

## Edge Cases & Failure Modes

Think through before writing phase files:
- What happens when external services fail?
- What happens on partial writes?
- Race conditions in concurrent operations
- Data consistency under failures
- Retry and rollback strategies

## Best Practices

- Document design decisions AND their rationale (not just what, but why)
- Consider both technical correctness and team capability
- Design with testing in mind — testable = better designed
- Plan for observability (logs, metrics, traces) from the start
