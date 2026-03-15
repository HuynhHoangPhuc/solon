package cmd

import "github.com/spf13/cobra"

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Report management commands",
	Long:  "Index and manage plan reports.",
}

func init() {
	rootCmd.AddCommand(reportCmd)
}
