package cmd

import (
	"strings"
	"testing"
)

// TestFileNamingGuidanceContainsKebabCase verifies guidance mentions kebab-case for JS/TS/Python.
func TestFileNamingGuidanceContainsKebabCase(t *testing.T) {
	if !strings.Contains(fileNamingGuidance, "kebab-case") {
		t.Errorf("fileNamingGuidance should mention kebab-case, got: %q", fileNamingGuidance)
	}
}

// TestFileNamingGuidanceContainsLanguageConventions verifies guidance mentions language-specific conventions.
func TestFileNamingGuidanceContainsLanguageConventions(t *testing.T) {
	// Should mention Go/Rust snake_case convention
	if !strings.Contains(fileNamingGuidance, "snake_case") {
		t.Errorf("fileNamingGuidance should mention snake_case for Go/Rust, got: %q", fileNamingGuidance)
	}
	// Should mention PascalCase for C#/Java
	if !strings.Contains(fileNamingGuidance, "PascalCase") {
		t.Errorf("fileNamingGuidance should mention PascalCase for C#/Java, got: %q", fileNamingGuidance)
	}
}

// TestFileNamingGuidanceMentionsGoalContext verifies guidance mentions the LLM tooling goal.
func TestFileNamingGuidanceMentionsGoalContext(t *testing.T) {
	if !strings.Contains(fileNamingGuidance, "LLM") {
		t.Errorf("fileNamingGuidance should mention LLM tools goal, got: %q", fileNamingGuidance)
	}
}
