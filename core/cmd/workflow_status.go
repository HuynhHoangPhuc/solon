package cmd

import (
	"encoding/json"
	"fmt"

	"solon-core/internal/workflow"

	"github.com/spf13/cobra"
)

var workflowStatusCmd = &cobra.Command{
	Use:   "status <plan-dir>",
	Short: "Show workflow completion status for a plan",
	Long:  "Count completed, in-progress, and pending phases and calculate overall progress.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planDir := args[0]
		result, err := workflow.GetStatus(planDir)
		if err != nil {
			return fmt.Errorf("status: %w", err)
		}

		format, _ := cmd.Flags().GetString("format")
		if format == "text" {
			fmt.Printf("Plan: %s\nStatus: %s\nProgress: %d%%\n",
				result.PlanDir, result.Status, result.Progress)
			fmt.Printf("Phases: %d total, %d completed, %d in-progress, %d pending\n",
				result.Phases.Total, result.Phases.Completed,
				result.Phases.InProgress, result.Phases.Pending)
			fmt.Printf("Reports: %d\n", result.Reports)
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
	workflowStatusCmd.Flags().String("format", "json", "Output format: json or text")
	workflowCmd.AddCommand(workflowStatusCmd)
}
