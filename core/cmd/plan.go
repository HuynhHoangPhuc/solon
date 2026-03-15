package cmd

import "github.com/spf13/cobra"

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Plan management commands",
	Long:  "Resolve, scaffold, validate, archive, and red-team review plans.",
}

func init() {
	rootCmd.AddCommand(planCmd)
}
