package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"solon-core/internal/template"

	"github.com/spf13/cobra"
)

var planRedTeamCmd = &cobra.Command{
	Use:   "red-team <plan-dir>",
	Short: "Generate red-team review prompt",
	Long:  "Output a structured red-team review prompt with 4 personas for a plan.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planDir := args[0]

		// Collect phase files
		entries, err := os.ReadDir(planDir)
		if err != nil {
			return fmt.Errorf("read plan dir: %w", err)
		}
		var phases []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasPrefix(e.Name(), "phase-") && strings.HasSuffix(e.Name(), ".md") {
				phases = append(phases, filepath.Join(planDir, e.Name()))
			}
		}

		prompt := template.RenderRedTeam(planDir, phases)
		fmt.Print(prompt)
		return nil
	},
}

func init() {
	planCmd.AddCommand(planRedTeamCmd)
}
