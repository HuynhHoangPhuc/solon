package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags "-X solon-hooks/cmd.Version=v0.2.0"
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of solon-hooks",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}
