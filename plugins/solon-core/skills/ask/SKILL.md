---
name: sl:ask
description: "Answer technical and architectural questions with expert consultation. Use for quick 'how do I X' questions."
argument-hint: "[technical-question]"
---

# Ask — Quick Expert Consultation

Technical Q&A backed by project context, docs, and web search. Lightweight alternative to `/sl:brainstorm`.

## Usage

```
/sl:ask <technical-question>
```

Examples:
```
/sl:ask how do I add middleware in this project?
/sl:ask what's the best way to handle file uploads in Rust?
/sl:ask should I use channels or shared state here?
```

## Core Principle

**Do NOT implement.** Advisory only. Be honest, brutal, concise.

## Workflow

### Step 1 — Understand Context

Read project docs for context:
- `docs/system-architecture.md`
- `docs/code-standards.md`
- `docs/codebase-summary.md`

### Step 2 — Classify Question

| Type | Action |
|------|--------|
| Project-specific ("how does X work here?") | Answer from codebase context |
| General technical ("best way to do X?") | WebSearch + `/sl:docs-seeker` for evidence |
| Complex/architectural ("should we migrate to X?") | Suggest redirecting to `/sl:brainstorm` |

### Step 3 — Research (if needed)

For questions needing external info:
1. `WebSearch` for current best practices (max 2 searches)
2. `/sl:docs-seeker` for library-specific docs (if applicable)
3. Check codebase patterns via `Grep` / `sl ast search`

### Step 4 — Answer

Format:

```markdown
## Answer

{Direct answer — lead with the recommendation}

### Code Example
{Minimal working example if relevant}

### Why
{Brief rationale — YAGNI/KISS/DRY lens}

### Caveats
{Edge cases, gotchas, alternatives to consider}
```

If the question is too complex for a quick answer:
> This is complex enough to warrant a full brainstorm. Run `/sl:brainstorm <topic>` for trade-off analysis and structured debate.

## Constraints

- Do NOT implement anything — advisory only
- Include code snippets when they clarify the answer
- Max 2 web searches per question
- YAGNI/KISS/DRY principles in all recommendations
- If question maps to existing project docs, point there first
