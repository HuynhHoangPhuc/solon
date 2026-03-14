// Plan path resolution using cascading strategy: session → branch.
package plan

import (
	"os"
	"path/filepath"
	"regexp"

	"solon-hooks/internal/exec"
	"solon-hooks/internal/hookio"
	"solon-hooks/internal/session"
)

// defaultBranchPattern matches feature branch naming conventions.
var defaultBranchPattern = regexp.MustCompile(`(?:feat|fix|chore|refactor|docs)/(?:[^/]+/)?(.+)`)

// ExtractSlugFromBranch extracts a sanitized feature slug from a git branch name.
// Uses the provided regex pattern string, falling back to the default pattern.
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
// Plan dirs are identified by a 6-digit timestamp prefix (e.g. "251212-...").
func FindMostRecentPlan(plansDir string) string {
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		return ""
	}
	timestampPrefix := regexp.MustCompile(`^\d{6}`)
	var candidates []string
	for _, e := range entries {
		if e.IsDir() && timestampPrefix.MatchString(e.Name()) {
			candidates = append(candidates, e.Name())
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	// Sort lexicographically — timestamp prefix ensures chronological order
	latest := candidates[0]
	for _, c := range candidates[1:] {
		if c > latest {
			latest = c
		}
	}
	return filepath.Join(plansDir, latest)
}

// ResolvePlanPath resolves the active plan using the configured resolution order.
// - "session": reads the session state file → returns ACTIVE plan
// - "branch":  matches git branch slug → returns SUGGESTED plan
func ResolvePlanPath(sessionID string, cfg *hookio.SLConfig) hookio.PlanResolution {
	if cfg == nil {
		return hookio.PlanResolution{}
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
				return hookio.PlanResolution{Path: resolved, ResolvedBy: "session"}
			}

		case "branch":
			branch := exec.ExecGitSafe("git branch --show-current", "")
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
				if e.IsDir() && containsStr(e.Name(), slug) {
					match = e.Name()
				}
			}
			if match != "" {
				return hookio.PlanResolution{
					Path:       filepath.Join(plansDir, match),
					ResolvedBy: "branch",
				}
			}
		}
	}
	return hookio.PlanResolution{}
}

// GetReportsPath returns the reports path based on plan resolution.
// Only uses a plan-specific path for "session"-resolved plans.
func GetReportsPath(planPath, resolvedBy string, planCfg hookio.PlanConfig, pathsCfg hookio.PathsConfig, baseDir string) string {
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

// ExtractTaskListID returns the base directory name for session-resolved plans.
// Returns empty string for branch-resolved or unresolved plans.
func ExtractTaskListID(resolved hookio.PlanResolution) string {
	if resolved.ResolvedBy != "session" || resolved.Path == "" {
		return ""
	}
	return filepath.Base(resolved.Path)
}

func containsStr(s, substr string) bool {
	if substr == "" {
		return false
	}
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
