package cmd

import (
	"encoding/json"
	"fmt"

	"solon-core/internal/report"

	"github.com/spf13/cobra"
)

var reportIndexCmd = &cobra.Command{
	Use:   "index <plan-dir>",
	Short: "List all report files in a plan directory",
	Long:  "Scan reports/ and research/ subdirectories and return indexed report entries.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planDir := args[0]
		entries, err := report.IndexReports(planDir)
		if err != nil {
			return fmt.Errorf("index: %w", err)
		}

		format, _ := cmd.Flags().GetString("format")
		if format == "text" {
			if len(entries) == 0 {
				fmt.Println("No reports found.")
				return nil
			}
			for _, e := range entries {
				fmt.Printf("[%s] %s\n", e.Directory, e.Filename)
			}
			return nil
		}

		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal result: %w", err)
		}
		fmt.Println(string(data))
		return nil
	},
}

func init() {
	reportIndexCmd.Flags().String("format", "json", "Output format: json or text")
	reportCmd.AddCommand(reportIndexCmd)
}
