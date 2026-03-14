// Package scout provides path blocking via .slignore pattern matching and broad-pattern detection.
package scout

import (
	"os"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

// DefaultPatterns are used when no .slignore file exists or it is empty.
var DefaultPatterns = []string{
	"node_modules", "dist", "build", ".next", ".nuxt",
	"__pycache__", ".venv", "venv",
	"vendor",
	"target",
	".git",
	"coverage",
}

// Matcher holds compiled gitignore patterns for fast path matching.
type Matcher struct {
	ig       *gitignore.GitIgnore
	patterns []string // normalized patterns passed to gitignore
	original []string // original patterns from the file
}

// MatchResult is returned by MatchPath.
type MatchResult struct {
	Blocked bool
	Pattern string // which original pattern matched
}

// LoadPatterns reads .slignore patterns from slignorePath.
// Falls back to DefaultPatterns if the file is missing, empty, or unreadable.
func LoadPatterns(slignorePath string) []string {
	if slignorePath == "" {
		return DefaultPatterns
	}
	data, err := os.ReadFile(slignorePath)
	if err != nil {
		return DefaultPatterns
	}
	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	if len(patterns) == 0 {
		return DefaultPatterns
	}
	return patterns
}

// CreateMatcher compiles patterns into a gitignore-compatible matcher.
// Single names (no / or *) are expanded to match anywhere in the path tree.
func CreateMatcher(patterns []string) *Matcher {
	var normalized []string
	for _, p := range patterns {
		if strings.HasPrefix(p, "!") {
			inner := p[1:]
			if strings.ContainsAny(inner, "/*") {
				normalized = append(normalized, p)
			} else {
				normalized = append(normalized, "!**/"+inner)
				normalized = append(normalized, "!**/"+inner+"/**")
			}
		} else {
			if strings.ContainsAny(p, "/*") {
				normalized = append(normalized, p)
			} else {
				normalized = append(normalized,
					"**/"+p,
					"**/"+p+"/**",
					p,
					p+"/**",
				)
			}
		}
	}

	ig := gitignore.CompileIgnoreLines(normalized...)
	return &Matcher{ig: ig, patterns: normalized, original: patterns}
}

// MatchPath checks whether testPath is blocked by the matcher.
func MatchPath(m *Matcher, testPath string) MatchResult {
	if testPath == "" || m == nil {
		return MatchResult{Blocked: false}
	}

	// Normalize separators and strip leading ./ / ../
	normalized := strings.ReplaceAll(testPath, `\`, "/")
	normalized = strings.TrimPrefix(normalized, "./")
	for strings.HasPrefix(normalized, "/") {
		normalized = normalized[1:]
	}
	for strings.HasPrefix(normalized, "../") {
		normalized = normalized[3:]
	}
	if normalized == "" {
		return MatchResult{Blocked: false}
	}

	if m.ig.MatchesPath(normalized) {
		return MatchResult{
			Blocked: true,
			Pattern: findMatchingPattern(m.original, normalized),
		}
	}
	return MatchResult{Blocked: false}
}

// findMatchingPattern finds which original pattern matched testPath (for error messages).
func findMatchingPattern(originals []string, testPath string) string {
	for _, p := range originals {
		if strings.HasPrefix(p, "!") {
			continue
		}
		// Quick substring check
		simplified := strings.ReplaceAll(strings.ReplaceAll(p, "**", ""), "*", "")
		if simplified != "" && strings.Contains(testPath, simplified) {
			return p
		}
		// Compile individually and test
		var check []string
		if strings.ContainsAny(p, "/*") {
			check = []string{p}
		} else {
			check = []string{"**/" + p, "**/" + p + "/**", p, p + "/**"}
		}
		ig := gitignore.CompileIgnoreLines(check...)
		if ig.MatchesPath(testPath) {
			return p
		}
	}
	// Fallback: first non-negation pattern
	for _, p := range originals {
		if !strings.HasPrefix(p, "!") {
			return p
		}
	}
	return "unknown"
}
