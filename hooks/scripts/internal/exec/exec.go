// Package exec provides safe command execution helpers with timeouts.
// All functions return empty string on error (fail-open, never panic).
package exec

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

const defaultTimeoutMs = 5000

// gitAllowlist is the set of read-only git commands permitted by ExecGitSafe.
var gitAllowlist = map[string]bool{
	"git branch --show-current":       true,
	"git rev-parse --abbrev-ref HEAD": true,
	"git rev-parse --show-toplevel":   true,
}

// ExecSafe runs a shell command via /bin/sh with a timeout.
// Returns trimmed stdout or "" on any error.
func ExecSafe(cmd string, cwd string, timeoutMs int) string {
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

// ExecFileSafe runs a binary directly with args (no shell injection risk).
// Returns trimmed stdout or "" on any error.
func ExecFileSafe(binary string, args []string, timeoutMs int) string {
	if timeoutMs <= 0 {
		timeoutMs = defaultTimeoutMs
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	c := exec.CommandContext(ctx, binary, args...)
	out, err := c.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ExecGitSafe runs only whitelisted read-only git commands.
// Returns trimmed stdout or "" if cmd is not in the allowlist or fails.
func ExecGitSafe(cmd string, cwd string) string {
	if !gitAllowlist[cmd] {
		return ""
	}
	return ExecSafe(cmd, cwd, defaultTimeoutMs)
}
