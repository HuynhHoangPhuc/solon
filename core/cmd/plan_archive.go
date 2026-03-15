package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var planArchiveCmd = &cobra.Command{
	Use:   "archive <plan-dir>",
	Short: "Mark plan as archived",
	Long:  "Update plan.md frontmatter status to 'archived'.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planDir := args[0]
		planFile := filepath.Join(planDir, "plan.md")

		data, err := os.ReadFile(planFile)
		if err != nil {
			return fmt.Errorf("read plan.md: %w", err)
		}

		content := string(data)
		updated := updateFrontmatterStatus(content, "archived")
		if updated == content {
			fmt.Println("No status field found in frontmatter; no changes made")
			return nil
		}

		if err := os.WriteFile(planFile, []byte(updated), 0644); err != nil {
			return fmt.Errorf("write plan.md: %w", err)
		}
		fmt.Printf("Archived: %s\n", planFile)
		return nil
	},
}

// updateFrontmatterStatus replaces status: <value> in YAML frontmatter.
func updateFrontmatterStatus(content, newStatus string) string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if inFrontmatter {
				break
			}
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.HasPrefix(trimmed, "status:") {
			lines[i] = "status: " + newStatus
			return strings.Join(lines, "\n")
		}
	}
	return content
}

func init() {
	planCmd.AddCommand(planArchiveCmd)
}
