// Scaffolder creates plan directories and template files.
package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"solon-core/internal/template"
	"solon-core/internal/types"
)

// ScaffoldMode defines how many phases and what structure to create.
type ScaffoldMode string

const (
	ModeFast     ScaffoldMode = "fast"
	ModeHard     ScaffoldMode = "hard"
	ModeParallel ScaffoldMode = "parallel"
	ModeTwo      ScaffoldMode = "two"
)

// ScaffoldResult is the JSON output of plan scaffolding.
type ScaffoldResult struct {
	PlanDir      string   `json:"planDir"`
	Mode         string   `json:"mode"`
	FilesCreated []string `json:"filesCreated"`
}

// defaultPhases defines the phase templates per mode.
var defaultPhases = map[ScaffoldMode][]string{
	ModeFast: {
		"phase-01-research.md",
		"phase-02-implementation.md",
		"phase-03-testing.md",
	},
	ModeHard: {
		"phase-01-research.md",
		"phase-02-design.md",
		"phase-03-implementation.md",
		"phase-04-testing.md",
		"phase-05-review.md",
	},
	ModeParallel: {
		"phase-01-research.md",
		"phase-02-design.md",
		"phase-03-implementation-a.md",
		"phase-04-implementation-b.md",
		"phase-05-integration.md",
		"phase-06-testing.md",
		"phase-07-review.md",
	},
	ModeTwo: {
		"phase-01-first-pass.md",
		"phase-02-second-pass.md",
	},
}

// ScaffoldPlan creates a plan directory with template files.
func ScaffoldPlan(slug string, mode ScaffoldMode, numPhases int, cfg *types.SLConfig) (*ScaffoldResult, error) {
	if slug == "" {
		return nil, fmt.Errorf("slug is required")
	}
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Resolve plan directory name
	branch := ""
	plansDir := cfg.Paths.Plans
	if plansDir == "" {
		plansDir = "plans"
	}

	dirName := BuildPlanDirName(cfg.Plan, branch, slug)
	planDir := filepath.Join(plansDir, dirName)

	// Create directories
	if err := os.MkdirAll(filepath.Join(planDir, "research"), 0755); err != nil {
		return nil, fmt.Errorf("create plan dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(planDir, "reports"), 0755); err != nil {
		return nil, fmt.Errorf("create reports dir: %w", err)
	}

	var filesCreated []string

	// Write plan.md from template
	planContent := template.RenderPlan(slug, string(mode), dirName)
	planFile := filepath.Join(planDir, "plan.md")
	if err := os.WriteFile(planFile, []byte(planContent), 0644); err != nil {
		return nil, fmt.Errorf("write plan.md: %w", err)
	}
	filesCreated = append(filesCreated, "plan.md")

	// Determine phase files
	phases := defaultPhases[mode]
	if phases == nil {
		phases = defaultPhases[ModeHard]
	}

	// If custom numPhases specified and > 0, generate generic phase names
	if numPhases > 0 {
		phases = make([]string, numPhases)
		for i := range phases {
			phases[i] = fmt.Sprintf("phase-%02d-step-%d.md", i+1, i+1)
		}
	}

	// Write phase files from template
	for _, phaseName := range phases {
		title := phaseTitle(phaseName)
		phaseContent := template.RenderPhase(title, len(phases))
		phaseFile := filepath.Join(planDir, phaseName)
		if err := os.WriteFile(phaseFile, []byte(phaseContent), 0644); err != nil {
			return nil, fmt.Errorf("write %s: %w", phaseName, err)
		}
		filesCreated = append(filesCreated, phaseName)
	}

	return &ScaffoldResult{
		PlanDir:      planDir,
		Mode:         string(mode),
		FilesCreated: filesCreated,
	}, nil
}

// phaseTitle converts a phase filename to a human-readable title.
// e.g. "phase-01-research.md" → "Research"
func phaseTitle(filename string) string {
	name := strings.TrimSuffix(filename, ".md")
	// Remove "phase-NN-" prefix
	parts := strings.SplitN(name, "-", 3)
	if len(parts) >= 3 {
		name = parts[2]
	}
	// Convert kebab-case to Title Case
	words := strings.Split(name, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
