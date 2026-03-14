package cmd

import (
	"testing"

	"solon-hooks/internal/intent"
)

// TestIntentClassifyDebugPrompt verifies debug-flavored prompts are classified as DEBUG.
func TestIntentClassifyDebugPrompt(t *testing.T) {
	prompts := []string{
		"fix this error in the login handler",
		"there is a bug in the parser",
		"the tests are failing after my change",
	}
	for _, p := range prompts {
		cat := intent.Classify(p)
		if cat != "DEBUG" {
			t.Errorf("prompt %q expected DEBUG, got %q", p, cat)
		}
	}
}

// TestIntentClassifyImplementPrompt verifies implementation prompts are classified as IMPLEMENT.
func TestIntentClassifyImplementPrompt(t *testing.T) {
	prompts := []string{
		"add a new feature to export CSV",
		"implement the user profile page",
		"build the authentication middleware",
	}
	for _, p := range prompts {
		cat := intent.Classify(p)
		if cat != "IMPLEMENT" {
			t.Errorf("prompt %q expected IMPLEMENT, got %q", p, cat)
		}
	}
}

// TestIntentClassifyResearchPrompt verifies research prompts are classified as RESEARCH.
func TestIntentClassifyResearchPrompt(t *testing.T) {
	prompts := []string{
		"investigate how the caching layer works",
		"compare these two approaches to rate limiting",
		"research options for background job processing",
	}
	for _, p := range prompts {
		cat := intent.Classify(p)
		if cat != "RESEARCH" {
			t.Errorf("prompt %q expected RESEARCH, got %q", p, cat)
		}
	}
}

// TestIntentClassifyExplainPrompt verifies explain prompts are classified as EXPLAIN.
func TestIntentClassifyExplainPrompt(t *testing.T) {
	cat := intent.Classify("explain how the context window compaction works")
	if cat != "EXPLAIN" {
		t.Errorf("explain prompt expected EXPLAIN, got %q", cat)
	}
}

// TestIntentClassifyUnknownPromptReturnsEmpty verifies unrecognised prompts return "".
func TestIntentClassifyUnknownPromptReturnsEmpty(t *testing.T) {
	cat := intent.Classify("hello there")
	if cat != "" {
		t.Errorf("unrecognised prompt should return empty string, got %q", cat)
	}
}

// TestIntentStrategyReturnsGuidanceForKnownCategory verifies Strategy returns non-empty text.
func TestIntentStrategyReturnsGuidanceForKnownCategory(t *testing.T) {
	categories := []string{"DEBUG", "IMPLEMENT", "RESEARCH", "EXPLAIN", "TEST", "DEPLOY", "REFACTOR"}
	for _, cat := range categories {
		s := intent.Strategy(cat)
		if s == "" {
			t.Errorf("Strategy(%q) should return guidance, got empty string", cat)
		}
	}
}

// TestIntentStrategyReturnsEmptyForUnknown verifies unknown category returns "".
func TestIntentStrategyReturnsEmptyForUnknown(t *testing.T) {
	s := intent.Strategy("UNKNOWN_CAT")
	if s != "" {
		t.Errorf("unknown category should return empty Strategy, got %q", s)
	}
}
