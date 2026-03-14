// Package truncation provides line-based output truncation for large tool outputs.
package truncation

import (
	"fmt"
	"strings"
)

const (
	DefaultMaxLines  = 200
	DefaultHeadLines = 50
	DefaultTailLines = 30
)

// ToolBudget defines per-tool truncation limits.
type ToolBudget struct {
	MaxLines  int
	HeadLines int
	TailLines int
}

// ToolBudgets maps Claude Code tool names to their output budgets.
// Tools not listed here use the default limits.
var ToolBudgets = map[string]ToolBudget{
	"Bash": {MaxLines: 500, HeadLines: 80, TailLines: 50},
	"Grep": {MaxLines: 200, HeadLines: 40, TailLines: 20},
	"Read": {MaxLines: 300, HeadLines: 60, TailLines: 30},
	"Glob": {MaxLines: 150, HeadLines: 30, TailLines: 20},
}

// BudgetForTool returns the truncation budget for a tool, falling back to defaults.
func BudgetForTool(toolName string) ToolBudget {
	if b, ok := ToolBudgets[toolName]; ok {
		return b
	}
	return ToolBudget{
		MaxLines:  DefaultMaxLines,
		HeadLines: DefaultHeadLines,
		TailLines: DefaultTailLines,
	}
}

// TruncateOutput trims output to headLines+tailLines if it exceeds maxLines.
// Returns (result, wasChanged). Pure function, safe for concurrent use.
func TruncateOutput(output string, maxLines, headLines, tailLines int) (string, bool) {
	if output == "" {
		return output, false
	}
	lines := strings.Split(output, "\n")
	if len(lines) <= maxLines {
		return output, false
	}

	dropped := len(lines) - headLines - tailLines
	if dropped <= 0 {
		return output, false
	}

	head := strings.Join(lines[:headLines], "\n")
	tail := strings.Join(lines[len(lines)-tailLines:], "\n")
	marker := fmt.Sprintf("\n\n... [%d lines truncated for context efficiency] ...\n\n", dropped)

	return head + marker + tail, true
}
