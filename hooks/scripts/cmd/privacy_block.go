package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"solon-hooks/internal/config"
	"solon-hooks/internal/privacy"

	"github.com/spf13/cobra"
)

var privacyBlockCmd = &cobra.Command{
	Use:   "privacy-block",
	Short: "Handle PreToolUse privacy-block (blocks sensitive file access)",
	RunE:  runPrivacyBlock,
}

func runPrivacyBlock(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("privacy-block") {
		os.Exit(0)
	}

	var input struct {
		ToolName  string                 `json:"tool_name"`
		ToolInput map[string]interface{} `json:"tool_input"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		os.Exit(0)
	}
	if input.ToolInput == nil || input.ToolName == "" {
		os.Exit(0)
	}

	result := privacy.CheckPrivacy(input.ToolName, input.ToolInput, privacy.PrivacyOpts{AllowBash: true})

	if result.Approved {
		if result.Suspicious {
			fmt.Fprintf(os.Stderr, "\x1b[33mWARN:\x1b[0m Approved path is outside project: %s\n", result.FilePath)
		}
		fmt.Fprintf(os.Stderr, "\x1b[32m✓\x1b[0m Privacy: User-approved access to %s\n", filepath.Base(result.FilePath))
		os.Exit(0)
	}

	if result.IsBash {
		fmt.Fprintf(os.Stderr, "\x1b[33mWARN:\x1b[0m %s\n", result.Reason)
		os.Exit(0)
	}

	if result.Blocked {
		promptJSON, _ := json.MarshalIndent(result.PromptData, "", "  ")
		os.Stderr.WriteString(formatPrivacyBlockMsg(result.FilePath, string(promptJSON)))
		os.Exit(2)
	}

	os.Exit(0)
	return nil
}

func formatPrivacyBlockMsg(filePath, promptJSON string) string {
	return fmt.Sprintf(
		"\n\x1b[36mNOTE:\x1b[0m This is not an error - this block protects sensitive data.\n\n"+
			"\x1b[33mPRIVACY BLOCK\x1b[0m: Sensitive file access requires user approval\n\n"+
			"  \x1b[33mFile:\x1b[0m %s\n\n"+
			"  This file may contain secrets (API keys, passwords, tokens).\n\n"+
			"\x1b[90m@@PRIVACY_PROMPT_START@@\x1b[0m\n"+
			"%s\n"+
			"\x1b[90m@@PRIVACY_PROMPT_END@@\x1b[0m\n\n"+
			"  \x1b[34mClaude:\x1b[0m Use AskUserQuestion tool with the JSON above, then:\n"+
			"  \x1b[32mIf \"Yes\":\x1b[0m Use bash to read: cat \"%s\"\n"+
			"  \x1b[31mIf \"No\":\x1b[0m  Continue without this file.\n",
		filePath, promptJSON, filePath,
	)
}
