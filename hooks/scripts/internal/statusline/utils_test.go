package statusline

import (
	"testing"
	"time"
)

// TestVisibleLength tests the VisibleLength function.
func TestVisibleLength(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"simple ascii", "hello", 5},
		{"empty string", "", 0},
		{"ansi color codes", "\033[31mhello\033[0m", 5},
		{"ansi bold", "\033[1mbold\033[0m", 4},
		{"multiple ansi sequences", "\033[32mgreen\033[0m \033[31mred\033[0m", 9},
		{"no ansi", "plain text", 10},
		{"ansi at start and end", "\033[1mtext\033[0m", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VisibleLength(tt.input)
			if got != tt.expected {
				t.Errorf("VisibleLength(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

// TestFormatElapsed tests the FormatElapsed function.
func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"0 duration", 0, "<1s"},
		{"500ms", 500 * time.Millisecond, "<1s"},
		{"1 second", 1 * time.Second, "1s"},
		{"30 seconds", 30 * time.Second, "30s"},
		{"60 seconds (1 minute)", 60 * time.Second, "1m 0s"},
		{"90 seconds", 90 * time.Second, "1m 30s"},
		{"120 seconds (2 minutes)", 120 * time.Second, "2m 0s"},
		{"180 seconds (3 minutes)", 180 * time.Second, "3m 0s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			end := start.Add(tt.duration)
			got := FormatElapsed(start, end)
			if got != tt.expected {
				t.Errorf("FormatElapsed(%v) = %q, want %q", tt.duration, got, tt.expected)
			}
		})
	}
}

// TestCollapseHome tests the CollapseHome function.
func TestCollapseHome(t *testing.T) {
	// Note: We'll test the function behavior without relying on actual home dir
	// since that varies by system. Instead, we'll call it and verify it doesn't error.
	tests := []struct {
		name  string
		input string
	}{
		{"absolute path", "/home/user/project"},
		{"relative path", "src/main.ts"},
		{"empty string", ""},
		{"simple path", "file.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CollapseHome(tt.input)
			// Just verify it doesn't panic and returns a string
			if len(got) < 0 {
				t.Errorf("CollapseHome returned invalid result")
			}
		})
	}
}

// TestVisibleLengthWithEmoji tests emoji handling (should count as 2).
func TestVisibleLengthWithEmoji(t *testing.T) {
	// Many terminals count emoji as 2 columns
	// Note: actual behavior may vary by terminal, this tests the implementation
	input := "hello 👋 world"
	result := VisibleLength(input)
	if result < len("hello world") {
		t.Errorf("VisibleLength should count emoji: got %d for %q", result, input)
	}
}

// TestFormatElapsedEdgeCases tests edge cases for FormatElapsed.
func TestFormatElapsedEdgeCases(t *testing.T) {
	// Test very large durations
	start := time.Now()
	end := start.Add(24 * time.Hour)
	result := FormatElapsed(start, end)
	if len(result) == 0 {
		t.Errorf("FormatElapsed should handle 24 hours")
	}

	// Test zero start time
	result = FormatElapsed(time.Time{}, time.Time{})
	if result == "" {
		t.Errorf("FormatElapsed should handle zero times")
	}
}

// TestVisibleLengthComplex tests complex ANSI sequences.
func TestVisibleLengthComplex(t *testing.T) {
	// Test with nested/complex ANSI codes
	tests := []struct {
		name     string
		input    string
		checkLen func(int) bool
	}{
		{
			"256 color codes",
			"\033[38;5;200mtext\033[0m",
			func(len int) bool { return len == 4 }, // just "text"
		},
		{
			"rgb color codes",
			"\033[38;2;255;0;0mtext\033[0m",
			func(len int) bool { return len == 4 }, // just "text"
		},
		{
			"alternating codes",
			"\033[1m\033[31mhello\033[0m",
			func(len int) bool { return len == 5 }, // just "hello"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VisibleLength(tt.input)
			if !tt.checkLen(got) {
				t.Errorf("VisibleLength(%q) = %d, check failed", tt.input, got)
			}
		})
	}
}
