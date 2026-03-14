// Package colors provides ANSI terminal color helpers.
// Respects NO_COLOR and FORCE_COLOR environment variables.
// Defaults to color-on since statusline output always renders in a TTY context.
package colors

import "os"

// ANSI escape code constants.
const (
	Reset   = "\x1b[0m"
	Dim     = "\x1b[2m"
	Red     = "\x1b[31m"
	Green   = "\x1b[32m"
	Yellow  = "\x1b[33m"
	Magenta = "\x1b[35m"
	Cyan    = "\x1b[36m"
)

// ShouldUseColor returns true when ANSI color output is appropriate.
// NO_COLOR disables; FORCE_COLOR enables; default is true.
func ShouldUseColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}
	return true // default on for statusline context
}

// Has256Color returns true when the terminal supports 256 colors.
func Has256Color() bool {
	ct := os.Getenv("COLORTERM")
	return ct == "truecolor" || ct == "24bit" || ct == "256color"
}

// Colorize wraps text with an ANSI code + reset, or returns plain text when
// color is disabled.
func Colorize(text, code string) string {
	if !ShouldUseColor() {
		return text
	}
	return code + text + Reset
}

// Green returns text in green.
func GreenStr(text string) string { return Colorize(text, Green) }

// Yellow returns text in yellow.
func YellowStr(text string) string { return Colorize(text, Yellow) }

// Red returns text in red.
func RedStr(text string) string { return Colorize(text, Red) }

// Cyan returns text in cyan.
func CyanStr(text string) string { return Colorize(text, Cyan) }

// Magenta returns text in magenta.
func MagentaStr(text string) string { return Colorize(text, Magenta) }

// DimStr returns dimmed text.
func DimStr(text string) string { return Colorize(text, Dim) }

// GetContextColor returns the ANSI color code for a context-usage percentage:
// red ≥85%, yellow ≥70%, green otherwise.
func GetContextColor(percent int) string {
	if percent >= 85 {
		return Red
	}
	if percent >= 70 {
		return Yellow
	}
	return Green
}

// ColoredBar generates a progress bar string for the given percentage.
// Filled segments use ▰ and empty segments use ▱.
func ColoredBar(percent, width int) string {
	if width <= 0 {
		width = 12
	}
	clamped := percent
	if clamped < 0 {
		clamped = 0
	}
	if clamped > 100 {
		clamped = 100
	}
	filled := (clamped * width) / 100
	// Round instead of truncate for better visual accuracy
	if (clamped*width)%100 >= 50 {
		filled++
	}
	if filled > width {
		filled = width
	}
	empty := width - filled

	filledStr := repeatStr("▰", filled)
	emptyStr := repeatStr("▱", empty)

	if !ShouldUseColor() {
		return filledStr + emptyStr
	}

	color := GetContextColor(percent)
	return color + filledStr + Dim + emptyStr + Reset
}

func repeatStr(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
