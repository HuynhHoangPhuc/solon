// PostToolUse(Edit/Write/MultiEdit): Track edits and remind to run code-simplifier.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"solon-hooks/internal/config"
	"solon-hooks/internal/git"
	"solon-hooks/internal/hookio"
)

var postEditCmd = &cobra.Command{
	Use:   "post-edit",
	Short: "Handle PostToolUse post-edit simplify reminder",
	RunE:  runPostEdit,
}

const (
	simplifySessionFile = "sl-simplify-session.json"
	editThreshold       = 5
	sessionTTLMs        = 2 * 60 * 60 * 1000  // 2 hours in ms
	reminderCooldownMs  = 10 * 60 * 1000       // 10 minutes in ms
)

type simplifySession struct {
	StartTime     int64    `json:"startTime"`
	EditCount     int      `json:"editCount"`
	ModifiedFiles []string `json:"modifiedFiles"`
	LastReminder  int64    `json:"lastReminder"`
	SimplifierRun bool     `json:"simplifierRun"`
}

func newSimplifySession() *simplifySession {
	return &simplifySession{
		StartTime:     time.Now().UnixMilli(),
		ModifiedFiles: []string{},
	}
}

func loadSimplifySession() *simplifySession {
	path := filepath.Join(os.TempDir(), simplifySessionFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return newSimplifySession()
	}
	var s simplifySession
	if err := json.Unmarshal(data, &s); err != nil {
		return newSimplifySession()
	}
	// Reset if older than 2 hours
	if time.Now().UnixMilli()-s.StartTime > sessionTTLMs {
		return newSimplifySession()
	}
	return &s
}

func saveSimplifySession(s *simplifySession) {
	path := filepath.Join(os.TempDir(), simplifySessionFile)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0644)
}

func runPostEdit(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("post-edit-simplify-reminder") {
		os.Exit(0)
	}

	var input hookio.PostToolUseInput
	if err := hookio.ReadInput(&input); err != nil {
		hookio.WriteOutput(map[string]interface{}{"continue": true})
		return nil
	}

	editTools := map[string]bool{"Edit": true, "Write": true, "MultiEdit": true}
	if !editTools[input.ToolName] {
		hookio.WriteOutput(map[string]interface{}{"continue": true})
		return nil
	}

	cwd := input.CWD
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	git.InvalidateCache(cwd)

	s := loadSimplifySession()
	s.EditCount++

	// Extract file path from tool input
	filePath := ""
	if fp, ok := input.ToolInput["file_path"].(string); ok && fp != "" {
		filePath = fp
	} else if p, ok := input.ToolInput["path"].(string); ok && p != "" {
		filePath = p
	}
	if filePath != "" {
		found := false
		for _, f := range s.ModifiedFiles {
			if f == filePath {
				found = true
				break
			}
		}
		if !found {
			s.ModifiedFiles = append(s.ModifiedFiles, filePath)
		}
	}

	now := time.Now().UnixMilli()
	shouldRemind := s.EditCount >= editThreshold &&
		!s.SimplifierRun &&
		now-s.LastReminder > reminderCooldownMs

	output := map[string]interface{}{"continue": true}
	if shouldRemind {
		s.LastReminder = now
		output["additionalContext"] = fmt.Sprintf(
			"\n\n[Code Simplification Reminder] You have modified %d files in this session. "+
				"Consider using the `code-simplifier` agent to refine recent changes before proceeding to code review. "+
				"This is a MANDATORY step in the workflow.",
			len(s.ModifiedFiles),
		)
	}

	saveSimplifySession(s)
	hookio.WriteOutput(output)
	return nil
}
