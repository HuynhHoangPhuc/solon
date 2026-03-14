// PostToolUse hook: detects AI-generated comment patterns in code edits and warns.
package cmd

import (
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"solon-hooks/internal/config"
	"solon-hooks/internal/hookio"
)

var commentSlopCheckerCmd = &cobra.Command{
	Use:   "comment-slop-checker",
	Short: "Detect AI-generated comment patterns in code edits",
	RunE:  runCommentSlopChecker,
}

// slopPatterns matches common AI-generated comment patterns across //, #, /* styles.
var slopPatterns = []*regexp.Regexp{
	// "This function/method/class handles/does/performs..."
	regexp.MustCompile(`(?m)^\s*(?://|#|/\*)\s*This (?:function|method|class|module|component|hook|helper) (?:handles|does|performs|is responsible|takes care|manages|implements|provides|creates|returns|checks)`),
	// "Updated to..." / "Modified to..." / "Changed to..."
	regexp.MustCompile(`(?m)^\s*(?://|#|/\*)\s*(?:Updated|Modified|Changed|Refactored|Fixed|Added|Removed) (?:to |the |for |by )`),
	// "Helper function/method that..."
	regexp.MustCompile(`(?m)^\s*(?://|#|/\*)\s*Helper (?:function|method|class|utility) (?:that|which|to|for)`),
	// "We need to..." / "We use..." / "Here we..."
	regexp.MustCompile(`(?m)^\s*(?://|#|/\*)\s*(?:We|Here we) (?:need to|use|create|define|implement|check|handle|call)`),
	// "The following..." / "Below is..." / "This is the..."
	regexp.MustCompile(`(?m)^\s*(?://|#|/\*)\s*(?:The following|Below is|Above is|This is the)`),
}

func runCommentSlopChecker(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("comment-slop-checker") {
		os.Exit(0)
	}

	var input hookio.PostToolUseInput
	if err := hookio.ReadInput(&input); err != nil {
		hookio.WriteOutput(map[string]interface{}{"continue": true})
		return nil
	}

	content := extractWrittenContent(input.ToolName, input.ToolInput)
	if content == "" {
		hookio.WriteOutput(map[string]interface{}{"continue": true})
		return nil
	}

	matchCount := 0
	for _, pat := range slopPatterns {
		matchCount += len(pat.FindAllStringIndex(content, -1))
	}

	output := map[string]interface{}{"continue": true}
	if matchCount > 0 {
		output["additionalContext"] = "\n[Comment Quality] Detected AI-generated comment pattern(s) in your edit. " +
			"Avoid narrating what code does (\"This function handles...\", \"Updated to...\"). " +
			"Comments should explain WHY, not WHAT. Remove or rewrite slop comments.\n"
	}

	hookio.WriteOutput(output)
	return nil
}

// extractWrittenContent gets the code string from tool_input based on tool type.
func extractWrittenContent(toolName string, input map[string]interface{}) string {
	if input == nil {
		return ""
	}
	switch toolName {
	case "Edit":
		s, _ := input["new_string"].(string)
		return s
	case "Write":
		s, _ := input["content"].(string)
		return s
	case "MultiEdit":
		edits, ok := input["edits"].([]interface{})
		if !ok {
			return ""
		}
		var parts []string
		for _, e := range edits {
			if m, ok := e.(map[string]interface{}); ok {
				if s, ok := m["new_string"].(string); ok {
					parts = append(parts, s)
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}
