// TaskCompleted: Log task completion and inject progress context.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"solon-hooks/internal/config"
	"solon-hooks/internal/hookio"
)

var taskCompletedCmd = &cobra.Command{
	Use:   "task-completed",
	Short: "Handle TaskCompleted hook",
	RunE:  runTaskCompleted,
}

// taskFile is the minimal shape of a ~/.claude/tasks/{team}/*.json file.
type taskFile struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// taskCounts holds task status tallies for a team.
type taskCounts struct {
	Pending    int
	InProgress int
	Completed  int
	Total      int
}

// countTeamTasks reads task JSON files and tallies status counts.
func countTeamTasks(teamName string) *taskCounts {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	taskDir := filepath.Join(home, ".claude", "tasks", teamName)
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		return nil
	}
	counts := &taskCounts{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(taskDir, e.Name()))
		if err != nil {
			continue
		}
		var t taskFile
		if err := json.Unmarshal(data, &t); err != nil || t.Status == "" {
			continue
		}
		switch t.Status {
		case "pending":
			counts.Pending++
		case "in_progress":
			counts.InProgress++
		case "completed":
			counts.Completed++
		}
	}
	counts.Total = counts.Pending + counts.InProgress + counts.Completed
	return counts
}

// logTaskCompletion appends a completion line to the team reports file.
func logTaskCompletion(teamName, taskID, taskSubject, teammateName string) {
	reportsPath := os.Getenv("SL_REPORTS_PATH")
	if reportsPath == "" {
		return
	}
	logFile := filepath.Join(reportsPath, fmt.Sprintf("team-%s-completions.md", teamName))
	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("- [%s] Task #%s \"%s\" completed by %s\n", timestamp, taskID, taskSubject, teammateName)
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line)
}

func runTaskCompleted(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("task-completed-handler") {
		os.Exit(0)
	}

	defer func() {
		if r := recover(); r != nil {
			os.Exit(0)
		}
	}()

	var payload hookio.TaskCompletedInput
	if err := hookio.ReadInput(&payload); err != nil {
		os.Exit(0)
	}

	if payload.TeamName == "" {
		os.Exit(0)
	}

	logTaskCompletion(payload.TeamName, payload.TaskID, payload.TaskSubject, payload.TeammateName)

	counts := countTeamTasks(payload.TeamName)
	lines := []string{
		"## Task Completed",
		fmt.Sprintf("Task #%s \"%s\" completed by %s.", payload.TaskID, payload.TaskSubject, payload.TeammateName),
	}

	if counts != nil {
		remaining := counts.Pending + counts.InProgress
		lines = append(lines, fmt.Sprintf(
			"Progress: %d/%d done. %d pending, %d in progress.",
			counts.Completed, counts.Total, counts.Pending, counts.InProgress,
		))
		if remaining == 0 {
			lines = append(lines, "")
			lines = append(lines, "**All tasks completed.** Consider shutting down teammates and synthesizing results.")
		}
	}

	hookio.WriteOutput(map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     "TaskCompleted",
			"additionalContext": strings.Join(lines, "\n"),
		},
	})
	return nil
}
