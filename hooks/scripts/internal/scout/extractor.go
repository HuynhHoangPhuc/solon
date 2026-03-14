// Path extraction from Claude Code tool inputs (Read, Glob, Bash, etc.)
package scout

import (
	"regexp"
	"strings"
)

// excludeFlags are CLI flags whose following value should NOT be checked as a path.
var excludeFlags = map[string]bool{
	"--exclude": true, "--ignore": true, "--skip": true, "--prune": true,
	"-x": true, "-path": true, "--exclude-dir": true,
}

// filesystemCommands are shell commands where bare directory names are treated as paths.
var filesystemCommands = map[string]bool{
	"cd": true, "ls": true, "cat": true, "head": true, "tail": true,
	"less": true, "more": true, "rm": true, "cp": true, "mv": true,
	"find": true, "touch": true, "mkdir": true, "rmdir": true,
	"stat": true, "file": true, "du": true, "tree": true,
	"chmod": true, "chown": true, "ln": true, "readlink": true,
	"realpath": true, "wc": true, "tee": true, "tar": true,
	"zip": true, "unzip": true, "open": true, "code": true,
	"vim": true, "nano": true, "bat": true, "rsync": true,
	"scp": true, "diff": true,
}

// blockedDirNames matches the DEFAULT_PATTERNS directory names.
var blockedDirNames = map[string]bool{
	"node_modules": true, "__pycache__": true, ".git": true,
	"dist": true, "build": true, ".next": true, ".nuxt": true,
	".venv": true, "venv": true, "vendor": true, "target": true, "coverage": true,
}

// commandKeywords are shell builtins and common tool names that are NOT paths.
var commandKeywords = map[string]bool{
	"echo": true, "cat": true, "ls": true, "cd": true, "rm": true, "cp": true,
	"mv": true, "find": true, "grep": true, "head": true, "tail": true,
	"wc": true, "du": true, "tree": true, "touch": true, "mkdir": true,
	"rmdir": true, "pwd": true, "which": true, "env": true, "export": true,
	"source": true, "bash": true, "sh": true, "zsh": true, "true": true,
	"false": true, "test": true, "xargs": true, "tee": true, "sort": true,
	"uniq": true, "cut": true, "tr": true, "sed": true, "awk": true,
	"diff": true, "chmod": true, "chown": true, "ln": true, "file": true,
	"npm": true, "pnpm": true, "yarn": true, "bun": true, "npx": true,
	"pnpx": true, "bunx": true, "node": true,
	"run": true, "build": true, "lint": true, "dev": true,
	"start": true, "install": true, "ci": true, "exec": true,
	"add": true, "remove": true, "update": true, "publish": true,
	"pack": true, "init": true, "create": true,
	"tsc": true, "esbuild": true, "vite": true, "webpack": true,
	"rollup": true, "turbo": true, "nx": true,
	"jest": true, "vitest": true, "mocha": true, "eslint": true, "prettier": true,
	"git": true, "commit": true, "push": true, "pull": true, "merge": true,
	"rebase": true, "checkout": true, "branch": true,
	"status": true, "log": true, "reset": true, "stash": true, "fetch": true,
	"clone": true,
	"docker": true, "compose": true, "up": true, "down": true, "ps": true,
	"logs": true, "container": true, "image": true,
	"sudo": true, "time": true, "timeout": true, "watch": true, "make": true,
	"cargo": true, "python": true, "python3": true, "pip": true,
	"ruby": true, "gem": true, "go": true, "rust": true, "java": true,
	"javac": true, "mvn": true, "gradle": true,
}

var sedAwkRegex = regexp.MustCompile(`^s[/|@#,]`)
var pathLikeExtension = regexp.MustCompile(`\.\w{1,6}$`)
var pathLikeSegment = regexp.MustCompile(`^[a-zA-Z0-9_\-]+/`)

// ExtractFromToolInput extracts all path-like strings from a tool_input map.
func ExtractFromToolInput(toolInput map[string]interface{}) []string {
	var paths []string
	if toolInput == nil {
		return paths
	}
	for _, param := range []string{"file_path", "path", "pattern"} {
		if v, ok := toolInput[param].(string); ok && v != "" {
			if n := NormalizeExtractedPath(v); n != "" {
				paths = append(paths, n)
			}
		}
	}
	if cmd, ok := toolInput["command"].(string); ok && cmd != "" {
		paths = append(paths, ExtractFromCommand(cmd)...)
	}
	// Remove empty strings
	result := paths[:0]
	for _, p := range paths {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// ExtractFromCommand extracts path-like tokens from a Bash command string.
func ExtractFromCommand(command string) []string {
	if command == "" {
		return nil
	}
	var paths []string

	// Extract quoted strings first (preserves spaces in paths)
	quotedRe := regexp.MustCompile(`["']([^"']+)["']`)
	for _, m := range quotedRe.FindAllStringSubmatch(command, -1) {
		content := m[1]
		if sedAwkRegex.MatchString(content) {
			continue
		}
		if LooksLikePath(content) {
			paths = append(paths, NormalizeExtractedPath(content))
		}
	}

	// Remove quoted regions and tokenize
	withoutQuotes := quotedRe.ReplaceAllString(command, " ")
	tokens := strings.Fields(withoutQuotes)

	var commandName string
	isFsCommand := false
	skipNextToken := false
	var heredocDelimiter string
	nextIsHeredocDelimiter := false

	for _, token := range tokens {
		if nextIsHeredocDelimiter {
			heredocDelimiter = strings.Trim(token, `'"`)
			nextIsHeredocDelimiter = false
			continue
		}
		if heredocDelimiter != "" {
			if token == heredocDelimiter {
				heredocDelimiter = ""
			}
			continue
		}
		if strings.HasPrefix(token, "<<") && len(token) > 2 {
			raw := strings.TrimPrefix(token, "<<")
			raw = strings.TrimPrefix(raw, "-")
			heredocDelimiter = strings.Trim(raw, `'"`)
			continue
		}
		if token == "<<" || token == "<<-" {
			nextIsHeredocDelimiter = true
			continue
		}
		if skipNextToken {
			skipNextToken = false
			continue
		}
		if token == "&&" || token == ";" || strings.HasPrefix(token, "|") {
			commandName = ""
			isFsCommand = false
			continue
		}
		if IsSkippableToken(token) {
			if excludeFlags[token] {
				skipNextToken = true
			}
			continue
		}
		if commandName == "" {
			commandName = strings.ToLower(token)
			isFsCommand = filesystemCommands[commandName]
			if IsCommandKeyword(token) || isFsCommand {
				continue
			}
		}
		if isFsCommand && IsBlockedDirName(token) {
			paths = append(paths, NormalizeExtractedPath(token))
			continue
		}
		if IsCommandKeyword(token) {
			continue
		}
		if LooksLikePath(token) {
			paths = append(paths, NormalizeExtractedPath(token))
		}
	}
	return paths
}

// IsBlockedDirName returns true if token is a known blocked directory name.
func IsBlockedDirName(token string) bool {
	return blockedDirNames[token]
}

// LooksLikePath returns true if str resembles a filesystem path.
func LooksLikePath(str string) bool {
	if str == "" || len(str) < 2 {
		return false
	}
	if strings.ContainsAny(str, "/\\") {
		return true
	}
	if strings.HasPrefix(str, "./") || strings.HasPrefix(str, "../") {
		return true
	}
	if pathLikeExtension.MatchString(str) {
		return true
	}
	if pathLikeSegment.MatchString(str) {
		return true
	}
	return false
}

// IsSkippableToken returns true for tokens that are flags, operators, or pure numbers.
func IsSkippableToken(token string) bool {
	if strings.HasPrefix(token, "-") {
		return true
	}
	for _, op := range []string{"||", "&&", ">", ">>", "<", "<<"} {
		if token == op {
			return true
		}
	}
	if strings.HasPrefix(token, "|") || strings.HasPrefix(token, ">") ||
		strings.HasPrefix(token, "<") || strings.HasPrefix(token, "&") {
		return true
	}
	allDigits := true
	for _, c := range token {
		if c < '0' || c > '9' {
			allDigits = false
			break
		}
	}
	return len(token) > 0 && allDigits
}

// IsCommandKeyword returns true if token is a known shell command/keyword.
func IsCommandKeyword(token string) bool {
	return commandKeywords[strings.ToLower(token)]
}

// NormalizeExtractedPath cleans a path token: strips quotes, backticks, trailing slashes.
func NormalizeExtractedPath(p string) string {
	if p == "" {
		return ""
	}
	normalized := strings.TrimSpace(p)
	// Strip matching surrounding quotes
	if len(normalized) >= 2 {
		if (normalized[0] == '"' && normalized[len(normalized)-1] == '"') ||
			(normalized[0] == '\'' && normalized[len(normalized)-1] == '\'') {
			normalized = normalized[1 : len(normalized)-1]
		}
	}
	// Strip shell meta-chars from edges
	normalized = strings.TrimLeft(normalized, "`({[")
	normalized = strings.TrimRight(normalized, "`)}];")
	// Normalize separators
	normalized = strings.ReplaceAll(normalized, `\`, "/")
	// Remove trailing slash (unless root)
	if len(normalized) > 1 && strings.HasSuffix(normalized, "/") {
		normalized = normalized[:len(normalized)-1]
	}
	return normalized
}
