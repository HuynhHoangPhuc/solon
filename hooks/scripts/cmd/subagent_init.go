package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"solon-hooks/internal/config"
	"solon-hooks/internal/context"
	slexec "solon-hooks/internal/exec"
	"solon-hooks/internal/hookio"
	"solon-hooks/internal/plan"
	"solon-hooks/internal/wisdom"

	"github.com/spf13/cobra"
)

var subagentInitCmd = &cobra.Command{
	Use:   "subagent-init",
	Short: "Handle SubagentStart hook",
	RunE:  runSubagentInit,
}

func runSubagentInit(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("subagent-init") {
		os.Exit(0)
	}

	var payload hookio.SubagentStartInput
	if err := json.NewDecoder(os.Stdin).Decode(&payload); err != nil || payload.SessionID == "" && payload.AgentType == "" {
		// try partial decode - empty stdin is ok, just exit
		if payload.AgentType == "" && payload.AgentID == "" {
			os.Exit(0)
		}
	}

	agentType := payload.AgentType
	if agentType == "" {
		agentType = "unknown"
	}
	agentID := payload.AgentID
	if agentID == "" {
		agentID = "unknown"
	}

	cfg := config.LoadConfig(config.LoadConfigOptions{
		IncludeProject:    false,
		IncludeAssertions: false,
		IncludeLocale:     true,
	})

	effectiveCwd := strings.TrimSpace(payload.CWD)
	if effectiveCwd == "" {
		effectiveCwd, _ = os.Getwd()
	}

	sessionID := payload.SessionID
	if sessionID == "" {
		sessionID = os.Getenv("SL_SESSION_ID")
	}

	resolved := plan.ResolvePlanPath(sessionID, &cfg)
	reportsPath := plan.GetReportsPath(resolved.Path, resolved.ResolvedBy, cfg.Plan, cfg.Paths, effectiveCwd)
	taskListID := plan.ExtractTaskListID(resolved)

	plansPath := filepath.Join(effectiveCwd, plan.NormalizePath(cfg.Paths.Plans))
	if plan.NormalizePath(cfg.Paths.Plans) == "" {
		plansPath = filepath.Join(effectiveCwd, "plans")
	}
	docsPath := filepath.Join(effectiveCwd, plan.NormalizePath(cfg.Paths.Docs))
	if plan.NormalizePath(cfg.Paths.Docs) == "" {
		docsPath = filepath.Join(effectiveCwd, "docs")
	}

	namePattern := plan.ResolveNamingPattern(cfg.Plan, "")

	activePlan := ""
	suggestedPlan := ""
	if resolved.ResolvedBy == "session" {
		activePlan = resolved.Path
	} else if resolved.ResolvedBy == "branch" {
		suggestedPlan = resolved.Path
	}

	skillsVenv := context.ResolveSkillsVenv("")

	var thinkingLang, responseLang string
	if cfg.Locale.ThinkingLanguage != nil {
		thinkingLang = *cfg.Locale.ThinkingLanguage
	}
	if cfg.Locale.ResponseLanguage != nil {
		responseLang = *cfg.Locale.ResponseLanguage
	}
	effectiveThinking := thinkingLang
	if effectiveThinking == "" && responseLang != "" {
		effectiveThinking = "en"
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("## Subagent: %s", agentType))
	lines = append(lines, fmt.Sprintf("ID: %s | CWD: %s", agentID, effectiveCwd))
	lines = append(lines, "")

	lines = append(lines, "## Context")
	if activePlan != "" {
		lines = append(lines, fmt.Sprintf("- Plan: %s", activePlan))
		if taskListID != "" {
			lines = append(lines, fmt.Sprintf("- Task List: %s (shared with session)", taskListID))
		}
	} else if suggestedPlan != "" {
		lines = append(lines, fmt.Sprintf("- Plan: none | Suggested: %s", suggestedPlan))
	} else {
		lines = append(lines, "- Plan: none")
	}
	lines = append(lines, fmt.Sprintf("- Reports: %s", reportsPath))
	lines = append(lines, fmt.Sprintf("- Paths: %s/ | %s/", plansPath, docsPath))

	// Inject workflow progress from sc binary if available
	if activePlan != "" {
		if progress := fetchWorkflowProgress(activePlan); progress != "" {
			lines = append(lines, fmt.Sprintf("- %s", progress))
		}
	}

	lines = append(lines, "")

	hasThinking := effectiveThinking != "" && effectiveThinking != responseLang
	if hasThinking || responseLang != "" {
		lines = append(lines, "## Language")
		if hasThinking {
			lines = append(lines, fmt.Sprintf("- Thinking: Use %s for reasoning (logic, precision).", effectiveThinking))
		}
		if responseLang != "" {
			lines = append(lines, fmt.Sprintf("- Response: Respond in %s (natural, fluent).", responseLang))
		}
		lines = append(lines, "")
	}

	lines = append(lines, "## Rules")
	lines = append(lines, fmt.Sprintf("- Reports → %s", reportsPath))
	lines = append(lines, "- YAGNI / KISS / DRY")
	lines = append(lines, "- Concise, list unresolved Qs at end")
	if skillsVenv != "" {
		lines = append(lines, fmt.Sprintf("- Python scripts in .claude/skills/: Use `%s`", skillsVenv))
		lines = append(lines, "- Never use global pip install")
	}

	lines = append(lines, "")
	lines = append(lines, "## Naming")
	lines = append(lines, fmt.Sprintf("- Report: %s", filepath.Join(reportsPath, agentType+"-"+namePattern+".md")))
	lines = append(lines, fmt.Sprintf("- Plan dir: %s/", filepath.Join(plansPath, namePattern)))

	if cfg.Trust.Enabled && cfg.Trust.Passphrase != nil && *cfg.Trust.Passphrase != "" {
		lines = append(lines, "")
		lines = append(lines, "## Trust Verification")
		lines = append(lines, fmt.Sprintf("Passphrase: \"%s\"", *cfg.Trust.Passphrase))
	}

	// Inject prior learnings from wisdom file
	wisdomContent := wisdom.ReadWisdom(activePlan, sessionID, 15)
	if wisdomContent != "" {
		lines = append(lines, "")
		lines = append(lines, "## Prior Learnings")
		lines = append(lines, wisdomContent)
	}

	if cfg.Subagent != nil {
		if agentCfg, ok := cfg.Subagent.Agents[agentType]; ok && agentCfg.ContextPrefix != "" {
			lines = append(lines, "")
			lines = append(lines, "## Agent Instructions")
			lines = append(lines, agentCfg.ContextPrefix)
		}
	}

	output := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     "SubagentStart",
			"additionalContext": strings.Join(lines, "\n"),
		},
	}
	return json.NewEncoder(os.Stdout).Encode(output)
}

// fetchWorkflowProgress calls `sc workflow status <planDir>` and returns a
// brief progress string. Returns "" if sc is not found or the call fails.
func fetchWorkflowProgress(planDir string) string {
	scPath := findSCBinary()
	if scPath == "" {
		return ""
	}
	output := slexec.ExecFileSafe(scPath, []string{"workflow", "status", planDir}, 5000)
	if output == "" {
		return ""
	}
	var result struct {
		Progress int `json:"progress"`
		Phases   struct {
			Total     int `json:"total"`
			Completed int `json:"completed"`
		} `json:"phases"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return ""
	}
	if result.Phases.Total == 0 {
		return ""
	}
	return fmt.Sprintf("Plan progress: %d%% (%d/%d phases complete)",
		result.Progress, result.Phases.Completed, result.Phases.Total)
}
