// Package intent classifies user prompts into intent categories via keyword matching.
package intent

import "strings"

// category defines an intent with keyword triggers and guidance text.
type category struct {
	Name     string
	Keywords []string
	Strategy string
}

// categories ordered by priority: most specific first.
// DEBUG and TEST win over IMPLEMENT for ambiguous prompts like "fix and add".
var categories = []category{
	{
		"DEBUG",
		[]string{"fix", "bug", "error", "broken", "failing", "crash", "issue", "wrong", "not working"},
		"Reproduce first. Check logs/traces. Isolate root cause before fixing.",
	},
	{
		"TEST",
		[]string{"test", "coverage", "spec", "assert", "verify", "validate"},
		"Write focused tests. Cover edge cases. Don't mock internals.",
	},
	{
		"DEPLOY",
		[]string{"deploy", "push", "release", "publish", "ship", "ci/cd", "pipeline"},
		"Verify tests pass. Check CI config. Confirm with user before pushing.",
	},
	{
		"REFACTOR",
		[]string{"refactor", "clean up", "simplify", "reorganize", "extract", "restructure"},
		"Preserve behavior. Run tests before and after. Small incremental changes.",
	},
	{
		"EXPLAIN",
		[]string{"explain", "how does", "what is", "why does", "describe", "walk through", "tell me about"},
		"Use /preview for visuals. Tailor depth to user's coding level.",
	},
	{
		"RESEARCH",
		[]string{"explore", "investigate", "analyze", "compare", "evaluate", "look into", "research"},
		"Gather info before proposing changes. Read docs, search code, compare approaches.",
	},
	{
		"IMPLEMENT",
		[]string{"build", "create", "add", "write", "implement", "develop", "set up", "feature", "make"},
		"Follow active plan. Write tests. Run compile after edits.",
	},
}

// Classify returns the intent category name for a prompt, or "" if no match.
// Uses first-match-wins on lowercased prompt with strings.Contains.
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

// Strategy returns the guidance text for a category name, or "" if not found.
func Strategy(categoryName string) string {
	for _, cat := range categories {
		if cat.Name == categoryName {
			return cat.Strategy
		}
	}
	return ""
}
