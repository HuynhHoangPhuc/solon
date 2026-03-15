package cmd

import "github.com/spf13/cobra"

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Task management commands",
	Long:  "Hydrate plan phases into tasks and sync completion state.",
}

func init() {
	rootCmd.AddCommand(taskCmd)
}
