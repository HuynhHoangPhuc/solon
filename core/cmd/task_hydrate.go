package cmd

import (
	"encoding/json"
	"fmt"

	"solon-core/internal/task"

	"github.com/spf13/cobra"
)

var taskHydrateCmd = &cobra.Command{
	Use:   "hydrate <plan-dir>",
	Short: "Extract tasks from phase files in a plan directory",
	Long:  "Scan phase-*.md files and produce structured task definitions with blocking relationships.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planDir := args[0]
		result, err := task.HydratePlan(planDir)
		if err != nil {
			return fmt.Errorf("hydrate: %w", err)
		}

		format, _ := cmd.Flags().GetString("format")
		if format == "text" {
			if result.Skipped {
				fmt.Printf("Skipped: %s\n", result.SkipReason)
				return nil
			}
			fmt.Printf("Plan: %s\nTasks: %d\n\n", result.PlanDir, result.TaskCount)
			for _, t := range result.Tasks {
				fmt.Printf("  [%d] %s (priority: %s, effort: %s, todos: %d)\n",
					t.Phase, t.Title, t.Priority, t.Effort, t.TodoCount)
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
	taskHydrateCmd.Flags().String("format", "json", "Output format: json or text")
	taskCmd.AddCommand(taskHydrateCmd)
}
