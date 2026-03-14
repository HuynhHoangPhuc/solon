// Package truncation provides line-based output truncation for large tool outputs.
package truncation

import (
	"fmt"
	"strings"
)

const (
	DefaultMaxLines  = 200
	DefaultHeadLines = 50
	DefaultTailLines = 30
)

// TruncateOutput trims output to headLines+tailLines if it exceeds maxLines.
// Returns (result, wasChanged). Pure function, safe for concurrent use.
func TruncateOutput(output string, maxLines, headLines, tailLines int) (string, bool) {
	if output == "" {
		return output, false
	}
	lines := strings.Split(output, "\n")
	if len(lines) <= maxLines {
		return output, false
	}

	dropped := len(lines) - headLines - tailLines
	if dropped <= 0 {
		return output, false
	}

	head := strings.Join(lines[:headLines], "\n")
	tail := strings.Join(lines[len(lines)-tailLines:], "\n")
	marker := fmt.Sprintf("\n\n... [%d lines truncated for context efficiency] ...\n\n", dropped)

	return head + marker + tail, true
}
