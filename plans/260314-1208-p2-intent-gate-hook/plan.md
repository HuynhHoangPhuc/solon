---
title: "P2 Intent Gate Hook — User Intent Classification & Strategy Injection"
description: "UserPromptSubmit hook classifying user intent into 7 categories and injecting compact strategy guidance"
status: completed
priority: P2
effort: 2h
branch: main
tags: [hooks, go, agent-quality, intent-classification]
created: 2026-03-14
depends_on: plans/260314-1118-p0-hooks-implementation
---

# P2 Intent Gate Hook

## Summary

UserPromptSubmit hook that classifies user prompts into 7 intent categories via keyword matching, injecting compact strategy guidance (~30 tokens). Fires every prompt, no cooldown. ~130 LOC new Go code.

**Source:** [Brainstorm report](../reports/brainstorm-260314-1111-token-quality-latency-comparison.md) — Feature #8
**Design:** [Brainstorm session](../reports/brainstorm-260314-1111-token-quality-latency-comparison.md)

## Phases

| # | Phase | Status | Effort | Files |
|---|-------|--------|--------|-------|
| 1 | [Intent Classifier + Hook](phase-01-intent-classifier-and-hook.md) | Completed | 2h | 2 new, 3 modified |

## Architecture

```
UserPromptSubmit → intent_gate.go
  ├─ Read prompt from UserPromptSubmitInput.Prompt
  ├─ Classify via intent.Classify(prompt) → IntentCategory
  ├─ Look up strategy guidance for category
  └─ Inject: "[Intent: CATEGORY] guidance text"
```

## Intent Categories

| # | Intent | Keywords | Guidance |
|---|--------|----------|----------|
| 1 | RESEARCH | explore, investigate, analyze, find, search, compare, evaluate, look into | Gather info before proposing changes. Read docs, search code, compare approaches. |
| 2 | IMPLEMENT | build, create, add, write, implement, develop, set up, feature | Follow active plan. Write tests. Run compile after edits. |
| 3 | DEBUG | fix, bug, error, broken, failing, crash, issue, wrong, not working | Reproduce first. Check logs/traces. Isolate root cause before fixing. |
| 4 | REFACTOR | refactor, clean, simplify, reorganize, extract, rename, move, restructure | Preserve behavior. Run tests before and after. Small incremental changes. |
| 5 | EXPLAIN | explain, how does, what is, why, describe, walk through, tell me about | Use /preview for visuals. Tailor to user's coding level. |
| 6 | TEST | test, coverage, spec, assert, verify, validate, check | Write focused tests. Cover edge cases. Don't mock internals. |
| 7 | DEPLOY | deploy, push, release, publish, ship, ci, cd, pipeline | Verify tests pass. Check CI config. Confirm with user before pushing. |

**Fallback:** If no category matches, skip injection (don't inject generic advice).

## Key Design Decisions

1. **Keyword matching only** — no LLM call, no regex complexity. Simple `strings.Contains` on lowercased prompt. Fast (<1ms).
2. **First match wins** — categories ordered by specificity (DEBUG before IMPLEMENT, since "fix" is more specific than "add"). If multiple match, most specific wins.
3. **No cooldown** — intent changes per message. Injection is small (~30 tokens). Acceptable overhead.
4. **Compact output** — single `[Intent: X]` line. Not a strategy block. Keeps context lean.
5. **Classifier as separate package** — `internal/intent/classifier.go` for testability. Hook in `cmd/intent_gate.go`.
6. **Config toggle** — `"intent-gate": true` in `.sl.json` hooks map. Can be disabled per-project.

## Config Schema Addition

```json
{
  "hooks": {
    "intent-gate": true
  }
}
```
