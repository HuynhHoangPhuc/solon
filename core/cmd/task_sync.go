package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"solon-core/internal/task"

	"github.com/spf13/cobra"
)

var taskSyncCmd = &cobra.Command{
	Use:   "sync <plan-dir>",
	Short: "Mark completed phase TODO items as done",
	Long:  "Replace all '- [ ]' with '- [x]' in specified completed phase files.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planDir := args[0]
		completedStr, _ := cmd.Flags().GetString("completed")

		var completedPhases []int
		for _, s := range strings.Split(completedStr, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			n, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("invalid phase number %q: %w", s, err)
			}
			completedPhases = append(completedPhases, n)
		}

		result, err := task.SyncCompletions(planDir, completedPhases)
		if err != nil {
			return fmt.Errorf("sync: %w", err)
		}

		format, _ := cmd.Flags().GetString("format")
		if format == "text" {
			fmt.Printf("Plan: %s\nFiles modified: %d\nCheckboxes updated: %d\n",
				result.PlanDir, len(result.FilesModified), result.CheckboxesUpdated)
			for _, d := range result.Details {
				fmt.Printf("  %s: %d updated\n", d.File, d.Updated)
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
	taskSyncCmd.Flags().String("completed", "", "Comma-separated list of completed phase numbers (e.g. 1,2,3)")
	taskSyncCmd.Flags().String("format", "json", "Output format: json or text")
	taskCmd.AddCommand(taskSyncCmd)
}
