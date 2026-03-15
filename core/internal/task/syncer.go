package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SyncResult is the output of SyncCompletions.
type SyncResult struct {
	PlanDir          string         `json:"planDir"`
	FilesModified    []string       `json:"filesModified"`
	CheckboxesUpdated int           `json:"checkboxesUpdated"`
	Details          []SyncDetail   `json:"details"`
}

// SyncDetail holds per-file sync stats.
type SyncDetail struct {
	File    string `json:"file"`
	Updated int    `json:"updated"`
}

// SyncCompletions marks all TODO items as done in the specified phase files.
// Uses atomic write (temp file + rename) for safety.
func SyncCompletions(planDir string, completedPhases []int) (*SyncResult, error) {
	if len(completedPhases) == 0 {
		return &SyncResult{PlanDir: planDir, FilesModified: []string{}, Details: []SyncDetail{}}, nil
	}

	entries, err := os.ReadDir(planDir)
	if err != nil {
		return nil, fmt.Errorf("read plan dir: %w", err)
	}

	// Build set of phase numbers to sync
	phaseSet := make(map[int]bool, len(completedPhases))
	for _, p := range completedPhases {
		phaseSet[p] = true
	}

	result := &SyncResult{
		PlanDir:       planDir,
		FilesModified: []string{},
		Details:       []SyncDetail{},
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "phase-") || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		phaseNum := extractPhaseNum(e.Name())
		if !phaseSet[phaseNum] {
			continue
		}

		updated, err := syncPhaseFile(planDir, e.Name())
		if err != nil {
			return nil, fmt.Errorf("sync %s: %w", e.Name(), err)
		}
		if updated > 0 {
			result.FilesModified = append(result.FilesModified, e.Name())
			result.CheckboxesUpdated += updated
			result.Details = append(result.Details, SyncDetail{File: e.Name(), Updated: updated})
		}
	}

	return result, nil
}

// syncPhaseFile replaces all unchecked "- [ ]" with "- [x]" in a phase file.
// Returns the count of replacements made. Uses atomic write.
func syncPhaseFile(planDir, fname string) (int, error) {
	fpath := filepath.Join(planDir, fname)
	data, err := os.ReadFile(fpath)
	if err != nil {
		return 0, err
	}

	content := string(data)
	count := strings.Count(content, "- [ ]")
	if count == 0 {
		return 0, nil
	}

	updated := strings.ReplaceAll(content, "- [ ]", "- [x]")

	// Atomic write: write to temp file then rename
	tmpFile := fpath + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(updated), 0644); err != nil {
		return 0, fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmpFile, fpath); err != nil {
		_ = os.Remove(tmpFile)
		return 0, fmt.Errorf("rename temp file: %w", err)
	}

	return count, nil
}
