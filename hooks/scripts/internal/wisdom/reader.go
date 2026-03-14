// Package wisdom provides utilities for reading and resolving wisdom files.
package wisdom

import (
	"os"
	"path/filepath"
	"strings"
)

// ReadWisdom reads the last maxLines from .wisdom.md in the plan directory.
// Falls back to /tmp/sl-wisdom-{sessionID}.md if planPath is empty.
// Returns empty string if no wisdom file exists.
func ReadWisdom(planPath, sessionID string, maxLines int) string {
	path := resolveWisdomPath(planPath, sessionID)
	if path == "" {
		return ""
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		return ""
	}

	// Take last N lines (newest entries)
	start := 0
	if len(lines) > maxLines {
		start = len(lines) - maxLines
	}
	return strings.Join(lines[start:], "\n")
}

// resolveWisdomPath returns the wisdom file path, preferring plan dir over /tmp.
func resolveWisdomPath(planPath, sessionID string) string {
	if planPath != "" {
		p := filepath.Join(planPath, ".wisdom.md")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if sessionID != "" {
		return filepath.Join(os.TempDir(), "sl-wisdom-"+sessionID+".md")
	}
	return ""
}
