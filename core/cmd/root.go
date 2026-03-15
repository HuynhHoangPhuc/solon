package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "sc",
	Short: "Solon orchestration engine",
	Long:  "sc — development workflow orchestration for Claude Code projects.",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print sc version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("sc %s\n", Version)
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
