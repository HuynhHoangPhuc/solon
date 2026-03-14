// PostToolUse hook: truncates large tool outputs to save context window space.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"solon-hooks/internal/config"
	"solon-hooks/internal/hookio"
	"solon-hooks/internal/truncation"
)

// truncationWhitelist contains tools whose output must never be truncated.
// Claude needs full output from these to verify file changes correctly.
var truncationWhitelist = map[string]bool{
	"Edit":         true,
	"Write":        true,
	"MultiEdit":    true,
	"NotebookEdit": true,
}

var toolOutputTruncationCmd = &cobra.Command{
	Use:   "tool-output-truncation",
	Short: "Truncate large tool outputs to save context",
	RunE:  runToolOutputTruncation,
}

func runToolOutputTruncation(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("tool-output-truncation") {
		os.Exit(0)
	}

	var input hookio.PostToolUseInput
	if err := hookio.ReadInput(&input); err != nil {
		hookio.WriteOutput(map[string]interface{}{"continue": true})
		return nil
	}

	if truncationWhitelist[input.ToolName] {
		hookio.WriteOutput(map[string]interface{}{"continue": true})
		return nil
	}

	budget := truncation.BudgetForTool(input.ToolName)
	result, changed := truncation.TruncateOutput(
		input.ToolOutput,
		budget.MaxLines,
		budget.HeadLines,
		budget.TailLines,
	)

	if !changed {
		hookio.WriteOutput(map[string]interface{}{"continue": true})
		return nil
	}

	hookio.WriteOutput(map[string]interface{}{
		"continue": true,
		"hookSpecificOutput": map[string]interface{}{
			"tool_output": result,
		},
	})
	return nil
}
