// ANSI error message formatting for scout-block denials.
package scout

import (
	"os"
	"path/filepath"
	"strings"
)

func supportsColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}
	// Check if stderr is a TTY via file stat (best-effort on Unix)
	info, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func colorize(text, code string) string {
	if !supportsColor() {
		return text
	}
	return code + text + "\x1b[0m"
}

func formatConfigPath(claudeDir string) string {
	if claudeDir != "" {
		return filepath.Join(claudeDir, ".slignore")
	}
	return ".claude/.slignore"
}

// FormatBlockedError returns a rich ANSI error message for a blocked path.
func FormatBlockedError(blockedPath, pattern, tool, claudeDir string) string {
	configPath := formatConfigPath(claudeDir)
	displayPath := blockedPath
	if len(displayPath) > 60 {
		displayPath = "..." + displayPath[len(displayPath)-57:]
	}

	lines := []string{
		"",
		colorize("NOTE:", "\x1b[36m") + " This is not an error - this block is intentional to optimize context.",
		"",
		colorize("BLOCKED", "\x1b[31m") + ": Access to '" + displayPath + "' denied",
		"",
		"  " + colorize("Pattern:", "\x1b[33m") + "  " + pattern,
		"  " + colorize("Tool:", "\x1b[33m") + "     " + tool,
		"",
		"  " + colorize("To allow, add to", "\x1b[34m") + " " + configPath + ":",
		"    !" + pattern,
		"",
		"  " + colorize("Config:", "\x1b[2m") + " " + configPath,
		"",
	}
	return strings.Join(lines, "\n")
}

// FormatBroadPatternError returns a rich ANSI error message for broad glob patterns.
func FormatBroadPatternError(pattern string, suggestions []string) string {
	lines := []string{
		"",
		colorize("NOTE:", "\x1b[36m") + " This is not an error - this block is intentional to optimize context.",
		"",
		colorize("BLOCKED", "\x1b[31m") + ": Overly broad glob pattern detected",
		"",
		"  " + colorize("Pattern:", "\x1b[33m") + "  " + pattern,
		"  " + colorize("Reason:", "\x1b[33m") + "   Would return ALL matching files, filling context",
		"",
		"  " + colorize("Use more specific patterns:", "\x1b[34m"),
	}
	for _, s := range suggestions {
		lines = append(lines, "    • "+s)
	}
	lines = append(lines, "")
	lines = append(lines, "  "+colorize("Tip: Target specific directories to avoid context overflow", "\x1b[2m"))
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}
