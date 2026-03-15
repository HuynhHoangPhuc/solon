// Package workflow provides plan workflow status utilities.
package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PhaseStatus categorizes a single phase.
type PhaseStatus struct {
	Phase     int    `json:"phase"`
	File      string `json:"file"`
	State     string `json:"state"` // completed, in-progress, pending
	TodoTotal int    `json:"todoTotal"`
	TodoDone  int    `json:"todoDone"`
}

// PhaseCounts holds aggregate phase counts by state.
type PhaseCounts struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	InProgress int `json:"inProgress"`
	Pending    int `json:"pending"`
}

// WorkflowStatus is the JSON output of GetStatus.
type WorkflowStatus struct {
	PlanDir  string       `json:"planDir"`
	Status   string       `json:"status"` // completed, in-progress, pending
	Progress int          `json:"progress"` // 0-100
	Phases   PhaseCounts  `json:"phases"`
	Reports  int          `json:"reports"`
	Detail   []PhaseStatus `json:"detail,omitempty"`
}

// GetStatus returns the workflow status for the given plan directory.
func GetStatus(planDir string) (*WorkflowStatus, error) {
	entries, err := os.ReadDir(planDir)
	if err != nil {
		return nil, fmt.Errorf("read plan dir: %w", err)
	}

	result := &WorkflowStatus{PlanDir: planDir}

	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "phase-") || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		ps, err := categorizePhase(planDir, e.Name())
		if err != nil {
			return nil, fmt.Errorf("categorize %s: %w", e.Name(), err)
		}
		result.Detail = append(result.Detail, ps)
		result.Phases.Total++
		switch ps.State {
		case "completed":
			result.Phases.Completed++
		case "in-progress":
			result.Phases.InProgress++
		default:
			result.Phases.Pending++
		}
	}

	// Calculate progress percentage
	if result.Phases.Total > 0 {
		result.Progress = (result.Phases.Completed * 100) / result.Phases.Total
	}

	// Overall status
	switch {
	case result.Phases.Total == 0:
		result.Status = "pending"
	case result.Phases.Completed == result.Phases.Total:
		result.Status = "completed"
	case result.Phases.Completed > 0 || result.Phases.InProgress > 0:
		result.Status = "in-progress"
	default:
		result.Status = "pending"
	}

	// Count reports
	result.Reports = countReportFiles(planDir)

	return result, nil
}

// categorizePhase reads a phase file and determines its completion state.
func categorizePhase(planDir, fname string) (PhaseStatus, error) {
	data, err := os.ReadFile(filepath.Join(planDir, fname))
	if err != nil {
		return PhaseStatus{}, err
	}

	ps := PhaseStatus{
		Phase: extractPhaseNumFromName(fname),
		File:  fname,
	}

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]") {
			ps.TodoDone++
			ps.TodoTotal++
		} else if strings.HasPrefix(trimmed, "- [ ]") {
			ps.TodoTotal++
		}
	}

	switch {
	case ps.TodoTotal == 0:
		ps.State = "pending"
	case ps.TodoDone == ps.TodoTotal:
		ps.State = "completed"
	case ps.TodoDone > 0:
		ps.State = "in-progress"
	default:
		ps.State = "pending"
	}

	return ps, nil
}

// extractPhaseNumFromName parses the phase number from a filename like phase-03-xxx.md.
func extractPhaseNumFromName(fname string) int {
	parts := strings.SplitN(strings.TrimPrefix(fname, "phase-"), "-", 2)
	if len(parts) == 0 {
		return 0
	}
	n := 0
	for _, ch := range parts[0] {
		if ch < '0' || ch > '9' {
			break
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

// countReportFiles counts files in reports/ and research/ subdirectories.
func countReportFiles(planDir string) int {
	count := 0
	for _, sub := range []string{"reports", "research"} {
		entries, err := os.ReadDir(filepath.Join(planDir, sub))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				count++
			}
		}
	}
	return count
}
