// Validator checks plan directory completeness.
package plan

import (
	"os"
	"path/filepath"
	"strings"
)

// ValidationResult is the JSON output of plan validation.
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	PlanDir  string           `json:"planDir"`
	Errors   []string         `json:"errors,omitempty"`
	Warnings []string         `json:"warnings,omitempty"`
	Stats    ValidationStats  `json:"stats"`
}

// ValidationStats holds counts about the plan.
type ValidationStats struct {
	PhaseCount    int `json:"phaseCount"`
	TodoTotal     int `json:"todoTotal"`
	TodoCompleted int `json:"todoCompleted"`
}

// ValidatePlan checks a plan directory for completeness.
func ValidatePlan(planDir string) *ValidationResult {
	result := &ValidationResult{
		Valid:   true,
		PlanDir: planDir,
	}

	// Check plan.md exists
	planFile := filepath.Join(planDir, "plan.md")
	if _, err := os.Stat(planFile); os.IsNotExist(err) {
		result.Valid = false
		result.Errors = append(result.Errors, "plan.md not found")
	} else if err == nil {
		// Check frontmatter
		if status := extractFrontmatterField(planFile, "status"); status == "" {
			result.Warnings = append(result.Warnings, "plan.md missing status in frontmatter")
		}
	}

	// Count and validate phase files
	entries, err := os.ReadDir(planDir)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, "cannot read plan directory")
		return result
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "phase-") || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		result.Stats.PhaseCount++

		// Check phase has TODO section
		phasePath := filepath.Join(planDir, e.Name())
		data, err := os.ReadFile(phasePath)
		if err != nil {
			continue
		}
		content := string(data)
		hasTodo := strings.Contains(content, "## TODO") || strings.Contains(content, "## Todo")
		if !hasTodo {
			result.Warnings = append(result.Warnings, e.Name()+" missing TODO section")
		}

		// Count todo items
		for _, line := range strings.Split(content, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "- [ ]") || strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]") {
				result.Stats.TodoTotal++
				if strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]") {
					result.Stats.TodoCompleted++
				}
			}
		}
	}

	if result.Stats.PhaseCount == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "no phase files found")
	}

	return result
}
