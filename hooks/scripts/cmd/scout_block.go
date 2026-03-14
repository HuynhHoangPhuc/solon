package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"

	"solon-hooks/internal/config"
	"solon-hooks/internal/logging"
	"solon-hooks/internal/scout"

	"github.com/spf13/cobra"
)

var scoutBlockCmd = &cobra.Command{
	Use:   "scout-block",
	Short: "Handle PreToolUse scout-block (blocks disallowed paths/commands)",
	RunE:  runScoutBlock,
}

func runScoutBlock(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("scout-block") {
		os.Exit(0)
	}

	timer := logging.CreateHookTimer("scout-block")

	var raw json.RawMessage
	if err := json.NewDecoder(os.Stdin).Decode(&raw); err != nil || len(raw) == 0 {
		os.Stderr.WriteString("ERROR: Empty input\n")
		timer.End(logging.LogData{Status: "error", Exit: 2})
		os.Exit(2)
	}

	var data struct {
		ToolName  string                 `json:"tool_name"`
		ToolInput map[string]interface{} `json:"tool_input"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		os.Stderr.WriteString("WARN: JSON parse failed, allowing operation\n")
		timer.End(logging.LogData{Status: "ok", Exit: 0})
		os.Exit(0)
	}

	if data.ToolInput == nil {
		os.Stderr.WriteString("WARN: Invalid JSON structure, allowing operation\n")
		timer.End(logging.LogData{Status: "ok", Exit: 0})
		os.Exit(0)
	}

	toolName := data.ToolName
	if toolName == "" {
		toolName = "unknown"
	}

	cwd, _ := os.Getwd()
	claudeDir := filepath.Join(cwd, ".claude")

	result := scout.CheckScoutBlock(toolName, data.ToolInput, scout.CheckOptions{
		ClaudeDir:          claudeDir,
		SlignorePath:       filepath.Join(cwd, ".claude", ".slignore"),
		CheckBroadPatterns: true,
	})

	if result.IsAllowedCommand {
		timer.End(logging.LogData{Tool: toolName, Status: "ok", Exit: 0})
		os.Exit(0)
	}

	if result.Blocked && result.IsBroadPattern {
		msg := scout.FormatBroadPatternError(result.Pattern, result.Suggestions)
		os.Stderr.WriteString(msg)
		timer.End(logging.LogData{Tool: toolName, Status: "block", Exit: 2})
		os.Exit(2)
	}

	if result.Blocked {
		msg := scout.FormatBlockedError(result.Path, result.Pattern, toolName, claudeDir)
		os.Stderr.WriteString(msg)
		timer.End(logging.LogData{Tool: toolName, Status: "block", Exit: 2})
		os.Exit(2)
	}

	timer.End(logging.LogData{Tool: toolName, Status: "ok", Exit: 0})
	os.Exit(0)
	return nil
}
