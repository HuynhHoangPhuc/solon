package plan

import (
	"testing"

	"solon-hooks/internal/hookio"
)

// TestSanitizeSlug tests the SanitizeSlug function.
func TestSanitizeSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		checkLen bool
		checkVal string
	}{
		{"removes invalid chars", "invalid<>chars", false, "invalidchars"},
		{"converts to kebab-case", "hello world", false, "hello-world"},
		{"collapses multiple hyphens", "hello---world", false, "hello-world"},
		{"trims leading/trailing hyphens", "-hello-world-", false, "hello-world"},
		{"truncates at 100 chars", "a" + string(make([]byte, 150)) + "z", true, ""},
		{"handles empty string", "", false, ""},
		{"converts underscores to hyphens", "hello_world_test", false, "hello-world-test"},
		{"removes special chars", "hello@world#test", false, "hello-world-test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeSlug(tt.input)
			if tt.checkLen {
				// Just verify length constraint
				if len(got) > 100 {
					t.Errorf("SanitizeSlug(%q) returned length %d, max is 100", tt.input, len(got))
				}
			} else if len(tt.checkVal) > 0 && got != tt.checkVal {
				t.Errorf("SanitizeSlug(%q) = %q, want %q", tt.input, got, tt.checkVal)
			}
		})
	}
}

// TestFormatDate tests the FormatDate function.
func TestFormatDate(t *testing.T) {
	// Test with known patterns - we'll just verify the output format is reasonable
	tests := []struct {
		name   string
		format string
		check  func(string) bool
	}{
		{
			"expands YYYY",
			"YYYY-MM-DD",
			func(s string) bool { return len(s) == 10 && s[4] == '-' && s[7] == '-' },
		},
		{
			"expands YY",
			"YY-MM-DD",
			func(s string) bool { return len(s) == 8 && s[2] == '-' && s[5] == '-' },
		},
		{
			"expands HH:mm",
			"HH:mm",
			func(s string) bool { return len(s) == 5 && s[2] == ':' },
		},
		{
			"returns empty for empty pattern",
			"",
			func(s string) bool { return s == "" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDate(tt.format)
			if !tt.check(got) {
				t.Errorf("FormatDate(%q) = %q, failed check", tt.format, got)
			}
		})
	}
}

// TestNormalizePath tests path normalization.
func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"trims whitespace", "  path/to/file  ", "path/to/file"},
		{"removes trailing slashes", "path/to/dir/", "path/to/dir"},
		{"removes trailing backslashes", "path\\to\\dir\\", "path\\to\\dir"},
		{"empty string", "", ""},
		{"whitespace only", "   ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizePath(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestValidateNamingPattern tests naming pattern validation.
func TestValidateNamingPattern(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		wantValid bool
	}{
		{"valid pattern", "prefix-{slug}", true},
		{"valid with issue and date", "2024-01-{slug}", true},
		{"valid simple", "test-{slug}", true},
		{"missing slug", "test-{issue}", false},
		{"empty pattern", "", false},
		{"unresolved placeholder", "{unknown}-{slug}", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateNamingPattern(tt.pattern)
			if result.Valid != tt.wantValid {
				t.Errorf("ValidateNamingPattern(%q).Valid = %v, want %v", tt.pattern, result.Valid, tt.wantValid)
			}
		})
	}
}

// TestExtractIssueFromBranch tests issue extraction from branch names.
func TestExtractIssueFromBranch(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{"extracts from feat/GH-88-name", "feat/GH-88-name", "88"},
		{"extracts from fix-123-desc", "fix-123-desc", "123"},
		{"extracts from issue/GH-42", "issue/GH-42", "42"},
		{"extracts from #123", "#123", "123"},
		{"extracts from bugfix/456", "bugfix/456", "456"},
		{"returns empty for no issue", "feature/no-number", ""},
		{"returns empty for empty branch", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractIssueFromBranch(tt.branch)
			if got != tt.expected {
				t.Errorf("ExtractIssueFromBranch(%q) = %q, want %q", tt.branch, got, tt.expected)
			}
		})
	}
}

// TestFormatIssueID tests issue ID formatting.
func TestFormatIssueID(t *testing.T) {
	tests := []struct {
		name       string
		issueID    string
		prefix     *string
		expected   string
	}{
		{"with default prefix", "123", nil, "#123"},
		{"with custom prefix", "88", strPtr("GH-"), "GH-88"},
		{"empty issue ID", "", nil, ""},
		{"empty issue with prefix", "", strPtr("GH-"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := hookio.PlanConfig{IssuePrefix: tt.prefix}
			got := FormatIssueID(tt.issueID, cfg)
			if got != tt.expected {
				t.Errorf("FormatIssueID(%q, %v) = %q, want %q", tt.issueID, tt.prefix, got, tt.expected)
			}
		})
	}
}

// TestResolveNamingPattern tests full pattern resolution.
func TestResolveNamingPattern(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		branch      string
		issuePrefix *string
		checkFunc   func(string) bool
	}{
		{
			"resolves with date and slug",
			"{date}-{slug}",
			"main",
			nil,
			func(s string) bool { return len(s) > 0 && contains(s, "{slug}") },
		},
		{
			"resolves with issue extraction",
			"{date}-{issue}-{slug}",
			"feat/GH-88-name",
			strPtr("GH-"),
			func(s string) bool { return contains(s, "GH-88") && contains(s, "{slug}") },
		},
		{
			"removes {issue} when no issue found",
			"{date}-{issue}-{slug}",
			"main",
			nil,
			func(s string) bool { return !contains(s, "{issue}") && contains(s, "{slug}") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := hookio.PlanConfig{
				NamingFormat: tt.pattern,
				DateFormat:   "YYMMDD-HHmm",
				IssuePrefix:  tt.issuePrefix,
			}
			got := ResolveNamingPattern(cfg, tt.branch)
			if !tt.checkFunc(got) {
				t.Errorf("ResolveNamingPattern(%q, %q) = %q, failed check", tt.pattern, tt.branch, got)
			}
		})
	}
}

// Helper function
func strPtr(s string) *string {
	return &s
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
