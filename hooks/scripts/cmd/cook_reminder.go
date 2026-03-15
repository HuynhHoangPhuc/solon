// SubagentStop(Plan): Remind to run /cook after plan agent finishes.
package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"solon-hooks/internal/config"
	"solon-hooks/internal/hookio"
	"solon-hooks/internal/session"
)

var cookReminderCmd = &cobra.Command{
	Use:   "cook-reminder",
	Short: "Handle SubagentStop cook-after-plan reminder",
	RunE:  runCookReminder,
}

func runCookReminder(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("cook-after-plan-reminder") {
		os.Exit(0)
	}

	defer func() {
		if r := recover(); r != nil {
			os.Exit(0)
		}
	}()

	// Consume stdin (not used)
	var input hookio.SubagentStopInput
	_ = hookio.ReadInput(&input)

	sessionID := os.Getenv("SL_SESSION_ID")
	var planPath string

	if sessionID != "" {
		state := session.ReadSessionState(sessionID)
		if state != nil && state.ActivePlan != nil {
			planPath = *state.ActivePlan
			if !filepath.IsAbs(planPath) && state.SessionOrigin != "" {
				planPath = filepath.Join(state.SessionOrigin, planPath)
			}
		}
	}

	hookio.WriteContext("MUST invoke /solon:cook --auto skill before implementing the plan\n")
	if planPath != "" {
		hookio.WriteContext("Best Practice: Run /clear then /solon:cook " + filepath.Join(planPath, "plan.md") + "\n")
	} else {
		hookio.WriteContext("Best Practice: Run /clear then /solon:cook {full-absolute-path-to-plan.md}\n")
	}
	return nil
}
