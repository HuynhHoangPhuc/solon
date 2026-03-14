// Context section builders for session reminder injection.
package context

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	usageCacheFile    = "sl-usage-limits-cache.json"
	warnThreshold     = 70
	criticalThreshold = 90
)

// BuildLanguageSection returns lines for the Language section (empty if no locale set).
func BuildLanguageSection(thinkingLanguage, responseLanguage *string) []string {
	thinking := ""
	if thinkingLanguage != nil {
		thinking = *thinkingLanguage
	}
	response := ""
	if responseLanguage != nil {
		response = *responseLanguage
	}

	effectiveThinking := thinking
	if effectiveThinking == "" && response != "" {
		effectiveThinking = "en"
	}
	hasThinking := effectiveThinking != "" && effectiveThinking != response

	if !hasThinking && response == "" {
		return nil
	}

	lines := []string{"## Language"}
	if hasThinking {
		lines = append(lines, fmt.Sprintf("- Thinking: Use %s for reasoning (logic, precision).", effectiveThinking))
	}
	if response != "" {
		lines = append(lines, fmt.Sprintf("- Response: Respond in %s (natural, fluent).", response))
	}
	lines = append(lines, "")
	return lines
}

// BuildSessionSection returns lines for the Session section with system info.
func BuildSessionSection(staticEnv map[string]string) []string {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memUsedMB := int(memStats.HeapInuse / 1024 / 1024)
	totalMemMB := int(getTotalMemMB())
	memPct := 0
	if totalMemMB > 0 {
		memPct = int(math.Round(float64(memUsedMB) / float64(totalMemMB) * 100))
	}

	cwd := getStatic(staticEnv, "cwd")
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	tz := getStatic(staticEnv, "timezone")
	if tz == "" {
		tz = localTZ()
	}
	platform := getStatic(staticEnv, "osPlatform")
	if platform == "" {
		platform = runtime.GOOS
	}
	user := getStatic(staticEnv, "user")
	if user == "" {
		user = firstNonEmptyEnv("USERNAME", "USER", "LOGNAME")
	}
	locale := getStatic(staticEnv, "locale")
	if locale == "" {
		locale = os.Getenv("LANG")
	}

	return []string{
		"## Session",
		fmt.Sprintf("- DateTime: %s", time.Now().Format("1/2/2006, 3:04:05 PM")),
		fmt.Sprintf("- CWD: %s", cwd),
		fmt.Sprintf("- Timezone: %s", tz),
		fmt.Sprintf("- Working directory: %s", cwd),
		fmt.Sprintf("- OS: %s", platform),
		fmt.Sprintf("- User: %s", user),
		fmt.Sprintf("- Locale: %s", locale),
		fmt.Sprintf("- Memory usage: %dMB/%dMB (%d%%)", memUsedMB, totalMemMB, memPct),
		"- CPU usage: N/A (Go runtime)",
		"- Spawning multiple subagents can cause performance issues, spawn and delegate tasks intelligently based on the available system resources.",
		"- Remember that each subagent only has 200K tokens in context window, spawn and delegate tasks intelligently to make sure their context windows don't get bloated.",
		"- IMPORTANT: Include these environment information when prompting subagents to perform tasks.",
		"",
	}
}

// BuildContextSection returns lines for the context-usage section (empty if no cache).
func BuildContextSection(sessionID string) []string {
	if sessionID == "" {
		return nil
	}
	contextPath := filepath.Join(os.TempDir(), fmt.Sprintf("sl-context-%s.json", sessionID))
	data, err := os.ReadFile(contextPath)
	if err != nil {
		return nil
	}
	var ctx struct {
		Timestamp int64   `json:"timestamp"`
		Tokens    int     `json:"tokens"`
		Size      int     `json:"size"`
		Percent   float64 `json:"percent"`
	}
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil
	}
	if time.Since(time.UnixMilli(ctx.Timestamp)) > 5*time.Minute {
		return nil
	}
	pct := int(math.Round(ctx.Percent))
	usedK := int(math.Round(float64(ctx.Tokens) / 1000))
	sizeK := int(math.Round(float64(ctx.Size) / 1000))

	lines := []string{
		"## Current Session's Context",
		fmt.Sprintf("- Context: %d%% used (%dK/%dK tokens)", pct, usedK, sizeK),
		"- **NOTE:** Optimize the workflow for token efficiency",
	}
	if pct >= criticalThreshold {
		lines = append(lines, "- **CRITICAL:** Context nearly full - consider compaction or being concise, update current phase's status before the compaction.")
	} else if pct >= warnThreshold {
		lines = append(lines, "- **WARNING:** Context usage moderate - being concise and optimize token efficiency.")
	}
	lines = append(lines, "")
	return lines
}

// usageData mirrors the usage cache JSON structure.
type usageData struct {
	FiveHour *struct {
		Utilization *float64 `json:"utilization"`
		ResetsAt    string   `json:"resets_at"`
	} `json:"five_hour"`
	SevenDay *struct {
		Utilization *float64 `json:"utilization"`
	} `json:"seven_day"`
}

func readUsageCache() *usageData {
	path := filepath.Join(os.TempDir(), usageCacheFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cache struct {
		Timestamp int64     `json:"timestamp"`
		Data      usageData `json:"data"`
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil
	}
	if time.Since(time.UnixMilli(cache.Timestamp)) > 5*time.Minute {
		return nil
	}
	return &cache.Data
}

func formatTimeUntilReset(resetAt string) string {
	if resetAt == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, resetAt)
	if err != nil {
		return ""
	}
	remaining := int(time.Until(t).Seconds())
	if remaining <= 0 || remaining > 18000 {
		return ""
	}
	h := remaining / 3600
	m := (remaining % 3600) / 60
	return fmt.Sprintf("%dh %dm", h, m)
}

func formatUsagePct(value float64, label string) string {
	pct := int(math.Round(value))
	if pct >= criticalThreshold {
		return fmt.Sprintf("%s: %d%% [CRITICAL]", label, pct)
	}
	if pct >= warnThreshold {
		return fmt.Sprintf("%s: %d%% [WARNING]", label, pct)
	}
	return fmt.Sprintf("%s: %d%%", label, pct)
}

// BuildUsageSection returns lines for the usage-limits section (empty if no cache).
func BuildUsageSection() []string {
	usage := readUsageCache()
	if usage == nil {
		return nil
	}
	var parts []string
	if usage.FiveHour != nil {
		if usage.FiveHour.Utilization != nil {
			parts = append(parts, formatUsagePct(*usage.FiveHour.Utilization, "5h"))
		}
		if t := formatTimeUntilReset(usage.FiveHour.ResetsAt); t != "" {
			parts = append(parts, "resets in "+t)
		}
	}
	if usage.SevenDay != nil && usage.SevenDay.Utilization != nil {
		parts = append(parts, formatUsagePct(*usage.SevenDay.Utilization, "7d"))
	}
	if len(parts) == 0 {
		return nil
	}
	return []string{"## Usage Limits", "- " + strings.Join(parts, " | "), ""}
}

// BuildRulesSection returns lines for the Rules section.
func BuildRulesSection(devRulesPath, catalogScript, skillsVenv, plansPath, docsPath string) []string {
	if plansPath == "" {
		plansPath = "plans"
	}
	if docsPath == "" {
		docsPath = "docs"
	}
	lines := []string{"## Rules"}
	if devRulesPath != "" {
		lines = append(lines, fmt.Sprintf(`- Read and follow development rules: "%s"`, devRulesPath))
	}
	lines = append(lines, fmt.Sprintf(`- Markdown files are organized in: Plans → "%s" directory, Docs → "%s" directory`, plansPath, docsPath))
	lines = append(lines, fmt.Sprintf(`- **IMPORTANT:** DO NOT create markdown files outside of "%s" or "%s" UNLESS the user explicitly requests it.`, plansPath, docsPath))
	if catalogScript != "" {
		lines = append(lines, fmt.Sprintf("- Activate skills: Run `python %s --skills` to generate a skills catalog and analyze it, then activate the relevant skills that are needed for the task during the process.", catalogScript))
	}
	if skillsVenv != "" {
		lines = append(lines, fmt.Sprintf("- Python scripts in .claude/skills/: Use `%s`", skillsVenv))
	}
	lines = append(lines,
		"- When skills' scripts are failed to execute, always fix them and run again, repeat until success.",
		"- Follow **YAGNI (You Aren't Gonna Need It) - KISS (Keep It Simple, Stupid) - DRY (Don't Repeat Yourself)** principles",
		"- Sacrifice grammar for the sake of concision when writing reports.",
		"- In reports, list any unresolved questions at the end, if any.",
		"- IMPORTANT: Ensure token consumption efficiency while maintaining high quality.",
		"",
	)
	return lines
}

// BuildModularizationSection returns the standard modularization reminder lines.
func BuildModularizationSection() []string {
	return []string{
		"## **[IMPORTANT] Consider Modularization:**",
		"- Check existing modules before creating new",
		"- Analyze logical separation boundaries (functions, classes, concerns)",
		"- Prefer kebab-case for JS/TS/Python/shell; respect language conventions (C#/Java use PascalCase, Go/Rust use snake_case)",
		"- Write descriptive code comments",
		"- After modularization, continue with main task",
		"- When not to modularize: Markdown files, plain text files, bash scripts, configuration files, environment variables files, etc.",
		"",
	}
}

// BuildPathsSection returns lines for the Paths section.
func BuildPathsSection(reportsPath, plansPath, docsPath string, docsMaxLoc int) []string {
	if docsMaxLoc <= 0 {
		docsMaxLoc = 800
	}
	return []string{
		"## Paths",
		fmt.Sprintf("Reports: %s | Plans: %s/ | Docs: %s/ | docs.maxLoc: %d", reportsPath, plansPath, docsPath, docsMaxLoc),
		"",
	}
}

// BuildPlanContextSection returns lines for the Plan Context section.
func BuildPlanContextSection(planLine, reportsPath, gitBranch, validationMode string, validationMin, validationMax int) []string {
	lines := []string{
		"## Plan Context",
		planLine,
		fmt.Sprintf("- Reports: %s", reportsPath),
	}
	if gitBranch != "" {
		lines = append(lines, fmt.Sprintf("- Branch: %s", gitBranch))
	}
	lines = append(lines, fmt.Sprintf("- Validation: mode=%s, questions=%d-%d", validationMode, validationMin, validationMax))
	lines = append(lines, "")
	return lines
}

// BuildNamingSection returns lines for the Naming section.
func BuildNamingSection(reportsPath, plansPath, namePattern string) []string {
	return []string{
		"## Naming",
		fmt.Sprintf("- Report: `%s{type}-%s.md`", reportsPath, namePattern),
		fmt.Sprintf("- Plan dir: `%s/%s/`", plansPath, namePattern),
		"- Replace `{type}` with: agent name, report type, or context",
		"- Replace `{slug}` in pattern with: descriptive-kebab-slug",
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

func getStatic(m map[string]string, key string) string {
	if m == nil {
		return ""
	}
	return m[key]
}

func firstNonEmptyEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func getTotalMemMB() uint64 {
	// Read from /proc/meminfo on Linux, fall back to 0 (unknown)
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			var kb uint64
			fmt.Sscanf(strings.TrimPrefix(line, "MemTotal:"), "%d", &kb)
			return kb / 1024
		}
	}
	return 0
}

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
