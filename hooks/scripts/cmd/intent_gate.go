// UserPromptSubmit hook: classifies user intent and injects compact strategy guidance.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"solon-hooks/internal/config"
	"solon-hooks/internal/hookio"
	"solon-hooks/internal/intent"
)

var intentGateCmd = &cobra.Command{
	Use:   "intent-gate",
	Short: "Classify user intent and inject strategy guidance",
	RunE:  runIntentGate,
}

func runIntentGate(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("intent-gate") {
		os.Exit(0)
	}

	defer func() {
		if r := recover(); r != nil {
			os.Exit(0)
		}
	}()

	var input hookio.UserPromptSubmitInput
	if err := hookio.ReadInput(&input); err != nil || input.Prompt == "" {
		os.Exit(0)
	}

	category := intent.Classify(input.Prompt)
	if category == "" {
		os.Exit(0)
	}

	strategy := intent.Strategy(category)
	hookio.WriteContext(fmt.Sprintf("[Intent: %s] %s\n", category, strategy))
	return nil
}
