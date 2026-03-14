// TeammateIdle: Inject available task context when teammate goes idle.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"solon-hooks/internal/config"
	"solon-hooks/internal/hookio"
)

var teammateIdleCmd = &cobra.Command{
	Use:   "teammate-idle",
	Short: "Handle TeammateIdle hook",
	RunE:  runTeammateIdle,
}

// idleTaskFile is the shape of a task file including blocking/ownership fields.
type idleTaskFile struct {
	ID        string   `json:"id"`
	Status    string   `json:"status"`
	Subject   string   `json:"subject"`
	BlockedBy []string `json:"blockedBy"`
	Owner     string   `json:"owner"`
}

type availableTaskInfo struct {
	Pending    int
	InProgress int
	Completed  int
	Total      int
	Unblocked  []struct{ ID, Subject string }
}

// getAvailableTasks reads all task files for a team and computes unblocked tasks.
func getAvailableTasks(teamName string) *availableTaskInfo {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	taskDir := filepath.Join(home, ".claude", "tasks", teamName)
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		return nil
	}

	var tasks []idleTaskFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(taskDir, e.Name()))
		if err != nil {
			continue
		}
		var t idleTaskFile
		if err := json.Unmarshal(data, &t); err != nil || t.Status == "" {
			continue
		}
		tasks = append(tasks, t)
	}

	// Build completed ID set
	completedIDs := make(map[string]bool)
	for _, t := range tasks {
		if t.Status == "completed" {
			completedIDs[t.ID] = true
		}
	}

	info := &availableTaskInfo{}
	for _, t := range tasks {
		switch t.Status {
		case "completed":
			info.Completed++
			continue
		case "in_progress":
			info.InProgress++
			continue
		case "pending":
			info.Pending++
		default:
			continue
		}
		// Check if unblocked and unowned
		allDepsComplete := true
		for _, depID := range t.BlockedBy {
			if !completedIDs[depID] {
				allDepsComplete = false
				break
			}
		}
		if allDepsComplete && t.Owner == "" {
			info.Unblocked = append(info.Unblocked, struct{ ID, Subject string }{t.ID, t.Subject})
		}
	}
	info.Total = info.Pending + info.InProgress + info.Completed
	return info
}

func runTeammateIdle(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("teammate-idle-handler") {
		os.Exit(0)
	}

	defer func() {
		if r := recover(); r != nil {
			os.Exit(0)
		}
	}()

	var payload hookio.TeammateIdleInput
	if err := hookio.ReadInput(&payload); err != nil {
		os.Exit(0)
	}

	if payload.TeamName == "" {
		os.Exit(0)
	}

	info := getAvailableTasks(payload.TeamName)
	lines := []string{
		"## Teammate Idle",
		fmt.Sprintf("%s is idle.", payload.TeammateName),
	}

	if info != nil {
		remaining := info.Pending + info.InProgress
		lines = append(lines, fmt.Sprintf("Tasks: %d/%d done. %d remaining.", info.Completed, info.Total, remaining))

		if len(info.Unblocked) > 0 {
			parts := make([]string, len(info.Unblocked))
			for i, t := range info.Unblocked {
				parts[i] = fmt.Sprintf("#%s \"%s\"", t.ID, t.Subject)
			}
			lines = append(lines, "Unblocked & unassigned: "+strings.Join(parts, ", "))
			lines = append(lines, fmt.Sprintf("Consider assigning work to %s or waking them with a message.", payload.TeammateName))
		} else if remaining == 0 {
			lines = append(lines, fmt.Sprintf("No remaining tasks. Consider shutting down %s.", payload.TeammateName))
		} else {
			lines = append(lines, fmt.Sprintf("All remaining tasks are blocked or assigned. %s may be waiting for dependencies.", payload.TeammateName))
		}
	}

	hookio.WriteOutput(map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     "TeammateIdle",
			"additionalContext": strings.Join(lines, "\n"),
		},
	})
	return nil
}
