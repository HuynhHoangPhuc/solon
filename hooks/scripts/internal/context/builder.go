// Package context builds session reminder context injected into Claude's context window.
package context

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"solon-hooks/internal/config"
	"solon-hooks/internal/exec"
	"solon-hooks/internal/hookio"
	"solon-hooks/internal/plan"
)

// ResolveRulesPath returns a relative or ~ path to a rules file, checking local then global.
func ResolveRulesPath(filename, configDirName string) string {
	if configDirName == "" {
		configDirName = ".claude"
	}
	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()

	localPath := filepath.Join(cwd, configDirName, "rules", filename)
	globalPath := filepath.Join(home, ".claude", "rules", filename)
	if fileExists(localPath) {
		return configDirName + "/rules/" + filename
	}
	if fileExists(globalPath) {
		return "~/.claude/rules/" + filename
	}
	// Backward compat: workflows/ legacy location
	localLegacy := filepath.Join(cwd, configDirName, "workflows", filename)
	globalLegacy := filepath.Join(home, ".claude", "workflows", filename)
	if fileExists(localLegacy) {
		return configDirName + "/workflows/" + filename
	}
	if fileExists(globalLegacy) {
		return "~/.claude/workflows/" + filename
	}
	return ""
}

// ResolveScriptPath returns a relative or ~ path to a scripts file.
func ResolveScriptPath(filename, configDirName string) string {
	if configDirName == "" {
		configDirName = ".claude"
	}
	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()

	localPath := filepath.Join(cwd, configDirName, "scripts", filename)
	globalPath := filepath.Join(home, ".claude", "scripts", filename)
	if fileExists(localPath) {
		return configDirName + "/scripts/" + filename
	}
	if fileExists(globalPath) {
		return "~/.claude/scripts/" + filename
	}
	return ""
}

// ResolveSkillsVenv returns the path to the skills venv Python binary.
func ResolveSkillsVenv(configDirName string) string {
	if configDirName == "" {
		configDirName = ".claude"
	}
	isWindows := runtime.GOOS == "windows"
	venvBin := "bin"
	pythonExe := "python3"
	if isWindows {
		venvBin = "Scripts"
		pythonExe = "python.exe"
	}
	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()

	localVenv := filepath.Join(cwd, configDirName, "skills", ".venv", venvBin, pythonExe)
	globalVenv := filepath.Join(home, ".claude", "skills", ".venv", venvBin, pythonExe)

	if fileExists(localVenv) {
		if isWindows {
			return configDirName + `\skills\.venv\Scripts\python.exe`
		}
		return configDirName + "/skills/.venv/bin/python3"
	}
	if fileExists(globalVenv) {
		if isWindows {
			return `~\.claude\skills\.venv\Scripts\python.exe`
		}
		return "~/.claude/skills/.venv/bin/python3"
	}
	return ""
}

// PlanContext holds resolved plan/naming context for reminder injection.
type PlanContext struct {
	ReportsPath    string
	GitBranch      string
	PlanLine       string
	NamePattern    string
	ValidationMode string
	ValidationMin  int
	ValidationMax  int
}

// BuildPlanContext resolves the active plan and naming pattern for the session.
func BuildPlanContext(sessionID string, cfg *hookio.SLConfig) PlanContext {
	gitBranch := exec.ExecSafe("git branch --show-current", "", 0)
	resolved := plan.ResolvePlanPath(sessionID, cfg)
	reportsPath := plan.GetReportsPath(resolved.Path, resolved.ResolvedBy, cfg.Plan, cfg.Paths, "")
	namePattern := plan.ResolveNamingPattern(cfg.Plan, gitBranch)

	var planLine string
	switch resolved.ResolvedBy {
	case "session":
		planLine = "- Plan: " + resolved.Path
	case "branch":
		planLine = "- Plan: none | Suggested: " + resolved.Path
	default:
		planLine = "- Plan: none"
	}

	v := cfg.Plan.Validation
	mode := v.Mode
	if mode == "" {
		mode = "prompt"
	}
	minQ := v.MinQuestions
	if minQ == 0 {
		minQ = 3
	}
	maxQ := v.MaxQuestions
	if maxQ == 0 {
		maxQ = 8
	}

	return PlanContext{
		ReportsPath:    reportsPath,
		GitBranch:      gitBranch,
		PlanLine:       planLine,
		NamePattern:    namePattern,
		ValidationMode: mode,
		ValidationMin:  minQ,
		ValidationMax:  maxQ,
	}
}

// WasRecentlyInjected returns true if the transcript already contains the
// modularization marker in the last 150 lines (prevents duplicate injection).
func WasRecentlyInjected(transcriptPath string) bool {
	if transcriptPath == "" {
		return false
	}
	data, err := os.ReadFile(transcriptPath)
	if err != nil {
		return false
	}
	lines := strings.Split(string(data), "\n")
	start := len(lines) - 150
	if start < 0 {
		start = 0
	}
	for _, line := range lines[start:] {
		if strings.Contains(line, "[IMPORTANT] Consider Modularization") {
			return true
		}
	}
	return false
}

// BuildReminderOpts holds all parameters for BuildReminderContext.
type BuildReminderOpts struct {
	SessionID      string
	Config         *hookio.SLConfig
	StaticEnv      map[string]string
	ConfigDirName  string
	BaseDir        string
}

// ReminderResult holds the assembled reminder context.
type ReminderResult struct {
	Content  string
	Lines    []string
	Sections map[string][]string
}

// BuildReminderContext assembles the full session reminder context.
func BuildReminderContext(opts BuildReminderOpts) ReminderResult {
	if opts.ConfigDirName == "" {
		opts.ConfigDirName = ".claude"
	}

	var cfg hookio.SLConfig
	if opts.Config != nil {
		cfg = *opts.Config
	} else {
		cfg = config.LoadConfig(config.LoadConfigOptions{
			IncludeProject:    false,
			IncludeAssertions: false,
			IncludeLocale:     true,
		})
	}

	devRulesPath := ResolveRulesPath("development-rules.md", opts.ConfigDirName)
	catalogScript := ResolveScriptPath("generate_catalogs.py", opts.ConfigDirName)
	skillsVenv := ResolveSkillsVenv(opts.ConfigDirName)
	planCtx := BuildPlanContext(opts.SessionID, &cfg)

	plansPathRel := plan.NormalizePath(cfg.Paths.Plans)
	if plansPathRel == "" {
		plansPathRel = "plans"
	}
	docsPathRel := plan.NormalizePath(cfg.Paths.Docs)
	if docsPathRel == "" {
		docsPathRel = "docs"
	}

	reportsPath := planCtx.ReportsPath
	plansPath := plansPathRel
	docsPath := docsPathRel
	if opts.BaseDir != "" {
		reportsPath = filepath.Join(opts.BaseDir, reportsPath)
		plansPath = filepath.Join(opts.BaseDir, plansPathRel)
		docsPath = filepath.Join(opts.BaseDir, docsPathRel)
	}

	docsMaxLoc := 800
	if cfg.Docs.MaxLoc > 0 {
		if v, err := strconv.Atoi(strconv.Itoa(cfg.Docs.MaxLoc)); err == nil && v > 0 {
			docsMaxLoc = v
		}
	}

	hooksConfig := cfg.Hooks
	contextEnabled := hooksConfig["context-tracking"] != false
	usageEnabled := hooksConfig["usage-context-awareness"] != false

	var lines []string
	sections := make(map[string][]string)

	langLines := BuildLanguageSection(cfg.Locale.ThinkingLanguage, cfg.Locale.ResponseLanguage)
	sections["language"] = langLines
	lines = append(lines, langLines...)

	sessLines := BuildSessionSection(opts.StaticEnv)
	sections["session"] = sessLines
	lines = append(lines, sessLines...)

	var ctxLines []string
	if contextEnabled {
		ctxLines = BuildContextSection(opts.SessionID)
	}
	sections["context"] = ctxLines
	lines = append(lines, ctxLines...)

	var usageLines []string
	if usageEnabled {
		usageLines = BuildUsageSection()
	}
	sections["usage"] = usageLines
	lines = append(lines, usageLines...)

	rulesLines := BuildRulesSection(devRulesPath, catalogScript, skillsVenv, plansPath, docsPath)
	sections["rules"] = rulesLines
	lines = append(lines, rulesLines...)

	modLines := BuildModularizationSection()
	sections["modularization"] = modLines
	lines = append(lines, modLines...)

	pathsLines := BuildPathsSection(reportsPath, plansPath, docsPath, docsMaxLoc)
	sections["paths"] = pathsLines
	lines = append(lines, pathsLines...)

	planLines := BuildPlanContextSection(
		planCtx.PlanLine, reportsPath, planCtx.GitBranch,
		planCtx.ValidationMode, planCtx.ValidationMin, planCtx.ValidationMax,
	)
	sections["planContext"] = planLines
	lines = append(lines, planLines...)

	namingLines := BuildNamingSection(reportsPath, plansPath, planCtx.NamePattern)
	sections["naming"] = namingLines
	lines = append(lines, namingLines...)

	return ReminderResult{
		Content:  strings.Join(lines, "\n"),
		Lines:    lines,
		Sections: sections,
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
