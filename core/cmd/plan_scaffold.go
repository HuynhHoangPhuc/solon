package cmd

import (
	"encoding/json"
	"fmt"

	"solon-core/internal/config"
	"solon-core/internal/plan"

	"github.com/spf13/cobra"
)

var planScaffoldCmd = &cobra.Command{
	Use:   "scaffold",
	Short: "Create plan directory with template files",
	Long:  "Scaffold a new plan directory with plan.md and phase template files.",
	RunE: func(cmd *cobra.Command, args []string) error {
		slug, _ := cmd.Flags().GetString("slug")
		if slug == "" {
			return fmt.Errorf("--slug is required")
		}
		modeStr, _ := cmd.Flags().GetString("mode")
		phases, _ := cmd.Flags().GetInt("phases")

		mode := plan.ModeHard
		switch modeStr {
		case "fast":
			mode = plan.ModeFast
		case "hard":
			mode = plan.ModeHard
		case "parallel":
			mode = plan.ModeParallel
		case "two":
			mode = plan.ModeTwo
		}

		cfg := config.LoadConfig()
		result, err := plan.ScaffoldPlan(slug, mode, phases, &cfg)
		if err != nil {
			return fmt.Errorf("scaffold: %w", err)
		}

		format, _ := cmd.Flags().GetString("format")
		if format == "text" {
			fmt.Printf("Created plan: %s (mode: %s)\n", result.PlanDir, result.Mode)
			for _, f := range result.FilesCreated {
				fmt.Printf("  + %s\n", f)
			}
			return nil
		}

		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal result: %w", err)
		}
		fmt.Println(string(data))
		return nil
	},
}

func init() {
	planScaffoldCmd.Flags().String("slug", "", "Plan slug (required)")
	planScaffoldCmd.Flags().String("mode", "hard", "Scaffold mode: fast, hard, parallel, two")
	planScaffoldCmd.Flags().Int("phases", 0, "Custom number of phases (overrides mode)")
	planScaffoldCmd.Flags().String("format", "json", "Output format: json or text")
	planCmd.AddCommand(planScaffoldCmd)
}
