package truncation

import (
	"strings"
	"testing"
)

// TestBudgetForTool tests known and unknown tool budget lookups.
func TestBudgetForTool(t *testing.T) {
	tests := []struct {
		name      string
		toolName  string
		wantMax   int
		wantHead  int
		wantTail  int
	}{
		{"Bash known tool", "Bash", 500, 80, 50},
		{"Grep known tool", "Grep", 200, 40, 20},
		{"Read known tool", "Read", 300, 60, 30},
		{"Glob known tool", "Glob", 150, 30, 20},
		{"unknown tool uses defaults", "Write", DefaultMaxLines, DefaultHeadLines, DefaultTailLines},
		{"empty string uses defaults", "", DefaultMaxLines, DefaultHeadLines, DefaultTailLines},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := BudgetForTool(tt.toolName)
			if b.MaxLines != tt.wantMax {
				t.Errorf("BudgetForTool(%q).MaxLines = %d, want %d", tt.toolName, b.MaxLines, tt.wantMax)
			}
			if b.HeadLines != tt.wantHead {
				t.Errorf("BudgetForTool(%q).HeadLines = %d, want %d", tt.toolName, b.HeadLines, tt.wantHead)
			}
			if b.TailLines != tt.wantTail {
				t.Errorf("BudgetForTool(%q).TailLines = %d, want %d", tt.toolName, b.TailLines, tt.wantTail)
			}
		})
	}
}

// TestTruncateOutput_Empty tests that empty input is returned unchanged.
func TestTruncateOutput_Empty(t *testing.T) {
	out, changed := TruncateOutput("", 100, 20, 10)
	if changed {
		t.Error("empty input should not be changed")
	}
	if out != "" {
		t.Errorf("empty input should return empty, got %q", out)
	}
}

// TestTruncateOutput_Short tests that short output (under maxLines) is unchanged.
func TestTruncateOutput_Short(t *testing.T) {
	input := strings.Repeat("line\n", 10) // 10 lines
	out, changed := TruncateOutput(input, 100, 20, 10)
	if changed {
		t.Error("short output should not be changed")
	}
	if out != input {
		t.Error("short output should be returned as-is")
	}
}

// TestTruncateOutput_Long tests that long output is truncated with a marker.
func TestTruncateOutput_Long(t *testing.T) {
	// Build 200 lines: "line 0" ... "line 199"
	var lines []string
	for i := 0; i < 200; i++ {
		lines = append(lines, "line content here")
	}
	input := strings.Join(lines, "\n")
	maxLines := 100
	headLines := 20
	tailLines := 10

	out, changed := TruncateOutput(input, maxLines, headLines, tailLines)
	if !changed {
		t.Fatal("long output should be changed")
	}

	// Marker must be present
	if !strings.Contains(out, "lines truncated") {
		t.Error("truncated output should contain marker")
	}

	// Dropped line count: 200 - 20 - 10 = 170
	if !strings.Contains(out, "170 lines truncated") {
		t.Errorf("marker should report 170 dropped lines, got output: %s", out)
	}

	// Head: first 20 lines, tail: last 10 lines
	outLines := strings.Split(out, "\n")
	// Output starts with head lines
	if !strings.HasPrefix(out, "line content here") {
		t.Error("output should start with head lines")
	}
	// Output ends with tail lines
	if !strings.HasSuffix(strings.TrimRight(out, "\n"), "line content here") {
		t.Errorf("output should end with tail lines, last part: %q", outLines[len(outLines)-1])
	}
}

// TestTruncateOutput_EdgeCase tests that headLines+tailLines >= total lines means no change.
func TestTruncateOutput_EdgeCase(t *testing.T) {
	// 10 lines, headLines=6, tailLines=6 → dropped = 10-6-6 = -2 → no truncation
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, "line")
	}
	input := strings.Join(lines, "\n")
	out, changed := TruncateOutput(input, 5, 6, 6)
	if changed {
		t.Error("when head+tail >= total lines, output should not change")
	}
	if out != input {
		t.Error("edge case output should equal input")
	}
}

// TestTruncateOutput_ExactBoundary tests output with exactly maxLines is not truncated.
func TestTruncateOutput_ExactBoundary(t *testing.T) {
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "line")
	}
	input := strings.Join(lines, "\n")
	out, changed := TruncateOutput(input, 50, 20, 10)
	if changed {
		t.Error("output at exactly maxLines should not be changed")
	}
	if out != input {
		t.Error("boundary output should equal input")
	}
}
