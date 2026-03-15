// Package exec provides safe shell command execution with timeouts.
package exec

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

const defaultTimeoutMs = 3000

// gitAllowlist is the set of read-only git commands permitted by GitSafe.
var gitAllowlist = map[string]bool{
	"git branch --show-current":       true,
	"git rev-parse --abbrev-ref HEAD": true,
	"git rev-parse --show-toplevel":   true,
}

// Safe runs a shell command via /bin/sh with a timeout.
// Returns trimmed stdout or "" on any error.
func Safe(cmd string, cwd string, timeoutMs int) string {
	if timeoutMs <= 0 {
		timeoutMs = defaultTimeoutMs
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	c := exec.CommandContext(ctx, "/bin/sh", "-c", cmd)
	if cwd != "" {
		c.Dir = cwd
	}
	out, err := c.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// GitSafe runs a whitelisted read-only git command.
func GitSafe(cmd string, cwd string) string {
	if !gitAllowlist[cmd] {
		return ""
	}
	return Safe(cmd, cwd, defaultTimeoutMs)
}
