// Package plan provides plan naming, resolution, scaffolding, and validation.
// Ported from solon-hooks/internal/plan — keep in sync.
package plan

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"solon-core/internal/types"
)

// invalidFilenameChars matches characters invalid in filenames across all OSes.
var invalidFilenameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f\x7f]`)

// issuePatterns tries to extract an issue number from a branch name.
var issuePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?:issue|gh|fix|feat|bug)[/\-]?(\d+)`),
	regexp.MustCompile(`[/\-](\d+)[/\-]`),
	regexp.MustCompile(`#(\d+)`),
}

// SanitizeSlug removes invalid chars, converts to kebab-case, truncates at 100.
func SanitizeSlug(slug string) string {
	if slug == "" {
		return ""
	}
	s := invalidFilenameChars.ReplaceAllString(slug, "")
	s = regexp.MustCompile(`[^a-zA-Z0-9\-]`).ReplaceAllString(s, "-")
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 100 {
		s = s[:100]
	}
	return s
}

// FormatDate formats a date pattern string using current local time.
// Supported tokens: YYYY, YY, MM, DD, HH, mm, ss
func FormatDate(format string) string {
	now := time.Now()
	replacements := []struct{ token, value string }{
		{"YYYY", fmt.Sprintf("%04d", now.Year())},
		{"YY", fmt.Sprintf("%02d", now.Year()%100)},
		{"MM", fmt.Sprintf("%02d", int(now.Month()))},
		{"DD", fmt.Sprintf("%02d", now.Day())},
		{"HH", fmt.Sprintf("%02d", now.Hour())},
		{"mm", fmt.Sprintf("%02d", now.Minute())},
		{"ss", fmt.Sprintf("%02d", now.Second())},
	}
	result := format
	for _, r := range replacements {
		result = strings.ReplaceAll(result, r.token, r.value)
	}
	return result
}

// NormalizePath trims whitespace and trailing slashes from a path.
func NormalizePath(pathValue string) string {
	if pathValue == "" {
		return ""
	}
	normalized := strings.TrimSpace(pathValue)
	normalized = strings.TrimRight(normalized, "/\\")
	return normalized
}

// ExtractIssueFromBranch extracts an issue number string from a branch name.
func ExtractIssueFromBranch(branch string) string {
	if branch == "" {
		return ""
	}
	for _, re := range issuePatterns {
		if m := re.FindStringSubmatch(branch); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

// FormatIssueID formats an issue ID with the configured prefix.
func FormatIssueID(issueID string, planConfig types.PlanConfig) string {
	if issueID == "" {
		return ""
	}
	if planConfig.IssuePrefix != nil && *planConfig.IssuePrefix != "" {
		return *planConfig.IssuePrefix + issueID
	}
	return "#" + issueID
}

// ResolveNamingPattern resolves the naming pattern with date and optional issue.
// Keeps {slug} as a placeholder for callers to substitute.
func ResolveNamingPattern(planConfig types.PlanConfig, gitBranch string) string {
	formattedDate := FormatDate(planConfig.DateFormat)
	issueID := ExtractIssueFromBranch(gitBranch)

	var fullIssue string
	if issueID != "" && planConfig.IssuePrefix != nil && *planConfig.IssuePrefix != "" {
		fullIssue = *planConfig.IssuePrefix + issueID
	}

	pattern := planConfig.NamingFormat
	pattern = strings.ReplaceAll(pattern, "{date}", formattedDate)

	if fullIssue != "" {
		pattern = strings.ReplaceAll(pattern, "{issue}", fullIssue)
	} else {
		pattern = regexp.MustCompile(`-?\{issue\}-?`).ReplaceAllString(pattern, "-")
		pattern = regexp.MustCompile(`--+`).ReplaceAllString(pattern, "-")
	}

	pattern = strings.TrimLeft(pattern, "-")
	pattern = strings.TrimRight(pattern, "-")
	pattern = regexp.MustCompile(`-+(\{slug\})`).ReplaceAllString(pattern, "-$1")
	pattern = regexp.MustCompile(`(\{slug\})-+`).ReplaceAllString(pattern, "$1-")
	pattern = regexp.MustCompile(`--+`).ReplaceAllString(pattern, "-")

	if os.Getenv("SL_DEBUG") != "" {
		if !strings.Contains(pattern, "{slug}") {
			fmt.Fprintf(os.Stderr, "[naming] Warning: pattern missing {slug}\n")
		}
	}
	return pattern
}

// BuildPlanDirName resolves the full plan directory name with slug substituted.
func BuildPlanDirName(planConfig types.PlanConfig, gitBranch, slug string) string {
	pattern := ResolveNamingPattern(planConfig, gitBranch)
	sanitized := SanitizeSlug(slug)
	if sanitized == "" {
		sanitized = "untitled"
	}
	return strings.ReplaceAll(pattern, "{slug}", sanitized)
}
