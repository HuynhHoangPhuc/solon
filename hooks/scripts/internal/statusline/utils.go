// utils.go: Statusline utility functions — visible length, elapsed time, home collapse, terminal width.
package statusline

import (
	"os"
	"regexp"
	"strconv"
	"time"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// VisibleLength returns the visible column width of a string.
// Strips ANSI escape codes, counts emoji as 2 columns (SMP + misc symbols).
func VisibleLength(str string) int {
	if str == "" {
		return 0
	}
	noAnsi := ansiRe.ReplaceAllString(str, "")
	width := 0
	for _, r := range noAnsi {
		if (r >= 0x1f300 && r <= 0x1f9ff) ||
			(r >= 0x2600 && r <= 0x26ff) ||
			(r >= 0x2700 && r <= 0x27bf) {
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

// FormatElapsed formats the duration from start to end (or now if end is zero).
// Returns "<1s" for sub-second durations, "Xs" for seconds, "Xm Ys" for minutes.
func FormatElapsed(start, end time.Time) string {
	if start.IsZero() {
		return "0s"
	}
	endT := end
	if endT.IsZero() {
		endT = time.Now()
	}
	ms := endT.Sub(start).Milliseconds()
	if ms < 0 || ms < 1000 {
		return "<1s"
	}
	if ms < 60000 {
		return strconv.FormatInt(ms/1000, 10) + "s"
	}
	mins := ms / 60000
	secs := (ms % 60000) / 1000
	return strconv.FormatInt(mins, 10) + "m " + strconv.FormatInt(secs, 10) + "s"
}

// CollapseHome replaces the home directory prefix with "~".
func CollapseHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if len(path) >= len(home) && path[:len(home)] == home {
		return "~" + path[len(home):]
	}
	return path
}

// GetTerminalWidth returns the terminal width for layout decisions.
// Priority: stderr columns → $COLUMNS env → 120.
func GetTerminalWidth() int {
	// os.Stderr doesn't expose columns directly; use syscall via fd check
	// Fall back to $COLUMNS or default 120
	if col := os.Getenv("COLUMNS"); col != "" {
		if n, err := strconv.Atoi(col); err == nil && n > 0 {
			return n
		}
	}
	return 120
}
