package cmd

import (
	"encoding/json"
	"fmt"

	"solon-core/internal/plan"

	"github.com/spf13/cobra"
)

var planValidateCmd = &cobra.Command{
	Use:   "validate <plan-dir>",
	Short: "Check plan completeness",
	Long:  "Validate a plan directory has required files, frontmatter, and TODO sections.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planDir := args[0]
		result := plan.ValidatePlan(planDir)

		format, _ := cmd.Flags().GetString("format")
		if format == "text" {
			if result.Valid {
				fmt.Printf("✓ Plan valid: %s (%d phases, %d/%d TODOs)\n",
					planDir, result.Stats.PhaseCount, result.Stats.TodoCompleted, result.Stats.TodoTotal)
			} else {
				fmt.Printf("✗ Plan invalid: %s\n", planDir)
				for _, e := range result.Errors {
					fmt.Printf("  ERROR: %s\n", e)
				}
			}
			for _, w := range result.Warnings {
				fmt.Printf("  WARN: %s\n", w)
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
	planValidateCmd.Flags().String("format", "json", "Output format: json or text")
	planCmd.AddCommand(planValidateCmd)
}
