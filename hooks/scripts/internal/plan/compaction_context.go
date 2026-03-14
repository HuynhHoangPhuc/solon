package plan

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"solon-hooks/internal/wisdom"
)

// BuildCompactionContext generates structured recovery context for post-compaction sessions.
// Returns empty string if no plan is active or nothing useful to inject.
func BuildCompactionContext(planPath, sessionID string) string {
	if planPath == "" {
		return ""
	}

	var sections []string

	// 1. Phase completion status
	phaseStatus := readPhaseStatus(planPath)
	if phaseStatus != "" {
		sections = append(sections, "Plan Progress:\n"+phaseStatus)
	}

	// 2. Accumulated wisdom
	wisdomContent := wisdom.ReadWisdom(planPath, sessionID, 10)
	if wisdomContent != "" {
		sections = append(sections, "Accumulated Learnings:\n"+wisdomContent)
	}

	if len(sections) == 0 {
		return ""
	}

	return "POST-COMPACTION RECOVERY CONTEXT:\n" +
		strings.Join(sections, "\n\n") +
		"\n\nUse this context to resume work without re-reading completed phases."
}

// readPhaseStatus scans phase-*.md files for todo completion percentages.
func readPhaseStatus(planPath string) string {
	phaseFiles, _ := filepath.Glob(filepath.Join(planPath, "phase-*.md"))
	if len(phaseFiles) == 0 {
		return ""
	}

	var lines []string
	for _, f := range phaseFiles {
		name := filepath.Base(f)
		total, done := countTodos(f)
		if total == 0 {
			lines = append(lines, fmt.Sprintf("  - %s: no todos", name))
		} else {
			status := "in-progress"
			if done == total {
				status = "complete"
			}
			lines = append(lines, fmt.Sprintf("  - %s: %d/%d todos (%s)", name, done, total, status))
		}
	}
	return strings.Join(lines, "\n")
}

// countTodos counts total and completed todos (- [ ] / - [x]) in a file.
func countTodos(path string) (total, done int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "- [ ]") {
			total++
		} else if strings.HasPrefix(line, "- [x]") || strings.HasPrefix(line, "- [X]") {
			total++
			done++
		}
	}
	return total, done
}
