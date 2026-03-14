package intent

import "testing"

func TestClassifyReturnsExpectedCategories(t *testing.T) {
	tests := []struct {
		prompt string
		want   string
	}{
		{"fix the login bug", "DEBUG"},
		{"there's an error in the handler", "DEBUG"},
		{"the build is broken", "DEBUG"},
		{"write unit tests for auth", "TEST"},
		{"check coverage for the api package", "TEST"},
		{"deploy to production", "DEPLOY"},
		{"release version 2.0", "DEPLOY"},
		{"refactor the database layer", "REFACTOR"},
		{"clean up the config code", "REFACTOR"},
		{"explain how the hook system works", "EXPLAIN"},
		{"what is the intent gate", "EXPLAIN"},
		{"research authentication libraries", "RESEARCH"},
		{"compare postgres vs mysql", "RESEARCH"},
		{"implement the user profile feature", "IMPLEMENT"},
		{"add a new endpoint", "IMPLEMENT"},
		{"build the dashboard", "IMPLEMENT"},
	}

	for _, tt := range tests {
		t.Run(tt.prompt, func(t *testing.T) {
			got := Classify(tt.prompt)
			if got != tt.want {
				t.Errorf("Classify(%q) = %q, want %q", tt.prompt, got, tt.want)
			}
		})
	}
}

func TestClassifyPriorityDebugBeforeImplement(t *testing.T) {
	// "fix" (DEBUG) wins over "add" (IMPLEMENT)
	got := Classify("fix and add a new feature")
	if got != "DEBUG" {
		t.Errorf("Classify should return DEBUG before IMPLEMENT, got %q", got)
	}
}

func TestClassifyPriorityDebugBeforeTest(t *testing.T) {
	// "fix the test" — DEBUG wins over TEST
	got := Classify("fix the test that is failing")
	if got != "DEBUG" {
		t.Errorf("Classify should return DEBUG before TEST, got %q", got)
	}
}

func TestClassifyNoMatchReturnsEmpty(t *testing.T) {
	tests := []string{"yes", "ok", "sure", "thanks", "hello", ""}
	for _, prompt := range tests {
		got := Classify(prompt)
		if got != "" {
			t.Errorf("Classify(%q) = %q, want empty string", prompt, got)
		}
	}
}

func TestClassifyCaseInsensitive(t *testing.T) {
	tests := []struct {
		prompt string
		want   string
	}{
		{"FIX the bug", "DEBUG"},
		{"IMPLEMENT the feature", "IMPLEMENT"},
		{"EXPLAIN how this works", "EXPLAIN"},
	}
	for _, tt := range tests {
		got := Classify(tt.prompt)
		if got != tt.want {
			t.Errorf("Classify(%q) = %q, want %q", tt.prompt, got, tt.want)
		}
	}
}

func TestStrategyReturnsGuidance(t *testing.T) {
	categories := []string{"DEBUG", "TEST", "DEPLOY", "REFACTOR", "EXPLAIN", "RESEARCH", "IMPLEMENT"}
	for _, cat := range categories {
		got := Strategy(cat)
		if got == "" {
			t.Errorf("Strategy(%q) returned empty string", cat)
		}
	}
}

func TestStrategyUnknownReturnsEmpty(t *testing.T) {
	got := Strategy("UNKNOWN")
	if got != "" {
		t.Errorf("Strategy(UNKNOWN) = %q, want empty string", got)
	}
}
