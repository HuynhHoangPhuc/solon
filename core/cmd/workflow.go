package cmd

import "github.com/spf13/cobra"

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Workflow management commands",
	Long:  "Inspect and manage plan workflow status.",
}

func init() {
	rootCmd.AddCommand(workflowCmd)
}
