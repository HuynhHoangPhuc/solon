// Broad glob pattern detection — blocks patterns that would fill context with too many files.
package scout

import (
	"fmt"
	"regexp"
	"strings"
)

// broadPatternRegexes matches glob patterns that are too broad.
var broadPatternRegexes = []*regexp.Regexp{
	regexp.MustCompile(`^\*\*$`),
	regexp.MustCompile(`^\*$`),
	regexp.MustCompile(`^\*\*/\*$`),
	regexp.MustCompile(`^\*\*/\.\*$`),
	regexp.MustCompile(`^\*\.\w+$`),
	regexp.MustCompile(`^\*\.\{[^}]+\}$`),
	regexp.MustCompile(`^\*\*/\*\.\w+$`),
	regexp.MustCompile(`^\*\*/\*\.\{[^}]+\}$`),
}

// specificDirs are well-scoped source directories that make broad patterns acceptable.
var specificDirs = []string{
	"src", "lib", "app", "apps", "packages", "components", "pages",
	"api", "server", "client", "web", "mobile", "shared", "common",
	"utils", "helpers", "services", "hooks", "store", "routes",
	"models", "controllers", "views", "tests", "__tests__", "spec",
}

// highRiskIndicators match paths that are likely project root or worktree root.
var highRiskIndicators = []*regexp.Regexp{
	regexp.MustCompile(`/worktrees/[^/]+/?$`),
	regexp.MustCompile(`^\.?/?$`),
	regexp.MustCompile(`^[^/]+/?$`),
}

// BroadPatternResult is the result of a broad-pattern check.
type BroadPatternResult struct {
	Blocked     bool
	Reason      string
	Pattern     string
	Suggestions []string
}

// IsBroadPattern returns true if pattern matches one of the broad-pattern regexes.
func IsBroadPattern(pattern string) bool {
	if pattern == "" {
		return false
	}
	normalized := strings.TrimSpace(pattern)
	for _, re := range broadPatternRegexes {
		if re.MatchString(normalized) {
			return true
		}
	}
	return false
}

// HasSpecificDirectory returns true if pattern starts with a known specific directory.
func HasSpecificDirectory(pattern string) bool {
	if pattern == "" {
		return false
	}
	for _, dir := range specificDirs {
		if strings.HasPrefix(pattern, dir+"/") || strings.HasPrefix(pattern, "./"+dir+"/") {
			return true
		}
	}
	// A non-wildcard first segment counts as specific
	firstSeg := strings.SplitN(pattern, "/", 2)[0]
	if firstSeg != "" && !strings.ContainsAny(firstSeg, "*") && firstSeg != "." {
		return true
	}
	return false
}

// IsHighLevelPath returns true if basePath is likely a project or worktree root.
func IsHighLevelPath(basePath string) bool {
	if basePath == "" {
		return true
	}
	normalized := strings.ReplaceAll(basePath, `\`, "/")
	for _, re := range highRiskIndicators {
		if re.MatchString(normalized) {
			return true
		}
	}
	segments := []string{}
	for _, s := range strings.Split(normalized, "/") {
		if s != "" && s != "." {
			segments = append(segments, s)
		}
	}
	if len(segments) <= 1 {
		return true
	}
	for _, dir := range specificDirs {
		if strings.Contains(normalized, "/"+dir+"/") ||
			strings.Contains(normalized, "/"+dir) ||
			strings.HasPrefix(normalized, dir+"/") ||
			normalized == dir {
			return false
		}
	}
	return true
}

// SuggestSpecificPatterns generates more targeted glob alternatives.
func SuggestSpecificPatterns(pattern string) []string {
	var suggestions []string
	extRe := regexp.MustCompile(`\*\.(\{[^}]+\}|\w+)$`)
	ext := ""
	if m := extRe.FindStringSubmatch(pattern); len(m) > 1 {
		ext = m[1]
	}
	commonDirs := []string{"src", "lib", "app", "components"}
	if strings.Contains(pattern, ".ts") || strings.Contains(pattern, "{ts") {
		suggestions = append(suggestions, "src/**/*.ts", "src/**/*.tsx")
	}
	if strings.Contains(pattern, ".js") || strings.Contains(pattern, "{js") {
		suggestions = append(suggestions, "src/**/*.js", "lib/**/*.js")
	}
	for _, dir := range commonDirs {
		if ext != "" {
			suggestions = append(suggestions, fmt.Sprintf("%s/**/*.%s", dir, ext))
		} else {
			suggestions = append(suggestions, fmt.Sprintf("%s/**/*", dir))
		}
	}
	if len(suggestions) > 4 {
		suggestions = suggestions[:4]
	}
	return suggestions
}

// DetectBroadPatternIssue checks if a Glob tool call uses an overly broad pattern.
func DetectBroadPatternIssue(pattern, basePath string) BroadPatternResult {
	if pattern == "" {
		return BroadPatternResult{Blocked: false}
	}
	if HasSpecificDirectory(pattern) {
		return BroadPatternResult{Blocked: false}
	}
	if !IsBroadPattern(pattern) {
		return BroadPatternResult{Blocked: false}
	}
	if !IsHighLevelPath(basePath) {
		return BroadPatternResult{Blocked: false}
	}
	location := basePath
	if location == "" {
		location = "project root"
	}
	return BroadPatternResult{
		Blocked:     true,
		Reason:      fmt.Sprintf("Pattern '%s' is too broad for %s", pattern, location),
		Pattern:     pattern,
		Suggestions: SuggestSpecificPatterns(pattern),
	}
}
