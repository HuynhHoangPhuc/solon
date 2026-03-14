package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "solon-hooks",
	Short: "Solon lifecycle hooks for Claude Code",
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(sessionInitCmd)
	rootCmd.AddCommand(subagentInitCmd)
	rootCmd.AddCommand(teamContextCmd)
	rootCmd.AddCommand(cookReminderCmd)
	rootCmd.AddCommand(devRulesCmd)
	rootCmd.AddCommand(usageAwarenessCmd)
	rootCmd.AddCommand(descriptiveNameCmd)
	rootCmd.AddCommand(scoutBlockCmd)
	rootCmd.AddCommand(privacyBlockCmd)
	rootCmd.AddCommand(postEditCmd)
	rootCmd.AddCommand(notifyCmd)
	rootCmd.AddCommand(taskCompletedCmd)
	rootCmd.AddCommand(teammateIdleCmd)
	rootCmd.AddCommand(statuslineCmd)
	rootCmd.AddCommand(preemptiveCompactionCmd)
	rootCmd.AddCommand(toolOutputTruncationCmd)
	rootCmd.AddCommand(todoEnforcerCmd)
	rootCmd.AddCommand(commentSlopCheckerCmd)
	rootCmd.AddCommand(wisdomAccumulatorCmd)
}
