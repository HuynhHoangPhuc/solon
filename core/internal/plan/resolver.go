// Plan path resolution using cascading strategy: session → branch.
// Ported from solon-hooks/internal/plan/resolver.go — keep in sync.
package plan

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"solon-core/internal/exec"
	"solon-core/internal/session"
	"solon-core/internal/types"
)

// defaultBranchPattern matches feature branch naming conventions.
var defaultBranchPattern = regexp.MustCompile(`(?:feat|fix|chore|refactor|docs)/(?:[^/]+/)?(.+)`)

// ExtractSlugFromBranch extracts a sanitized feature slug from a git branch name.
func ExtractSlugFromBranch(branch, pattern string) string {
	if branch == "" {
		return ""
	}
	var re *regexp.Regexp
	if pattern != "" {
		var err error
		re, err = regexp.Compile(pattern)
		if err != nil {
			re = defaultBranchPattern
		}
	} else {
		re = defaultBranchPattern
	}
	m := re.FindStringSubmatch(branch)
	if len(m) < 2 {
		return ""
	}
	return SanitizeSlug(m[1])
}

// FindMostRecentPlan returns the path to the most recently created plan directory.
func FindMostRecentPlan(plansDir string) string {
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		return ""
	}
	timestampPrefix := regexp.MustCompile(`^\d{6}`)
	var latest string
	for _, e := range entries {
		if e.IsDir() && timestampPrefix.MatchString(e.Name()) {
			if e.Name() > latest {
				latest = e.Name()
			}
		}
	}
	if latest == "" {
		return ""
	}
	return filepath.Join(plansDir, latest)
}

// ResolveResult is the JSON output of plan resolution.
type ResolveResult struct {
	Path       string `json:"path"`
	ResolvedBy string `json:"resolvedBy"`
	Absolute   string `json:"absolute,omitempty"`
	PlanFile   string `json:"planFile,omitempty"`
	Status     string `json:"status,omitempty"`
	Phases     int    `json:"phases,omitempty"`
}

// ResolvePlanPath resolves the active plan using the configured resolution order.
func ResolvePlanPath(sessionID string, cfg *types.SLConfig) types.PlanResolution {
	if cfg == nil {
		return types.PlanResolution{}
	}
	plansDir := cfg.Paths.Plans
	if plansDir == "" {
		plansDir = "plans"
	}
	resolution := cfg.Plan.Resolution
	order := resolution.Order
	if len(order) == 0 {
		order = []string{"session", "branch"}
	}

	for _, method := range order {
		switch method {
		case "session":
			state := session.ReadSessionState(sessionID)
			if state != nil && state.ActivePlan != nil && *state.ActivePlan != "" {
				resolved := *state.ActivePlan
				if !filepath.IsAbs(resolved) && state.SessionOrigin != "" {
					resolved = filepath.Join(state.SessionOrigin, resolved)
				}
				return types.PlanResolution{Path: resolved, ResolvedBy: "session"}
			}

		case "branch":
			branch := exec.GitSafe("git branch --show-current", "")
			slug := ExtractSlugFromBranch(branch, resolution.BranchPattern)
			if slug == "" {
				break
			}
			entries, err := os.ReadDir(plansDir)
			if err != nil {
				break
			}
			var match string
			for _, e := range entries {
				if e.IsDir() && strings.Contains(e.Name(), slug) {
					match = e.Name()
				}
			}
			if match != "" {
				return types.PlanResolution{
					Path:       filepath.Join(plansDir, match),
					ResolvedBy: "branch",
				}
			}
		}
	}
	return types.PlanResolution{}
}

// EnrichResolveResult adds absolute path, plan file info, status, and phase count.
func EnrichResolveResult(res types.PlanResolution) ResolveResult {
	result := ResolveResult{
		Path:       res.Path,
		ResolvedBy: res.ResolvedBy,
	}
	if res.Path == "" {
		return result
	}

	// Compute absolute path
	if filepath.IsAbs(res.Path) {
		result.Absolute = res.Path
	} else {
		cwd, _ := os.Getwd()
		result.Absolute = filepath.Join(cwd, res.Path)
	}

	// Check for plan.md
	planFile := filepath.Join(result.Absolute, "plan.md")
	if _, err := os.Stat(planFile); err == nil {
		result.PlanFile = "plan.md"
		result.Status = extractFrontmatterField(planFile, "status")
	}

	// Count phase files
	result.Phases = countPhaseFiles(result.Absolute)
	return result
}

// extractFrontmatterField reads a simple YAML frontmatter field value.
func extractFrontmatterField(filePath, field string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	inFrontmatter := false
	prefix := field + ":"
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if inFrontmatter {
				break
			}
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		}
	}
	return ""
}

// countPhaseFiles counts files matching phase-*.md in a directory.
func countPhaseFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "phase-") && strings.HasSuffix(e.Name(), ".md") {
			count++
		}
	}
	return count
}

// GetReportsPath returns the reports path based on plan resolution.
func GetReportsPath(planPath, resolvedBy string, planCfg types.PlanConfig, pathsCfg types.PathsConfig, baseDir string) string {
	reportsDir := NormalizePath(planCfg.ReportsDir)
	if reportsDir == "" {
		reportsDir = "reports"
	}
	plansDir := NormalizePath(pathsCfg.Plans)
	if plansDir == "" {
		plansDir = "plans"
	}

	var reportPath string
	if planPath != "" && resolvedBy == "session" {
		normalizedPlan := NormalizePath(planPath)
		if normalizedPlan != "" {
			reportPath = normalizedPlan + "/" + reportsDir
		}
	}
	if reportPath == "" {
		reportPath = plansDir + "/" + reportsDir
	}

	if baseDir != "" {
		return filepath.Join(baseDir, reportPath)
	}
	return reportPath + "/"
}
