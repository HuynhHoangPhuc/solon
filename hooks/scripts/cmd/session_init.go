package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"solon-hooks/internal/config"
	"solon-hooks/internal/hookio"
	"solon-hooks/internal/plan"
	"solon-hooks/internal/project"
	"solon-hooks/internal/session"

	"github.com/spf13/cobra"
)

var sessionInitCmd = &cobra.Command{
	Use:   "session-init",
	Short: "Handle SessionStart hook",
	RunE:  runSessionInit,
}

func runSessionInit(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("session-init") {
		os.Exit(0)
	}

	shadowedCleanup := cleanupOrphanedShadowedSkills()

	var data hookio.SessionStartInput
	if err := hookio.ReadInput(&data); err != nil {
		fmt.Fprintf(os.Stderr, "[session-init] Error: %s\n", err.Error())
		os.Exit(0)
	}

	envFile := os.Getenv("CLAUDE_ENV_FILE")
	source := data.Source
	if source == "" {
		source = "unknown"
	}
	sessionID := data.SessionID

	cfg := config.LoadConfig(config.LoadConfigOptions{
		IncludeProject:    true,
		IncludeAssertions: true,
		IncludeLocale:     true,
	})

	projectType := project.DetectProjectType(cfg.Project.Type)
	packageManager := project.DetectPackageManager(cfg.Project.PackageManager)
	framework := project.DetectFramework(cfg.Project.Framework)

	resolved := plan.ResolvePlanPath(sessionID, &cfg)

	if sessionID != "" {
		activePlan := resolved.Path
		if resolved.ResolvedBy != "session" {
			activePlan = ""
		}
		suggestedPlan := resolved.Path
		if resolved.ResolvedBy != "branch" {
			suggestedPlan = ""
		}
		var activePlanPtr, suggestedPlanPtr *string
		if activePlan != "" {
			activePlanPtr = &activePlan
		}
		if suggestedPlan != "" {
			suggestedPlanPtr = &suggestedPlan
		}
		cwd, _ := os.Getwd()
		session.WriteSessionState(sessionID, &hookio.SessionState{
			SessionOrigin: cwd,
			ActivePlan:    activePlanPtr,
			SuggestedPlan: suggestedPlanPtr,
			Timestamp:     unixMillis(),
			Source:        source,
		})
	}

	baseDir, _ := os.Getwd()
	reportsPath := plan.GetReportsPath(resolved.Path, resolved.ResolvedBy, cfg.Plan, cfg.Paths, baseDir)
	taskListID := plan.ExtractTaskListID(resolved)
	gitBranch := project.GetGitBranch()
	namePattern := plan.ResolveNamingPattern(cfg.Plan, gitBranch)

	home, _ := os.UserHomeDir()
	user := firstOf(os.Getenv("USERNAME"), os.Getenv("USER"), os.Getenv("LOGNAME"))
	nodeVersion := detectNodeVersion()
	pythonVersion := project.GetPythonVersion()
	gitRoot := project.GetGitRoot()
	gitURL := project.GetGitRemoteURL()
	timezone := localTZ()

	writeSessionEnvVars(envFile, writeEnvArgs{
		cfg:            cfg,
		sessionID:      sessionID,
		namePattern:    namePattern,
		resolved:       resolved,
		taskListID:     taskListID,
		reportsPath:    reportsPath,
		baseDir:        baseDir,
		projectType:    projectType,
		packageManager: packageManager,
		framework:      framework,
		nodeVersion:    nodeVersion,
		pythonVersion:  pythonVersion,
		gitRoot:        gitRoot,
		gitURL:         gitURL,
		gitBranch:      gitBranch,
		user:           user,
		timezone:       timezone,
		home:           home,
	})

	team := detectAgentTeam()
	if envFile != "" && team != nil {
		config.WriteEnv(envFile, "SL_AGENT_TEAM", team.TeamName)
		config.WriteEnv(envFile, "SL_AGENT_TEAM_MEMBERS", team.MemberCount)
	}

	writeSessionContext(sessionContextArgs{
		source:         source,
		projectType:    projectType,
		packageManager: packageManager,
		cfg:            cfg,
		gitRoot:        gitRoot,
		baseDir:        baseDir,
		resolved:       resolved,
		shadowedCleanup: shadowedCleanup,
		team:           team,
		codingLevel:    cfg.CodingLevel,
	})

	return nil
}

type writeEnvArgs struct {
	cfg            hookio.SLConfig
	sessionID      string
	namePattern    string
	resolved       hookio.PlanResolution
	taskListID     string
	reportsPath    string
	baseDir        string
	projectType    string
	packageManager string
	framework      string
	nodeVersion    string
	pythonVersion  string
	gitRoot        string
	gitURL         string
	gitBranch      string
	user           string
	timezone       string
	home           string
}

func writeSessionEnvVars(envFile string, a writeEnvArgs) {
	if envFile == "" {
		return
	}
	cfg := a.cfg
	config.WriteEnv(envFile, "SL_SESSION_ID", a.sessionID)
	config.WriteEnv(envFile, "SL_PLAN_NAMING_FORMAT", cfg.Plan.NamingFormat)
	config.WriteEnv(envFile, "SL_PLAN_DATE_FORMAT", cfg.Plan.DateFormat)
	issuePrefix := ""
	if cfg.Plan.IssuePrefix != nil {
		issuePrefix = *cfg.Plan.IssuePrefix
	}
	writeEnvForced(envFile, "SL_PLAN_ISSUE_PREFIX", issuePrefix)
	config.WriteEnv(envFile, "SL_PLAN_REPORTS_DIR", cfg.Plan.ReportsDir)
	config.WriteEnv(envFile, "SL_NAME_PATTERN", a.namePattern)

	activePlan := ""
	if a.resolved.ResolvedBy == "session" {
		activePlan = a.resolved.Path
	}
	writeEnvForced(envFile, "SL_ACTIVE_PLAN", activePlan)

	suggestedPlan := ""
	if a.resolved.ResolvedBy == "branch" {
		suggestedPlan = a.resolved.Path
	}
	writeEnvForced(envFile, "SL_SUGGESTED_PLAN", suggestedPlan)

	if a.taskListID != "" {
		config.WriteEnv(envFile, "CLAUDE_CODE_TASK_LIST_ID", a.taskListID)
	}
	config.WriteEnv(envFile, "SL_GIT_ROOT", a.gitRoot)
	config.WriteEnv(envFile, "SL_REPORTS_PATH", a.reportsPath)
	config.WriteEnv(envFile, "SL_DOCS_PATH", filepath.Join(a.baseDir, cfg.Paths.Docs))
	config.WriteEnv(envFile, "SL_PLANS_PATH", filepath.Join(a.baseDir, cfg.Paths.Plans))
	config.WriteEnv(envFile, "SL_PROJECT_ROOT", a.baseDir)
	config.WriteEnv(envFile, "SL_PROJECT_TYPE", a.projectType)
	writeEnvForced(envFile, "SL_PACKAGE_MANAGER", a.packageManager)
	writeEnvForced(envFile, "SL_FRAMEWORK", a.framework)
	config.WriteEnv(envFile, "SL_NODE_VERSION", a.nodeVersion)
	config.WriteEnv(envFile, "SL_PYTHON_VERSION", a.pythonVersion)
	config.WriteEnv(envFile, "SL_OS_PLATFORM", runtime.GOOS)
	config.WriteEnv(envFile, "SL_GIT_URL", a.gitURL)
	config.WriteEnv(envFile, "SL_GIT_BRANCH", a.gitBranch)
	config.WriteEnv(envFile, "SL_USER", a.user)
	config.WriteEnv(envFile, "SL_LOCALE", os.Getenv("LANG"))
	config.WriteEnv(envFile, "SL_TIMEZONE", a.timezone)
	config.WriteEnv(envFile, "SL_CLAUDE_SETTINGS_DIR", filepath.Join(a.home, ".claude"))

	if cfg.Locale.ThinkingLanguage != nil && *cfg.Locale.ThinkingLanguage != "" {
		config.WriteEnv(envFile, "SL_THINKING_LANGUAGE", *cfg.Locale.ThinkingLanguage)
	}
	if cfg.Locale.ResponseLanguage != nil && *cfg.Locale.ResponseLanguage != "" {
		config.WriteEnv(envFile, "SL_RESPONSE_LANGUAGE", *cfg.Locale.ResponseLanguage)
	}

	v := cfg.Plan.Validation
	if v.Mode == "" {
		v.Mode = "prompt"
	}
	if v.MinQuestions == 0 {
		v.MinQuestions = 3
	}
	if v.MaxQuestions == 0 {
		v.MaxQuestions = 8
	}
	config.WriteEnv(envFile, "SL_VALIDATION_MODE", v.Mode)
	config.WriteEnv(envFile, "SL_VALIDATION_MIN_QUESTIONS", v.MinQuestions)
	config.WriteEnv(envFile, "SL_VALIDATION_MAX_QUESTIONS", v.MaxQuestions)
	config.WriteEnv(envFile, "SL_VALIDATION_FOCUS_AREAS", strings.Join(v.FocusAreas, ","))

	codingLevel := cfg.CodingLevel
	if codingLevel < 0 {
		codingLevel = 5
	}
	config.WriteEnv(envFile, "SL_CODING_LEVEL", codingLevel)
	config.WriteEnv(envFile, "SL_CODING_LEVEL_STYLE", project.GetCodingLevelStyleName(codingLevel))
}

type sessionContextArgs struct {
	source          string
	projectType     string
	packageManager  string
	cfg             hookio.SLConfig
	gitRoot         string
	baseDir         string
	resolved        hookio.PlanResolution
	shadowedCleanup shadowedCleanupResult
	team            *teamInfo
	codingLevel     int
}

func writeSessionContext(a sessionContextArgs) {
	parts := []string{fmt.Sprintf("Project: %s", firstOf(a.projectType, "unknown"))}
	if a.packageManager != "" {
		parts = append(parts, fmt.Sprintf("PM: %s", a.packageManager))
	}
	parts = append(parts, fmt.Sprintf("Plan naming: %s", a.cfg.Plan.NamingFormat))
	if a.gitRoot != "" && a.gitRoot != a.baseDir {
		parts = append(parts, fmt.Sprintf("Root: %s", a.gitRoot))
	}
	if a.resolved.Path != "" {
		if a.resolved.ResolvedBy == "session" {
			parts = append(parts, fmt.Sprintf("Plan: %s", a.resolved.Path))
		} else {
			parts = append(parts, fmt.Sprintf("Suggested: %s", a.resolved.Path))
		}
	}
	hookio.WriteContext(fmt.Sprintf("Session %s. %s\n", a.source, strings.Join(parts, " | ")))

	sc := a.shadowedCleanup
	if len(sc.Restored) > 0 || len(sc.Kept) > 0 {
		hookio.WriteContext("\n[!] SKILL-DEDUP CLEANUP (Issue #422): Recovered orphaned .shadowed/ directory.\n")
		if len(sc.Restored) > 0 {
			hookio.WriteContext(fmt.Sprintf("Restored %d skill(s): %s\n", len(sc.Restored), strings.Join(sc.Restored, ", ")))
		}
		if len(sc.Kept) > 0 {
			hookio.WriteContext(fmt.Sprintf("[!] Kept %d for review (content differs): %s\n", len(sc.Kept), strings.Join(sc.Kept, ", ")))
		}
	}

	if a.team != nil {
		hookio.WriteContext(fmt.Sprintf("[i] Agent Team detected: \"%s\" (%d members)\n", a.team.TeamName, a.team.MemberCount))
	}

	if a.gitRoot != "" && a.gitRoot != a.baseDir {
		hookio.WriteContext("Subdirectory mode: Plans/docs will be created in current directory\n")
		hookio.WriteContext(fmt.Sprintf("   Git root: %s\n", a.gitRoot))
	}

	if a.source == "compact" {
		hookio.WriteContext("\nCONTEXT COMPACTED - APPROVAL STATE CHECK:\nIf you were waiting for user approval via AskUserQuestion, you MUST re-confirm before proceeding.\n")
	}

	guidelines := project.GetCodingLevelGuidelines(a.codingLevel, "")
	if guidelines != "" {
		hookio.WriteContext(fmt.Sprintf("\n%s\n", guidelines))
	}

	if len(a.cfg.Assertions) > 0 {
		hookio.WriteContext("\nUser Assertions:\n")
		for i, assertion := range a.cfg.Assertions {
			hookio.WriteContext(fmt.Sprintf("  %d. %s\n", i+1, assertion))
		}
	}
}

// detectNodeVersion runs `node --version` and returns the result, or "N/A".
func detectNodeVersion() string {
	out, err := exec.Command("node", "--version").Output()
	if err != nil || len(out) == 0 {
		return "N/A"
	}
	return strings.TrimSpace(string(out))
}

// localTZ returns the local timezone string.
func localTZ() string {
	if runtime.GOOS != "windows" {
		if link, err := os.Readlink("/etc/localtime"); err == nil {
			const prefix = "/usr/share/zoneinfo/"
			if idx := strings.Index(link, prefix); idx >= 0 {
				return link[idx+len(prefix):]
			}
		}
		if data, err := os.ReadFile("/etc/timezone"); err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	if tz := os.Getenv("TZ"); tz != "" {
		return tz
	}
	return "UTC"
}

// firstOf returns the first non-empty string from the provided values.
func firstOf(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// unixMillis returns the current Unix timestamp in milliseconds.
func unixMillis() int64 {
	return time.Now().UnixMilli()
}
