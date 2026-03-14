package compress

import (
	"testing"
)

func TestCompressTextEmpty(t *testing.T) {
	result, changed := CompressText("")
	if changed {
		t.Error("expected no change for empty input")
	}
	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestCompressTextStripsArticles(t *testing.T) {
	input := "Read the file and check a value in an array"
	result, changed := CompressText(input)
	if !changed {
		t.Error("expected change")
	}
	if result == input {
		t.Error("expected articles to be stripped")
	}
	// Should not contain standalone "the ", "a ", "an "
	for _, article := range []string{"the file", "a value", "an array"} {
		if contains(result, article) {
			t.Errorf("expected %q to be stripped from result: %q", article, result)
		}
	}
}

func TestCompressTextStripsCopulas(t *testing.T) {
	input := "This is important and values are required"
	result, changed := CompressText(input)
	if !changed {
		t.Error("expected change")
	}
	// "is" and "are" should be stripped
	if contains(result, " is ") {
		t.Errorf("expected 'is' stripped from: %q", result)
	}
	if contains(result, " are ") {
		t.Errorf("expected 'are' stripped from: %q", result)
	}
}

func TestCompressTextStripsFillers(t *testing.T) {
	input := "You basically need to actually check the very important value"
	result, changed := CompressText(input)
	if !changed {
		t.Error("expected change")
	}
	for _, filler := range []string{"basically", "actually", "very"} {
		if contains(result, filler) {
			t.Errorf("expected filler %q stripped from: %q", filler, result)
		}
	}
}

func TestCompressTextVerbosePhrases(t *testing.T) {
	tests := []struct {
		input    string
		contains string
	}{
		{"Do this in order to succeed", "to succeed"},
		{"Due to the fact that it failed", "because it failed"},
		{"Make sure to check errors", "ensure check errors"},
	}
	for _, tt := range tests {
		result, _ := CompressText(tt.input)
		if !containsCI(result, tt.contains) {
			t.Errorf("input=%q: expected %q in result %q", tt.input, tt.contains, result)
		}
	}
}

func TestCompressTextPreservesCodeBlocks(t *testing.T) {
	input := "Some text\n```\nthe value is basically important\n```\nMore text"
	result, _ := CompressText(input)
	// Code block content should be untouched
	if !contains(result, "the value is basically important") {
		t.Errorf("code block content was modified: %q", result)
	}
}

func TestCompressTextPreservesInlineCode(t *testing.T) {
	input := "Check `the important value` in the config"
	result, _ := CompressText(input)
	if !contains(result, "`the important value`") {
		t.Errorf("inline code was modified: %q", result)
	}
}

func TestCompressTextNoChangeOnCodeOnly(t *testing.T) {
	input := "```\nfn main() {\n    println!(\"hello\");\n}\n```"
	result, changed := CompressText(input)
	if changed {
		t.Errorf("expected no change for code-only input, got: %q", result)
	}
}

func TestCompressTextCollapsesMultipleSpaces(t *testing.T) {
	// After stripping words, multiple spaces should collapse
	input := "The very important task"
	result, _ := CompressText(input)
	if contains(result, "  ") {
		t.Errorf("expected no double spaces in: %q", result)
	}
}

func TestSplitPreservingCodeBasic(t *testing.T) {
	parts := splitPreservingCode("hello `world` foo")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}
	if parts[0].isCode || parts[0].text != "hello " {
		t.Errorf("part 0: %+v", parts[0])
	}
	if !parts[1].isCode || parts[1].text != "world" {
		t.Errorf("part 1: %+v", parts[1])
	}
	if parts[2].isCode || parts[2].text != " foo" {
		t.Errorf("part 2: %+v", parts[2])
	}
}

func TestSplitPreservingCodeUnmatched(t *testing.T) {
	parts := splitPreservingCode("hello `world")
	// Unmatched backtick should not panic
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d: %+v", len(parts), parts)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func containsCI(s, substr string) bool {
	ls := lower(s)
	lsub := lower(substr)
	return indexOf(ls, lsub) >= 0
}

func lower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		} else {
			b[i] = c
		}
	}
	return string(b)
}
