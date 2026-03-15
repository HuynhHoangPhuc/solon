# Validation Question Framework

## Question Categories

| Category | Keywords to Detect |
|----------|--------------------|
| **Architecture** | "approach", "pattern", "design", "structure", "database", "API", "service" |
| **Assumptions** | "assume", "expect", "should", "will", "must", "default", "existing" |
| **Tradeoffs** | "tradeoff", "vs", "alternative", "option", "choice", "either/or", "or we could" |
| **Risks** | "risk", "might", "could fail", "dependency", "blocker", "concern", "unknown" |
| **Scope** | "phase", "MVP", "future", "out of scope", "nice to have", "later", "v2" |

## Question Format Rules

- Each question: 2-4 concrete options (not open-ended)
- Mark recommended option with "(Recommended)" suffix
- "Other" option is implicit — always available via free text
- Questions surface implicit decisions, not obvious facts
- One question per genuine decision point

## Example Questions by Category

### Architecture
```
Q: How should the new service communicate with existing components?
1. REST API calls (Recommended)
2. Message queue / events
3. Direct database sharing
4. gRPC
```

### Assumptions
```
Q: Phase 2 assumes the auth token is always valid when reaching the API layer.
   Is there a refresh/retry mechanism needed?
1. No — token validation happens at gateway, not here (Recommended)
2. Yes — add token refresh logic in this service
3. Defer to Phase 4 (auth hardening phase)
```

### Tradeoffs
```
Q: The plan uses two approaches for caching. Which should we standardize on?
1. Redis for all caching (Recommended)
2. In-memory cache (simpler, not distributed)
3. No caching in MVP — optimize later
```

### Risks
```
Q: Phase 3 depends on the external payments API being available during development.
   How should we handle unavailability?
1. Mock the API in tests, integrate late (Recommended)
2. Build against sandbox environment throughout
3. Accept the risk — it's usually available
```

### Scope
```
Q: The plan includes audit logging. Is this required for MVP?
1. Yes — compliance requirement
2. No — defer to post-MVP (Recommended)
3. Partial — log critical actions only
```

## Validation Log Format

```markdown
## Validation Log

### Session {N} — {YYYY-MM-DD}
**Trigger:** {post-red-team | user-requested | auto}
**Questions asked:** {count}

#### Questions & Answers

1. **[{Category}]** {full question text}
   - Options: {A} | {B} | {C}
   - **Answer:** {user's choice}
   - **Custom input:** {verbatim "Other" text if provided}
   - **Rationale:** {why this decision matters for implementation}

#### Confirmed Decisions
- {decision label}: {choice} — {brief why}

#### Action Items
- [ ] {specific change needed in plan/phase file}

#### Impact on Phases
- Phase {N}: {what needs updating and why}
```

## Recording Rules

- **Full question text** — exact question, not a summary
- **All options** — every option presented
- **Verbatim custom input** — record "Other" text exactly as typed
- **Rationale** — explain why the decision affects implementation
- **Session numbering** — increment from last session in log
- **Trigger** — state what prompted this validation run

## Phase Propagation Mapping

| Change Type | Target Section in Phase File |
|-------------|------------------------------|
| Requirements change | Requirements |
| Architecture decision | Architecture |
| Scope reduction/expansion | Overview + Implementation Steps |
| New risk identified | Risk Assessment |
| Unknown surfaced | Key Insights (new subsection) |
