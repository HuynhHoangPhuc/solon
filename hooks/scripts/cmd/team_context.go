package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"solon-hooks/internal/config"
	"solon-hooks/internal/hookio"

	"github.com/spf13/cobra"
)

var teamContextCmd = &cobra.Command{
	Use:   "team-context",
	Short: "Handle SubagentStart team context injection",
	RunE:  runTeamContext,
}

type teamConfigJSON struct {
	Name    string `json:"name"`
	Members []struct {
		AgentID   string `json:"agentId"`
		Name      string `json:"name"`
		AgentType string `json:"agentType"`
	} `json:"members"`
}

type taskFileJSON struct {
	Status string `json:"status"`
}

type taskSummary struct {
	Pending    int
	InProgress int
	Completed  int
	Total      int
}

func runTeamContext(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("team-context-inject") {
		os.Exit(0)
	}

	var payload hookio.SubagentStartInput
	if err := json.NewDecoder(os.Stdin).Decode(&payload); err != nil {
		os.Exit(0)
	}

	agentID := payload.AgentID
	teamName := extractTeamName(agentID)
	if teamName == "" {
		os.Exit(0) // not a team agent
	}

	home, _ := os.UserHomeDir()
	teamsDir := filepath.Join(home, ".claude", "teams")
	configPath := filepath.Join(teamsDir, teamName, "config.json")

	teamCfg, err := readTeamConfig(configPath)
	if err != nil {
		os.Exit(0)
	}

	peerList := buildTeamPeerList(teamCfg, agentID)
	tasks := summarizeTeamTasks(home, teamName)

	var lines []string
	lines = append(lines, "## Team Context")
	displayName := teamCfg.Name
	if displayName == "" {
		displayName = teamName
	}
	lines = append(lines, fmt.Sprintf("Team: %s", displayName))
	lines = append(lines, fmt.Sprintf("Your peers: %s", peerList))
	if tasks != nil {
		lines = append(lines, fmt.Sprintf("Task summary: %d pending, %d in progress, %d completed",
			tasks.Pending, tasks.InProgress, tasks.Completed))
	}

	slCtx := buildSLContext()
	// > 1 means env vars beyond the always-present commit convention line
	if len(slCtx) > 1 {
		lines = append(lines, "")
		lines = append(lines, "## CK Context")
		lines = append(lines, slCtx...)
	}

	lines = append(lines, "")
	lines = append(lines, "Remember: Check TaskList, claim tasks, respect file ownership, use SendMessage to communicate.")

	output := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     "SubagentStart",
			"additionalContext": strings.Join(lines, "\n"),
		},
	}
	return json.NewEncoder(os.Stdout).Encode(output)
}

// extractTeamName extracts team name from "name@team-name" format.
// Rejects path traversal: no "..", "/", or "\" in team name.
func extractTeamName(agentID string) string {
	idx := strings.Index(agentID, "@")
	if idx < 1 {
		return ""
	}
	name := agentID[idx+1:]
	if strings.Contains(name, "..") || strings.ContainsAny(name, "/\\") {
		return ""
	}
	return name
}

func readTeamConfig(path string) (*teamConfigJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg teamConfigJSON
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func buildTeamPeerList(cfg *teamConfigJSON, currentAgentID string) string {
	if len(cfg.Members) == 0 {
		return "none"
	}
	var peers []string
	for _, m := range cfg.Members {
		if m.AgentID != currentAgentID {
			peers = append(peers, fmt.Sprintf("%s (%s)", m.Name, m.AgentType))
		}
	}
	if len(peers) == 0 {
		return "none"
	}
	return strings.Join(peers, ", ")
}

func summarizeTeamTasks(home, teamName string) *taskSummary {
	taskDir := filepath.Join(home, ".claude", "tasks", teamName)
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		return nil
	}
	var s taskSummary
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(taskDir, e.Name()))
		if err != nil {
			continue
		}
		var t taskFileJSON
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}
		s.Total++
		switch t.Status {
		case "pending":
			s.Pending++
		case "in_progress":
			s.InProgress++
		case "completed":
			s.Completed++
		}
	}
	return &s
}

func buildSLContext() []string {
	var ctx []string
	env := os.Environ()
	envMap := make(map[string]string, len(env))
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	if v := envMap["SL_REPORTS_PATH"]; v != "" {
		ctx = append(ctx, fmt.Sprintf("Reports: %s", v))
	}
	if v := envMap["SL_PLANS_PATH"]; v != "" {
		ctx = append(ctx, fmt.Sprintf("Plans: %s", v))
	}
	if v := envMap["SL_PROJECT_ROOT"]; v != "" {
		ctx = append(ctx, fmt.Sprintf("Project: %s", v))
	}
	if v := envMap["SL_NAME_PATTERN"]; v != "" {
		ctx = append(ctx, fmt.Sprintf("Naming: %s", v))
	}
	if v := envMap["SL_GIT_BRANCH"]; v != "" {
		ctx = append(ctx, fmt.Sprintf("Branch: %s", v))
	}
	if v := envMap["SL_ACTIVE_PLAN"]; v != "" {
		ctx = append(ctx, fmt.Sprintf("Active plan: %s", v))
	}
	ctx = append(ctx, "Commits: conventional (feat:, fix:, docs:, refactor:, test:, chore:)")
	return ctx
}
