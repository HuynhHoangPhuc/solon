// Package report provides plan report indexing utilities.
package report

import (
	"fmt"
	"os"
	"path/filepath"
)

// ReportEntry represents a single report file found in a plan directory.
type ReportEntry struct {
	Filename  string `json:"filename"`
	Directory string `json:"directory"` // "reports" or "research"
	Path      string `json:"path"`      // full path
}

// IndexReports scans reports/ and research/ subdirectories of planDir.
// Returns a list of report entries sorted by directory then filename.
func IndexReports(planDir string) ([]ReportEntry, error) {
	entries := []ReportEntry{}

	for _, sub := range []string{"reports", "research"} {
		subDir := filepath.Join(planDir, sub)
		dirEntries, err := os.ReadDir(subDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", sub, err)
		}
		for _, e := range dirEntries {
			if e.IsDir() {
				continue
			}
			entries = append(entries, ReportEntry{
				Filename:  e.Name(),
				Directory: sub,
				Path:      filepath.Join(subDir, e.Name()),
			})
		}
	}

	return entries, nil
}
