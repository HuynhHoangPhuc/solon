// Scout checker — main facade for scout-block path and pattern validation.
package scout

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Build command allowlist patterns — allowed even if they reference blocked paths.
var buildCommandPattern = regexp.MustCompile(
	`^(npm|pnpm|yarn|bun)\s+([^\s]+\s+)*(run\s+)?(build|test|lint|dev|start|install|ci|add|remove|update|publish|pack|init|create|exec)`)

// Tool command pattern — JS/TS, Go, Rust, Java, .NET, containers, IaC, Python, Ruby, PHP, Deno, Elixir.
var toolCommandPattern = regexp.MustCompile(
	`^(\.\/)?(npx|pnpx|bunx|tsc|esbuild|vite|webpack|rollup|turbo|nx|jest|vitest|mocha|eslint|prettier|go|cargo|make|mvn|mvnw|gradle|gradlew|dotnet|docker|podman|kubectl|helm|terraform|ansible|bazel|cmake|sbt|flutter|swift|ant|ninja|meson|python3?|pip|uv|deno|bundle|rake|gem|php|composer|ruby|mix|elixir)`)

var venvExecutablePattern = regexp.MustCompile(`(^|[/\\])\.?venv[/\\](bin|Scripts)[/\\]`)

var venvCreationPattern = regexp.MustCompile(
	`^(python3?|py)\s+(-[\w.]+\s+)*-m\s+venv\s+|^uv\s+venv(\s|$)|^virtualenv\s+`)

// CheckOptions controls scout-block behavior.
type CheckOptions struct {
	SlignorePath       string
	ClaudeDir          string
	CheckBroadPatterns bool
}

// CheckResult is the result of a scout-block check.
type CheckResult struct {
	Blocked          bool
	Path             string
	Pattern          string
	Reason           string
	IsBroadPattern   bool
	Suggestions      []string
	IsAllowedCommand bool
}

// IsBuildCommand returns true if command matches the build/tool allowlist.
func IsBuildCommand(command string) bool {
	if command == "" {
		return false
	}
	trimmed := strings.TrimSpace(command)
	return buildCommandPattern.MatchString(trimmed) || toolCommandPattern.MatchString(trimmed)
}

// IsVenvExecutable returns true if command invokes a binary inside a venv directory.
func IsVenvExecutable(command string) bool {
	return command != "" && venvExecutablePattern.MatchString(command)
}

// IsVenvCreationCommand returns true if command creates a Python virtual environment.
func IsVenvCreationCommand(command string) bool {
	return command != "" && venvCreationPattern.MatchString(strings.TrimSpace(command))
}

// stripCommandPrefix removes environment variable prefixes and sudo/nice/etc wrappers.
func stripCommandPrefix(command string) string {
	if command == "" {
		return command
	}
	s := strings.TrimSpace(command)
	// Strip VAR=value prefixes
	envVarPrefix := regexp.MustCompile(`^(\w+=\S+\s+)+`)
	s = envVarPrefix.ReplaceAllString(s, "")
	// Strip common wrappers
	wrapperPrefix := regexp.MustCompile(`^(sudo|env|nice|nohup|time|timeout)\s+`)
	s = wrapperPrefix.ReplaceAllString(s, "")
	// Strip VAR=value again (may appear after wrapper)
	s = envVarPrefix.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// IsAllowedCommand returns true if command is on the build/tool/venv allowlist.
func IsAllowedCommand(command string) bool {
	stripped := stripCommandPrefix(command)
	return IsBuildCommand(stripped) || IsVenvExecutable(stripped) || IsVenvCreationCommand(stripped)
}

// SplitCompoundCommand splits a compound shell command on &&, ||, and ;.
func SplitCompoundCommand(command string) []string {
	if command == "" {
		return nil
	}
	parts := regexp.MustCompile(`\s*(?:&&|\|\||;)\s*`).Split(command, -1)
	var result []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			result = append(result, p)
		}
	}
	return result
}

// UnwrapShellExecutor strips `bash -c '...'`, `sh -c "..."`, or `eval "..."` wrappers.
func UnwrapShellExecutor(command string) string {
	if command == "" {
		return command
	}
	re := regexp.MustCompile(`^(?:(?:bash|sh|zsh)\s+-c|eval)\s+["'](.+)["']\s*$`)
	if m := re.FindStringSubmatch(strings.TrimSpace(command)); len(m) > 1 {
		return m[1]
	}
	return command
}

// CheckScoutBlock checks whether a tool call accesses blocked directories or uses broad patterns.
func CheckScoutBlock(toolName string, toolInput map[string]interface{}, opts CheckOptions) CheckResult {
	if opts.ClaudeDir == "" {
		cwd, _ := os.Getwd()
		opts.ClaudeDir = filepath.Join(cwd, ".claude")
	}
	checkBroad := opts.CheckBroadPatterns // zero value = false, callers set true explicitly

	// Unwrap shell executor wrappers
	if cmd, ok := toolInput["command"].(string); ok {
		unwrapped := UnwrapShellExecutor(cmd)
		if unwrapped != cmd {
			// Clone map with unwrapped command
			newInput := make(map[string]interface{}, len(toolInput))
			for k, v := range toolInput {
				newInput[k] = v
			}
			newInput["command"] = unwrapped
			toolInput = newInput
		}
	}

	// Split compound commands — check each sub-command independently
	if cmd, ok := toolInput["command"].(string); ok {
		subCmds := SplitCompoundCommand(cmd)
		var nonAllowed []string
		for _, sub := range subCmds {
			if !IsAllowedCommand(strings.TrimSpace(sub)) {
				nonAllowed = append(nonAllowed, sub)
			}
		}
		if len(subCmds) > 0 && len(nonAllowed) == 0 {
			return CheckResult{Blocked: false, IsAllowedCommand: true}
		}
		if len(nonAllowed) < len(subCmds) {
			// Clone with only non-allowed sub-commands
			newInput := make(map[string]interface{}, len(toolInput))
			for k, v := range toolInput {
				newInput[k] = v
			}
			newInput["command"] = strings.Join(nonAllowed, " ; ")
			toolInput = newInput
		}
	}

	// Check for overly broad glob patterns
	if checkBroad && (toolName == "Glob" || toolInput["pattern"] != nil) {
		pattern, _ := toolInput["pattern"].(string)
		basePath, _ := toolInput["path"].(string)
		broadResult := DetectBroadPatternIssue(pattern, basePath)
		if broadResult.Blocked {
			return CheckResult{
				Blocked:        true,
				IsBroadPattern: true,
				Pattern:        pattern,
				Reason:         broadResult.Reason,
				Suggestions:    broadResult.Suggestions,
			}
		}
	}

	// Load .slignore patterns and check extracted paths
	slignorePath := opts.SlignorePath
	if slignorePath == "" {
		slignorePath = filepath.Join(opts.ClaudeDir, ".slignore")
	}
	patterns := LoadPatterns(slignorePath)
	matcher := CreateMatcher(patterns)
	extractedPaths := ExtractFromToolInput(toolInput)

	if len(extractedPaths) == 0 {
		return CheckResult{Blocked: false}
	}

	for _, p := range extractedPaths {
		result := MatchPath(matcher, p)
		if result.Blocked {
			return CheckResult{
				Blocked: true,
				Path:    p,
				Pattern: result.Pattern,
				Reason:  "Path matches blocked pattern: " + result.Pattern,
			}
		}
	}

	return CheckResult{Blocked: false}
}
