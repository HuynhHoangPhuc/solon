// UserPromptSubmit: Inject dev rules reminder context (throttled by transcript check).
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"solon-hooks/internal/config"
	ctx "solon-hooks/internal/context"
	"solon-hooks/internal/hookio"
)

var devRulesCmd = &cobra.Command{
	Use:   "dev-rules",
	Short: "Handle UserPromptSubmit dev-rules reminder",
	RunE:  runDevRules,
}

func runDevRules(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("dev-rules-reminder") {
		os.Exit(0)
	}

	var input hookio.UserPromptSubmitInput
	if err := hookio.ReadInput(&input); err != nil {
		hookio.Log("dev-rules", err.Error())
		os.Exit(0)
	}

	if ctx.WasRecentlyInjected(input.TranscriptPath) {
		os.Exit(0)
	}

	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = os.Getenv("SL_SESSION_ID")
	}

	baseDir, _ := os.Getwd()
	result := ctx.BuildReminderContext(ctx.BuildReminderOpts{
		SessionID: sessionID,
		BaseDir:   baseDir,
	})
	hookio.WriteContext(result.Content)
	return nil
}
