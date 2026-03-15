package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"solon-core/internal/config"
	"solon-core/internal/plan"

	"github.com/spf13/cobra"
)

var planResolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Resolve active plan path",
	Long:  "Detect the active plan using session state or branch name matching.",
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := os.Getenv("SL_SESSION_ID")
		cfg := config.LoadConfig()
		resolved := plan.ResolvePlanPath(sessionID, &cfg)
		result := plan.EnrichResolveResult(resolved)

		format, _ := cmd.Flags().GetString("format")
		if format == "text" {
			if result.Path == "" {
				fmt.Println("No active plan resolved")
				return nil
			}
			fmt.Printf("Plan: %s\nResolved by: %s\n", result.Path, result.ResolvedBy)
			if result.Status != "" {
				fmt.Printf("Status: %s\n", result.Status)
			}
			if result.Phases > 0 {
				fmt.Printf("Phases: %d\n", result.Phases)
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
	planResolveCmd.Flags().String("format", "json", "Output format: json or text")
	planCmd.AddCommand(planResolveCmd)
}
