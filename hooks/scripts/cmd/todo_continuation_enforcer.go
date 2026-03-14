// UserPromptSubmit hook: injects reminder when active plan has incomplete todos.
package cmd

import (
	"bufio"
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

const (
	todoEnforcerCooldownMs   = 10 * 60 * 1000 // 10 minutes
	todoEnforcerCooldownFile = "sl-todo-enforcer-cooldown.json"
)

var todoEnforcerCmd = &cobra.Command{
	Use:   "todo-enforcer",
	Short: "Remind about incomplete plan todos on user prompt",
	RunE:  runTodoEnforcer,
}

func runTodoEnforcer(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("todo-continuation-enforcer") {
		os.Exit(0)
	}

	var input hookio.UserPromptSubmitInput
	if err := hookio.ReadInput(&input); err != nil {
		os.Exit(0)
	}

	planPath := os.Getenv("SL_ACTIVE_PLAN")
	if planPath == "" {
		os.Exit(0)
	}

	if !todoEnforcerCooldownExpired() {
		os.Exit(0)
	}

	phaseFiles, _ := filepath.Glob(filepath.Join(planPath, "phase-*.md"))
	if len(phaseFiles) == 0 {
		os.Exit(0)
	}

	totalIncomplete := 0
	filesWithTodos := 0
	for _, f := range phaseFiles {
		count := countIncompleteTodos(f)
		if count > 0 {
			totalIncomplete += count
			filesWithTodos++
		}
	}

	if totalIncomplete == 0 {
		os.Exit(0)
	}

	writeTodoEnforcerCooldown()
	hookio.WriteContext(fmt.Sprintf(
		"\n[Plan Progress] %d incomplete todo(s) across %d phase file(s). "+
			"Continue working through the plan before considering the task done.\n",
		totalIncomplete, filesWithTodos,
	))
	return nil
}

// countIncompleteTodos counts `- [ ]` lines in a markdown file.
func countIncompleteTodos(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.HasPrefix(strings.TrimSpace(scanner.Text()), "- [ ]") {
			count++
		}
	}
	return count
}

type todoEnforcerCooldownState struct {
	Timestamp int64 `json:"timestamp"`
}

func todoEnforcerCooldownExpired() bool {
	path := filepath.Join(os.TempDir(), todoEnforcerCooldownFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	var state todoEnforcerCooldownState
	if err := json.Unmarshal(data, &state); err != nil {
		return true
	}
	return time.Now().UnixMilli()-state.Timestamp > todoEnforcerCooldownMs
}

func writeTodoEnforcerCooldown() {
	path := filepath.Join(os.TempDir(), todoEnforcerCooldownFile)
	data, _ := json.Marshal(todoEnforcerCooldownState{Timestamp: time.Now().UnixMilli()})
	_ = os.WriteFile(path, data, 0644)
}
