// Package privacy provides privacy-sensitive file detection for the privacy-block hook.
package privacy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ApprovedPrefix is the prefix that marks a file path as user-approved.
const ApprovedPrefix = "APPROVED:"

// safePatterns are file patterns exempt from privacy checks.
var safePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\.example$`),
	regexp.MustCompile(`(?i)\.sample$`),
	regexp.MustCompile(`(?i)\.template$`),
}

// privacyPatterns match filenames/paths that contain sensitive data.
var privacyPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\.env$`),
	regexp.MustCompile(`^\.env\.`),
	regexp.MustCompile(`\.env$`),
	regexp.MustCompile(`/\.env\.`),
	regexp.MustCompile(`(?i)credentials`),
	regexp.MustCompile(`(?i)secrets?\.ya?ml$`),
	regexp.MustCompile(`\.pem$`),
	regexp.MustCompile(`\.key$`),
	regexp.MustCompile(`id_rsa`),
	regexp.MustCompile(`id_ed25519`),
}

// PathEntry holds a path value and the field it was extracted from.
type PathEntry struct {
	Value string
	Field string
}

// PrivacyOpts controls privacy check behavior.
type PrivacyOpts struct {
	Disabled  bool
	ConfigDir string
	AllowBash bool // default true
}

// CheckResult is the result of a privacy check.
type CheckResult struct {
	Blocked    bool
	FilePath   string
	Reason     string
	Approved   bool
	IsBash     bool
	Suspicious bool
	PromptData map[string]interface{}
}

// IsSafeFile returns true if the file matches a safe pattern (e.g. .env.example).
func IsSafeFile(testPath string) bool {
	if testPath == "" {
		return false
	}
	base := filepath.Base(testPath)
	for _, p := range safePatterns {
		if p.MatchString(base) {
			return true
		}
	}
	return false
}

// HasApprovalPrefix returns true if the path starts with "APPROVED:".
func HasApprovalPrefix(testPath string) bool {
	return strings.HasPrefix(testPath, ApprovedPrefix)
}

// StripApprovalPrefix removes the "APPROVED:" prefix if present.
func StripApprovalPrefix(testPath string) string {
	if HasApprovalPrefix(testPath) {
		return testPath[len(ApprovedPrefix):]
	}
	return testPath
}

// IsSuspiciousPath returns true for paths with traversal or absolute references.
func IsSuspiciousPath(strippedPath string) bool {
	return strings.Contains(strippedPath, "..") || filepath.IsAbs(strippedPath)
}

// IsPrivacySensitive returns true if testPath matches a privacy-sensitive pattern.
func IsPrivacySensitive(testPath string) bool {
	if testPath == "" {
		return false
	}
	clean := StripApprovalPrefix(testPath)
	normalized := strings.ReplaceAll(clean, `\`, "/")
	// Best-effort URL decode
	if decoded, err := urlDecode(normalized); err == nil {
		normalized = decoded
	}
	if IsSafeFile(normalized) {
		return false
	}
	base := filepath.Base(normalized)
	for _, p := range privacyPatterns {
		if p.MatchString(base) || p.MatchString(normalized) {
			return true
		}
	}
	return false
}

// ExtractPaths extracts path values from a tool_input map.
func ExtractPaths(toolInput map[string]interface{}) []PathEntry {
	var paths []PathEntry
	if toolInput == nil {
		return paths
	}
	for _, field := range []string{"file_path", "path", "pattern"} {
		if v, ok := toolInput[field].(string); ok && v != "" {
			paths = append(paths, PathEntry{Value: v, Field: field})
		}
	}
	if cmd, ok := toolInput["command"].(string); ok && cmd != "" {
		// Check for APPROVED: prefixed paths first
		approvedRe := regexp.MustCompile(`APPROVED:[^\s]+`)
		approved := approvedRe.FindAllString(cmd, -1)
		if len(approved) > 0 {
			for _, p := range approved {
				paths = append(paths, PathEntry{Value: p, Field: "command"})
			}
		} else {
			// Extract .env references
			envRe := regexp.MustCompile(`\.env[^\s]*`)
			for _, m := range envRe.FindAllString(cmd, -1) {
				paths = append(paths, PathEntry{Value: m, Field: "command"})
			}
			// Variable assignments like VAR=.env.local
			varRe := regexp.MustCompile(`\w+=[^\s]*\.env[^\s]*`)
			for _, a := range varRe.FindAllString(cmd, -1) {
				parts := strings.SplitN(a, "=", 2)
				if len(parts) == 2 && parts[1] != "" {
					paths = append(paths, PathEntry{Value: parts[1], Field: "command"})
				}
			}
			// Command substitution $(.env...)
			cmdSubstRe := regexp.MustCompile(`\$\([^)]*?(\.env[^\s)]*)[^)]*\)`)
			for _, m := range cmdSubstRe.FindAllStringSubmatch(cmd, -1) {
				if len(m) > 1 {
					paths = append(paths, PathEntry{Value: m[1], Field: "command"})
				}
			}
		}
	}
	// Filter out empty values
	filtered := paths[:0]
	for _, p := range paths {
		if p.Value != "" {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// isPrivacyBlockDisabled reads the local config to check if privacy-block is disabled.
func isPrivacyBlockDisabled(configDir string) bool {
	var configPath string
	if configDir != "" {
		configPath = filepath.Join(configDir, ".sl.json")
	} else {
		cwd, _ := os.Getwd()
		configPath = filepath.Join(cwd, ".claude", ".sl.json")
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return false
	}
	v, ok := cfg["privacyBlock"]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && !b
}

// BuildPromptData constructs the prompt data for user approval.
func BuildPromptData(filePath string) map[string]interface{} {
	base := filepath.Base(filePath)
	return map[string]interface{}{
		"type":     "PRIVACY_PROMPT",
		"file":     filePath,
		"basename": base,
		"question": map[string]interface{}{
			"header": "File Access",
			"text":   fmt.Sprintf("I need to read %q which may contain sensitive data (API keys, passwords, tokens). Do you approve?", base),
			"options": []map[string]string{
				{"label": "Yes, approve access", "description": fmt.Sprintf("Allow reading %s this time", base)},
				{"label": "No, skip this file", "description": "Continue without accessing this file"},
			},
		},
	}
}

// CheckPrivacy checks whether a tool call accesses privacy-sensitive files.
func CheckPrivacy(toolName string, toolInput map[string]interface{}, opts PrivacyOpts) CheckResult {
	if opts.Disabled || isPrivacyBlockDisabled(opts.ConfigDir) {
		return CheckResult{Blocked: false}
	}

	allowBash := true
	if !opts.AllowBash {
		allowBash = false
	}
	isBashTool := toolName == "Bash"
	paths := ExtractPaths(toolInput)

	for _, entry := range paths {
		if !IsPrivacySensitive(entry.Value) {
			continue
		}
		if HasApprovalPrefix(entry.Value) {
			stripped := StripApprovalPrefix(entry.Value)
			return CheckResult{
				Blocked:    false,
				Approved:   true,
				FilePath:   stripped,
				Suspicious: IsSuspiciousPath(stripped),
			}
		}
		if isBashTool && allowBash {
			return CheckResult{
				Blocked:  false,
				IsBash:   true,
				FilePath: entry.Value,
				Reason:   fmt.Sprintf("Bash command accesses sensitive file: %s", entry.Value),
			}
		}
		return CheckResult{
			Blocked:    true,
			FilePath:   entry.Value,
			Reason:     "Sensitive file access requires user approval",
			PromptData: BuildPromptData(entry.Value),
		}
	}
	return CheckResult{Blocked: false}
}

// urlDecode performs a simple percent-decode (best effort).
func urlDecode(s string) (string, error) {
	// Use a simple state machine for %XX sequences
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			var val byte
			_, err := fmt.Sscanf(s[i+1:i+3], "%02x", &val)
			if err == nil {
				b.WriteByte(val)
				i += 2
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String(), nil
}
