package cmd

import (
	"testing"
)

// TestExtractWrittenContentEdit verifies new_string is extracted for Edit tool.
func TestExtractWrittenContentEdit(t *testing.T) {
	input := map[string]interface{}{
		"new_string": "func foo() {}",
	}
	result := extractWrittenContent("Edit", input)
	if result != "func foo() {}" {
		t.Errorf("expected new_string for Edit, got: %q", result)
	}
}

// TestExtractWrittenContentWrite verifies content field is extracted for Write tool.
func TestExtractWrittenContentWrite(t *testing.T) {
	input := map[string]interface{}{
		"content": "package main\n",
	}
	result := extractWrittenContent("Write", input)
	if result != "package main\n" {
		t.Errorf("expected content for Write, got: %q", result)
	}
}

// TestExtractWrittenContentUnknownTool verifies unknown tools return empty string.
func TestExtractWrittenContentUnknownTool(t *testing.T) {
	input := map[string]interface{}{
		"new_string": "something",
	}
	result := extractWrittenContent("Bash", input)
	if result != "" {
		t.Errorf("unknown tool should return empty string, got: %q", result)
	}
}

// TestExtractWrittenContentNilInput verifies nil input returns empty string.
func TestExtractWrittenContentNilInput(t *testing.T) {
	result := extractWrittenContent("Edit", nil)
	if result != "" {
		t.Errorf("nil input should return empty string, got: %q", result)
	}
}

// TestSlopPatternsDetectThisFunction verifies "This function handles..." is detected.
func TestSlopPatternsDetectThisFunction(t *testing.T) {
	code := "// This function handles the user authentication\nfunc auth() {}"
	matchCount := countSlopMatches(code)
	if matchCount == 0 {
		t.Errorf("expected slop match for 'This function handles...', got 0")
	}
}

// TestSlopPatternsDetectUpdatedTo verifies "Updated to..." is detected.
func TestSlopPatternsDetectUpdatedTo(t *testing.T) {
	code := "// Updated to use the new API\nfunc fetch() {}"
	matchCount := countSlopMatches(code)
	if matchCount == 0 {
		t.Errorf("expected slop match for 'Updated to...', got 0")
	}
}

// TestSlopPatternsCleanCodeNoMatch verifies clean idiomatic code produces no slop match.
func TestSlopPatternsCleanCodeNoMatch(t *testing.T) {
	// Comments explaining WHY, not WHAT
	code := "// retry on transient network failures (max 3 attempts)\nfunc fetch() {}"
	matchCount := countSlopMatches(code)
	if matchCount != 0 {
		t.Errorf("clean code should produce 0 slop matches, got %d", matchCount)
	}
}

// TestSlopPatternsCodeOnlyNoMatch verifies code with no comments produces no match.
func TestSlopPatternsCodeOnlyNoMatch(t *testing.T) {
	code := "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"
	matchCount := countSlopMatches(code)
	if matchCount != 0 {
		t.Errorf("code-only content should produce 0 slop matches, got %d", matchCount)
	}
}

// countSlopMatches is a test helper that reuses the package-level slopPatterns slice.
func countSlopMatches(content string) int {
	total := 0
	for _, pat := range slopPatterns {
		total += len(pat.FindAllStringIndex(content, -1))
	}
	return total
}
