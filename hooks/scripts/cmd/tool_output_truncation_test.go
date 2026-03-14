package cmd

import (
	"strings"
	"testing"

	"solon-hooks/internal/truncation"
)

// TestTruncationBudgetForBash verifies Bash tool gets the larger 500-line budget.
func TestTruncationBudgetForBash(t *testing.T) {
	budget := truncation.BudgetForTool("Bash")
	if budget.MaxLines != 500 {
		t.Errorf("Bash MaxLines expected 500, got %d", budget.MaxLines)
	}
	if budget.HeadLines != 80 {
		t.Errorf("Bash HeadLines expected 80, got %d", budget.HeadLines)
	}
	if budget.TailLines != 50 {
		t.Errorf("Bash TailLines expected 50, got %d", budget.TailLines)
	}
}

// TestTruncationShortBashOutputUnchanged verifies short output is not modified.
func TestTruncationShortBashOutputUnchanged(t *testing.T) {
	output := strings.Repeat("line\n", 10)
	budget := truncation.BudgetForTool("Bash")
	result, changed := truncation.TruncateOutput(output, budget.MaxLines, budget.HeadLines, budget.TailLines)
	if changed {
		t.Errorf("short Bash output should not be truncated")
	}
	if result != output {
		t.Errorf("short output should be returned unchanged")
	}
}

// TestTruncationLongBashOutputTruncated verifies >500 line output is truncated with head/tail.
func TestTruncationLongBashOutputTruncated(t *testing.T) {
	lines := make([]string, 600)
	for i := range lines {
		lines[i] = "output line"
	}
	output := strings.Join(lines, "\n")
	budget := truncation.BudgetForTool("Bash")
	result, changed := truncation.TruncateOutput(output, budget.MaxLines, budget.HeadLines, budget.TailLines)
	if !changed {
		t.Errorf("600-line Bash output should be truncated")
	}
	if !strings.Contains(result, "truncated") {
		t.Errorf("truncated output should contain truncation marker, got: %q", result[:200])
	}
}

// TestTruncationBudgetForGrep verifies Grep gets a smaller 200-line budget.
func TestTruncationBudgetForGrep(t *testing.T) {
	budget := truncation.BudgetForTool("Grep")
	if budget.MaxLines != 200 {
		t.Errorf("Grep MaxLines expected 200, got %d", budget.MaxLines)
	}
	if budget.HeadLines != 40 {
		t.Errorf("Grep HeadLines expected 40, got %d", budget.HeadLines)
	}
}

// TestTruncationBudgetForUnknownTool verifies unknown tools get the default budget.
func TestTruncationBudgetForUnknownTool(t *testing.T) {
	budget := truncation.BudgetForTool("UnknownTool")
	if budget.MaxLines != truncation.DefaultMaxLines {
		t.Errorf("unknown tool MaxLines expected %d, got %d", truncation.DefaultMaxLines, budget.MaxLines)
	}
	if budget.HeadLines != truncation.DefaultHeadLines {
		t.Errorf("unknown tool HeadLines expected %d, got %d", truncation.DefaultHeadLines, budget.HeadLines)
	}
	if budget.TailLines != truncation.DefaultTailLines {
		t.Errorf("unknown tool TailLines expected %d, got %d", truncation.DefaultTailLines, budget.TailLines)
	}
}

// TestTruncationWhitelistContainsEditTools verifies write-type tools are whitelisted.
func TestTruncationWhitelistContainsEditTools(t *testing.T) {
	whitelistedTools := []string{"Edit", "Write", "MultiEdit", "NotebookEdit"}
	for _, tool := range whitelistedTools {
		if !truncationWhitelist[tool] {
			t.Errorf("tool %q should be in truncation whitelist", tool)
		}
	}
}
