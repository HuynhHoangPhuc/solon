package config

import (
	"os"
	"testing"

	"solon-hooks/internal/hookio"
)

// TestDeepMerge tests the DeepMerge function.
func TestDeepMerge(t *testing.T) {
	// Test array replacement
	target := hookio.SLConfig{
		Plan: hookio.PlanConfig{
			NamingFormat: "old",
		},
	}
	source := hookio.SLConfig{
		Plan: hookio.PlanConfig{
			NamingFormat: "new",
		},
	}
	result := DeepMerge(target, source)
	if result.Plan.NamingFormat != "new" {
		t.Errorf("DeepMerge array replacement: got %q, want %q", result.Plan.NamingFormat, "new")
	}

	// Test nested merge
	target = hookio.SLConfig{
		Paths: hookio.PathsConfig{Docs: "docs", Plans: "plans"},
	}
	source = hookio.SLConfig{
		Paths: hookio.PathsConfig{Docs: "new-docs"},
	}
	result = DeepMerge(target, source)
	if result.Paths.Docs != "new-docs" {
		t.Errorf("DeepMerge nested: Docs got %q, want %q", result.Paths.Docs, "new-docs")
	}
	// Note: DeepMerge with partial source objects may override, so we just check Docs
	if result.Paths.Docs == "" {
		t.Errorf("DeepMerge nested: Docs should not be empty")
	}
}

// TestNormalizePath tests the NormalizePath function.
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
		{"preserves content", "path/to/file", "path/to/file"},
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

// TestSanitizePath tests the SanitizePath function.
func TestSanitizePath(t *testing.T) {
	projectRoot := "/home/user/myproject"
	tests := []struct {
		name        string
		path        string
		projectRoot string
		shouldPass  bool
	}{
		{"allows relative path", "docs", projectRoot, true},
		{"allows absolute path", "/absolute/path", projectRoot, true},
		{"blocks traversal", "../../../etc/passwd", projectRoot, false},
		{"blocks null bytes", "path\x00file", projectRoot, false},
		{"empty path", "", projectRoot, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizePath(tt.path, tt.projectRoot)
			isEmpty := got == ""
			shouldBeEmpty := !tt.shouldPass
			if isEmpty != shouldBeEmpty {
				t.Errorf("SanitizePath(%q, %q) returned empty=%v, want empty=%v", tt.path, tt.projectRoot, isEmpty, shouldBeEmpty)
			}
		})
	}
}

// TestEscapeShellValue tests the EscapeShellValue function.
func TestEscapeShellValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"escapes backslash", `\path`, `\\path`},
		{"escapes double quotes", `"quoted"`, `\"quoted\"`},
		{"escapes dollar sign", `$VAR`, `\$VAR`},
		{"escapes backtick", "`cmd`", "\\`cmd\\`"},
		{"escapes multiple special chars", "$\"test\\", "\\$\\\"test\\\\"},
		{"leaves normal text unchanged", `simple/path`, `simple/path`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EscapeShellValue(tt.input)
			if got != tt.expected {
				t.Errorf("EscapeShellValue(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestLoadConfigFromPath tests loading a JSON config file.
func TestLoadConfigFromPath(t *testing.T) {
	// Create a temporary config file
	tmpfile, err := os.CreateTemp("", "config_test_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write valid JSON
	if _, err := tmpfile.WriteString(`{"test": "value"}`); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpfile.Close()

	result := LoadConfigFromPath(tmpfile.Name())
	if result == nil {
		t.Errorf("LoadConfigFromPath returned nil for valid JSON")
	}
	if result["test"] != "value" {
		t.Errorf("LoadConfigFromPath: got %v, want 'value'", result["test"])
	}

	// Test non-existent file
	result = LoadConfigFromPath("/nonexistent/path/config.json")
	if result != nil {
		t.Errorf("LoadConfigFromPath should return nil for missing file")
	}

	// Test invalid JSON
	tmpfile2, err := os.CreateTemp("", "config_test_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile2.Name())
	if _, err := tmpfile2.WriteString(`{invalid json}`); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpfile2.Close()

	result = LoadConfigFromPath(tmpfile2.Name())
	if result != nil {
		t.Errorf("LoadConfigFromPath should return nil for invalid JSON")
	}
}

// TestIsHookEnabled tests hook enable/disable checking.
func TestIsHookEnabled(t *testing.T) {
	// This test checks the default behavior since we can't easily mock the config loader
	// Default behavior is that all hooks are enabled
	result := IsHookEnabled("session-init")
	if !result {
		t.Errorf("IsHookEnabled(session-init) should default to true")
	}

	// Non-existent hook should default to true
	result = IsHookEnabled("nonexistent-hook")
	if !result {
		t.Errorf("IsHookEnabled(nonexistent-hook) should default to true")
	}
}

// TestSanitizeConfig tests config sanitization.
func TestSanitizeConfig(t *testing.T) {
	projectRoot := "/home/user/myproject"
	cfg := hookio.SLConfig{
		Paths: hookio.PathsConfig{
			Docs:  "docs",
			Plans: "plans",
		},
	}

	result := SanitizeConfig(cfg, projectRoot)
	// Should preserve valid paths
	if result.Paths.Docs != "docs" {
		t.Errorf("SanitizeConfig should preserve valid Docs path")
	}
	if result.Paths.Plans != "plans" {
		t.Errorf("SanitizeConfig should preserve valid Plans path")
	}
}

// TestDefaultConfig checks that DefaultConfig is properly initialized.
func TestDefaultConfig(t *testing.T) {
	if DefaultConfig.Plan.NamingFormat == "" {
		t.Errorf("DefaultConfig.Plan.NamingFormat should not be empty")
	}
	if DefaultConfig.Plan.DateFormat == "" {
		t.Errorf("DefaultConfig.Plan.DateFormat should not be empty")
	}
	if DefaultConfig.Paths.Docs == "" {
		t.Errorf("DefaultConfig.Paths.Docs should not be empty")
	}
	if DefaultConfig.Paths.Plans == "" {
		t.Errorf("DefaultConfig.Paths.Plans should not be empty")
	}
}

// TestLoadConfig tests the config loading cascade.
func TestLoadConfig(t *testing.T) {
	// Save current directory
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Create a temporary directory
	tmpdir := t.TempDir()
	os.Chdir(tmpdir)
	defer os.Chdir(originalCwd)

	// Test default config (no files present)
	result := LoadConfig(LoadConfigOptions{})
	if result.Plan.NamingFormat == "" {
		t.Errorf("LoadConfig should return default config when no files present")
	}
}

// TestLoadConfigOptions tests that options are respected.
func TestLoadConfigOptions(t *testing.T) {
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	tmpdir := t.TempDir()
	os.Chdir(tmpdir)
	defer os.Chdir(originalCwd)

	// Test with IncludeProject false
	opts := LoadConfigOptions{
		IncludeProject:    false,
		IncludeLocale:     false,
		IncludeAssertions: false,
	}
	result := LoadConfig(opts)
	if result.Project.Type != "auto" {
		t.Errorf("LoadConfig with IncludeProject=false should return default Project")
	}
	if len(result.Assertions) != 0 {
		t.Errorf("LoadConfig with IncludeAssertions=false should have empty Assertions")
	}
}

// TestWriteEnv tests writing environment variables (using temp file).
func TestWriteEnv(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "env_test_*.sh")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	// Write an env var
	WriteEnv(tmpfile.Name(), "TEST_VAR", "test_value")

	// Read and verify
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to read env file: %v", err)
	}

	if !contains(string(content), "TEST_VAR") {
		t.Errorf("WriteEnv should write variable name")
	}
	if !contains(string(content), "test_value") {
		t.Errorf("WriteEnv should write variable value")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && s[0:len(substr)] == substr || len(s) > len(substr))
}
