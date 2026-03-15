// Package task provides plan task extraction and sync utilities.
package task

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// TaskDef represents a single plan phase as a task definition.
type TaskDef struct {
	Phase       int      `json:"phase"`
	Title       string   `json:"title"`
	Priority    string   `json:"priority"`
	Effort      string   `json:"effort"`
	Description string   `json:"description"`
	PhaseFile   string   `json:"phaseFile"`
	BlockedBy   []int    `json:"blockedBy"`
	TodoCount   int      `json:"todoCount"`
	DoneCount   int      `json:"doneCount"`
}

// HydrateResult is the JSON output of HydratePlan.
type HydrateResult struct {
	PlanDir    string    `json:"planDir"`
	TaskCount  int       `json:"taskCount"`
	Tasks      []TaskDef `json:"tasks"`
	Skipped    bool      `json:"skipped"`
	SkipReason string    `json:"skipReason"`
}

var (
	rePhaseTitleAlt = regexp.MustCompile(`(?i)^#\s+Phase\s+\d+:\s*(.+)`)
	rePhaseTitleLbl = regexp.MustCompile(`(?i)^#\s+Phase:\s*(.+)`)
	rePriority      = regexp.MustCompile(`(?i)\*\*Priority:\*\*\s*(\S+)`)
	reEffort        = regexp.MustCompile(`(?i)\*\*Effort:\*\*\s*(\S+)`)
	reDescription   = regexp.MustCompile(`(?i)\*\*Description:\*\*\s*(.+)`)
	rePhaseNum      = regexp.MustCompile(`^phase-(\d+)-`)
)

// HydratePlan scans planDir for phase-*.md files and returns structured TaskDef list.
func HydratePlan(planDir string) (*HydrateResult, error) {
	entries, err := os.ReadDir(planDir)
	if err != nil {
		return nil, fmt.Errorf("read plan dir: %w", err)
	}

	var phaseFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "phase-") && strings.HasSuffix(e.Name(), ".md") {
			phaseFiles = append(phaseFiles, e.Name())
		}
	}
	sort.Strings(phaseFiles)

	result := &HydrateResult{PlanDir: planDir}

	if len(phaseFiles) < 3 {
		result.Skipped = true
		result.SkipReason = fmt.Sprintf("only %d phase file(s) found, minimum 3 required", len(phaseFiles))
		return result, nil
	}

	for _, fname := range phaseFiles {
		task, err := parsePhaseFile(planDir, fname)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", fname, err)
		}
		result.Tasks = append(result.Tasks, task)
	}

	// Sequential blocking: phase N blocked by N-1
	for i := range result.Tasks {
		if i == 0 {
			result.Tasks[i].BlockedBy = []int{}
		} else {
			result.Tasks[i].BlockedBy = []int{result.Tasks[i-1].Phase}
		}
	}

	result.TaskCount = len(result.Tasks)
	return result, nil
}

// parsePhaseFile parses a single phase markdown file into a TaskDef.
func parsePhaseFile(planDir, fname string) (TaskDef, error) {
	phaseNum := extractPhaseNum(fname)
	data, err := os.ReadFile(filepath.Join(planDir, fname))
	if err != nil {
		return TaskDef{}, err
	}

	lines := strings.Split(string(data), "\n")
	task := TaskDef{
		Phase:     phaseNum,
		PhaseFile: fname,
		BlockedBy: []int{},
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Extract title from heading
		if task.Title == "" {
			if m := rePhaseTitleAlt.FindStringSubmatch(trimmed); len(m) == 2 {
				task.Title = strings.TrimSpace(stripBackticks(m[1]))
				continue
			}
			if m := rePhaseTitleLbl.FindStringSubmatch(trimmed); len(m) == 2 {
				task.Title = strings.TrimSpace(stripBackticks(m[1]))
				continue
			}
		}

		// Extract priority
		if task.Priority == "" {
			if m := rePriority.FindStringSubmatch(trimmed); len(m) == 2 {
				task.Priority = m[1]
			}
		}

		// Extract effort
		if task.Effort == "" {
			if m := reEffort.FindStringSubmatch(trimmed); len(m) == 2 {
				task.Effort = m[1]
			}
		}

		// Extract description
		if task.Description == "" {
			if m := reDescription.FindStringSubmatch(trimmed); len(m) == 2 {
				task.Description = strings.TrimSpace(m[1])
			}
		}

		// Count TODO items
		if strings.HasPrefix(trimmed, "- [ ]") {
			task.TodoCount++
		} else if strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]") {
			task.DoneCount++
			task.TodoCount++
		}
	}

	return task, nil
}

// extractPhaseNum parses the phase number from a filename like phase-03-xxx.md.
func extractPhaseNum(fname string) int {
	m := rePhaseNum.FindStringSubmatch(fname)
	if len(m) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// stripBackticks removes surrounding backticks from a string.
func stripBackticks(s string) string {
	return strings.ReplaceAll(s, "`", "")
}
