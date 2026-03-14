// PreToolUse(Write): Inject file naming guidance as allow response.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"solon-hooks/internal/config"
	"solon-hooks/internal/hookio"
)

var descriptiveNameCmd = &cobra.Command{
	Use:   "descriptive-name",
	Short: "Handle PreToolUse Write file naming guidance",
	RunE:  runDescriptiveName,
}

const fileNamingGuidance = `## File naming guidance:
- Skip this guidance if you are creating markdown or plain text files
- Prefer kebab-case for JS/TS/Python/shell (.js, .ts, .py, .sh) with descriptive names
- Respect language conventions: C#/Java/Kotlin/Swift use PascalCase (.cs, .java, .kt, .swift), Go/Rust use snake_case (.go, .rs)
- Other languages: follow their ecosystem's standard naming convention
- Goal: self-documenting names for LLM tools (Grep, Glob, Search)`

func runDescriptiveName(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("descriptive-name") {
		os.Exit(0)
	}

	hookio.WriteOutput(map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     "PreToolUse",
			"permissionDecision": "allow",
			"additionalContext": fileNamingGuidance,
		},
	})
	return nil
}
