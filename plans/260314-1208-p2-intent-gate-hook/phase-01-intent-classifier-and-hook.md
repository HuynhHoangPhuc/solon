# Phase 1: Intent Classifier & Hook

## Context Links
- [Brainstorm report](../reports/brainstorm-260314-1111-token-quality-latency-comparison.md) — Feature #8
- [todo_continuation_enforcer.go](../../hooks/scripts/cmd/todo_continuation_enforcer.go) — UserPromptSubmit pattern reference
- [dev_rules.go](../../hooks/scripts/cmd/dev_rules.go) — Another UserPromptSubmit hook

## Overview
- **Priority:** P2
- **Status:** Completed
- **Effort:** ~2h, ~130 LOC

Classify user prompts into 7 intent categories via keyword matching, inject compact strategy line.

## Key Insights

- UserPromptSubmitInput has `Prompt` string field — contains raw user text
- Existing UserPromptSubmit hooks (dev-rules, usage-awareness, todo-enforcer) coexist fine — each writes context independently
- oh-my-openagent intent gate uses keyword matching, no LLM call — proven approach
- Category ordering matters: more specific intents (DEBUG, DEPLOY) should be checked before general ones (IMPLEMENT)

## Architecture

```
UserPromptSubmit event
  → intent_gate.go (cmd)
    → intent.Classify(prompt) → category string
    → intent.Strategy(category) → guidance string
    → hookio.WriteContext("[Intent: CATEGORY] guidance")
```

## Files to Create

1. **`hooks/scripts/internal/intent/classifier.go`** (~50 LOC)
   - `Classify(prompt string) string` — returns category name or ""
   - `Strategy(category string) string` — returns guidance text
   - Category priority order: DEBUG > TEST > DEPLOY > REFACTOR > EXPLAIN > RESEARCH > IMPLEMENT
   - Keywords checked via `strings.Contains` on lowercased prompt

2. **`hooks/scripts/cmd/intent_gate.go`** (~50 LOC)
   - Cobra command `intent-gate`
   - Read UserPromptSubmitInput
   - Call intent.Classify() → intent.Strategy()
   - Write context via hookio.WriteContext()
   - Fail-open: any error → os.Exit(0)

## Files to Modify

1. **`hooks/scripts/cmd/root.go`**
   - Add `rootCmd.AddCommand(intentGateCmd)`

2. **`hooks/hooks.json`**
   - Add `intent-gate` to UserPromptSubmit hooks array

3. **`hooks/scripts/internal/config/config.go`**
   - Add `"intent-gate": true` to DefaultConfig.Hooks

## Implementation Steps

### Step 1: Create classifier package

```go
// internal/intent/classifier.go
package intent

import "strings"

// Category priority order (most specific first)
var categories = []struct {
    Name     string
    Keywords []string
    Strategy string
}{
    {"DEBUG", []string{"fix", "bug", "error", "broken", "failing", "crash", "issue", "wrong", "not working"}, "Reproduce first. Check logs/traces. Isolate root cause before fixing."},
    {"TEST", []string{"test", "coverage", "spec", "assert", "verify", "validate"}, "Write focused tests. Cover edge cases. Don't mock internals."},
    {"DEPLOY", []string{"deploy", "push", "release", "publish", "ship", "ci/cd", "pipeline"}, "Verify tests pass. Check CI config. Confirm with user before pushing."},
    {"REFACTOR", []string{"refactor", "clean up", "simplify", "reorganize", "extract", "restructure"}, "Preserve behavior. Run tests before and after. Small incremental changes."},
    {"EXPLAIN", []string{"explain", "how does", "what is", "why does", "describe", "walk through", "tell me about"}, "Use /preview for visuals. Tailor depth to user's coding level."},
    {"RESEARCH", []string{"explore", "investigate", "analyze", "compare", "evaluate", "look into", "research"}, "Gather info before proposing changes. Read docs, search code, compare approaches."},
    {"IMPLEMENT", []string{"build", "create", "add", "write", "implement", "develop", "set up", "feature", "make"}, "Follow active plan. Write tests. Run compile after edits."},
}

// Classify returns the intent category for a prompt, or "" if no match.
func Classify(prompt string) string {
    lower := strings.ToLower(prompt)
    for _, cat := range categories {
        for _, kw := range cat.Keywords {
            if strings.Contains(lower, kw) {
                return cat.Name
            }
        }
    }
    return ""
}

// Strategy returns the guidance text for a category.
func Strategy(category string) string {
    for _, cat := range categories {
        if cat.Name == category {
            return cat.Strategy
        }
    }
    return ""
}
```

### Step 2: Create intent gate hook

```go
// cmd/intent_gate.go
package cmd

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "solon-hooks/internal/config"
    "solon-hooks/internal/hookio"
    "solon-hooks/internal/intent"
)

var intentGateCmd = &cobra.Command{
    Use:   "intent-gate",
    Short: "Classify user intent and inject strategy guidance",
    RunE:  runIntentGate,
}

func runIntentGate(cmd *cobra.Command, args []string) error {
    if !config.IsHookEnabled("intent-gate") {
        os.Exit(0)
    }

    defer func() {
        if r := recover(); r != nil {
            os.Exit(0)
        }
    }()

    var input hookio.UserPromptSubmitInput
    if err := hookio.ReadInput(&input); err != nil || input.Prompt == "" {
        os.Exit(0)
    }

    category := intent.Classify(input.Prompt)
    if category == "" {
        os.Exit(0)
    }

    strategy := intent.Strategy(category)
    hookio.WriteContext(fmt.Sprintf("[Intent: %s] %s\n", category, strategy))
    return nil
}
```

### Step 3: Register command
Add to `root.go` init(): `rootCmd.AddCommand(intentGateCmd)`

### Step 4: Add to hooks.json
Add to UserPromptSubmit hooks array:
```json
{ "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/hooks/scripts/bin/solon-hooks intent-gate", "timeout": 5 }
```

### Step 5: Add default config
Add `"intent-gate": true` to DefaultConfig.Hooks.

### Step 6: Write tests
- `internal/intent/classifier_test.go` — test each category matches expected keywords
- Test priority ordering (DEBUG wins over IMPLEMENT for "fix and add")
- Test empty/no-match returns ""
- `cmd/intent_gate_test.go` — test hook end-to-end

## Todo List
- [x] Create `internal/intent/classifier.go` with Classify() and Strategy()
- [x] Create `cmd/intent_gate.go` with Cobra command
- [x] Register command in `cmd/root.go`
- [x] Add to UserPromptSubmit in `hooks/hooks.json`
- [x] Add default config to `internal/config/config.go`
- [x] Write unit tests for classifier
- [x] Write unit tests for hook
- [x] Build and verify: `cd hooks/scripts && make build && make test`

## Success Criteria
- Prompts containing intent keywords get classified correctly
- More specific intents (DEBUG) win over general ones (IMPLEMENT)
- No-match prompts produce no output (silent skip)
- Injection is ≤30 tokens per prompt
- Build passes, all tests green
- <1ms classification time (keyword matching only)

## Risk Assessment
- **False positives:** "check" could match TEST when user means something else. Mitigated by ordering (TEST before IMPLEMENT) and using multi-word keywords where possible.
- **Ambiguous prompts:** "fix the test" matches both DEBUG and TEST. DEBUG wins by priority — acceptable since fixing is the primary action.
- **Short prompts:** Single-word prompts like "yes" or "ok" won't match — correctly skipped.
- **Token overhead:** ~30 tokens per prompt × ~50 prompts/session = ~1500 tokens. <1% of 200K context. Negligible.

## Security Considerations
- Reads only prompt text from stdin (already provided by Claude Code)
- No file reads, no network calls
- No sensitive data processing
- Output is static guidance text only
