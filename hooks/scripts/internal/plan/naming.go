// Package plan provides naming utilities and plan path resolution.
package plan

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"solon-hooks/internal/hookio"
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
	// Remove invalid filename chars
	s := invalidFilenameChars.ReplaceAllString(slug, "")
	// Replace non-alphanumeric (except hyphen) with hyphen
	s = regexp.MustCompile(`[^a-zA-Z0-9\-]`).ReplaceAllString(s, "-")
	// Collapse multiple hyphens
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")
	// Trim leading/trailing hyphens
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
// Returns empty string for empty or whitespace-only input.
func NormalizePath(pathValue string) string {
	if pathValue == "" {
		return ""
	}
	normalized := strings.TrimSpace(pathValue)
	normalized = strings.TrimRight(normalized, "/\\")
	return normalized
}

// ValidateNamingPatternResult holds the result of pattern validation.
type ValidateNamingPatternResult struct {
	Valid bool
	Error string
}

// ValidateNamingPattern checks that a naming pattern contains {slug} and has
// no unresolved placeholders after slug removal.
func ValidateNamingPattern(pattern string) ValidateNamingPatternResult {
	if pattern == "" {
		return ValidateNamingPatternResult{false, "Pattern is empty or not a string"}
	}
	withoutSlug := strings.ReplaceAll(pattern, "{slug}", "")
	withoutSlug = regexp.MustCompile(`-+`).ReplaceAllString(withoutSlug, "-")
	withoutSlug = strings.Trim(withoutSlug, "-")
	if withoutSlug == "" {
		return ValidateNamingPatternResult{false, "Pattern resolves to empty after removing {slug}"}
	}
	if m := regexp.MustCompile(`\{[^}]+\}`).FindString(withoutSlug); m != "" {
		return ValidateNamingPatternResult{false, "Unresolved placeholder: " + m}
	}
	if !strings.Contains(pattern, "{slug}") {
		return ValidateNamingPatternResult{false, "Pattern must contain {slug} placeholder"}
	}
	return ValidateNamingPatternResult{Valid: true}
}

// ExtractIssueFromBranch extracts an issue number string from a branch name.
// Returns empty string if no match.
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
// Returns empty string if issueID is empty.
func FormatIssueID(issueID string, planConfig hookio.PlanConfig) string {
	if issueID == "" {
		return ""
	}
	if planConfig.IssuePrefix != nil && *planConfig.IssuePrefix != "" {
		return *planConfig.IssuePrefix + issueID
	}
	return "#" + issueID
}

// ResolveNamingPattern resolves the naming pattern with date and optional issue prefix.
// Keeps {slug} as a placeholder for agents to substitute.
func ResolveNamingPattern(planConfig hookio.PlanConfig, gitBranch string) string {
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

	// Clean up edge cases around {slug}
	pattern = strings.TrimLeft(pattern, "-")
	pattern = strings.TrimRight(pattern, "-")
	pattern = regexp.MustCompile(`-+(\{slug\})`).ReplaceAllString(pattern, "-$1")
	pattern = regexp.MustCompile(`(\{slug\})-+`).ReplaceAllString(pattern, "$1-")
	pattern = regexp.MustCompile(`--+`).ReplaceAllString(pattern, "-")

	if os.Getenv("SL_DEBUG") != "" {
		if v := ValidateNamingPattern(pattern); !v.Valid {
			fmt.Fprintf(os.Stderr, "[naming] Warning: %s\n", v.Error)
		}
	}

	return pattern
}
