package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveRulesPath_NotFound returns empty string when file does not exist.
func TestResolveRulesPath_NotFound(t *testing.T) {
	result := ResolveRulesPath("nonexistent-file-xyz.md", ".claude-test-missing")
	if result != "" {
		t.Errorf("expected empty string for missing file, got %q", result)
	}
}

// TestResolveRulesPath_LocalFile returns relative path when file exists locally.
func TestResolveRulesPath_LocalFile(t *testing.T) {
	// Create a temp dir, write the file, change cwd to it
	tmpDir := t.TempDir()
	configDir := "testcfg"
	rulesDir := filepath.Join(tmpDir, configDir, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	filename := "test-rule.md"
	if err := os.WriteFile(filepath.Join(rulesDir, filename), []byte("# rule"), 0644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	defer os.Chdir(orig) //nolint:errcheck
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	got := ResolveRulesPath(filename, configDir)
	if got == "" {
		t.Error("expected non-empty path for existing local rules file")
	}
	if !strings.Contains(got, configDir+"/rules/"+filename) {
		t.Errorf("expected path to contain %q, got %q", configDir+"/rules/"+filename, got)
	}
}

// TestResolveScriptPath_NotFound returns empty string when script file does not exist.
func TestResolveScriptPath_NotFound(t *testing.T) {
	result := ResolveScriptPath("nonexistent-script-xyz.py", ".claude-test-missing")
	if result != "" {
		t.Errorf("expected empty string for missing script, got %q", result)
	}
}

// TestResolveScriptPath_LocalFile returns relative path when script exists locally.
func TestResolveScriptPath_LocalFile(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := "testcfg"
	scriptsDir := filepath.Join(tmpDir, configDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	filename := "test_script.py"
	if err := os.WriteFile(filepath.Join(scriptsDir, filename), []byte("# script"), 0644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	defer os.Chdir(orig) //nolint:errcheck
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	got := ResolveScriptPath(filename, configDir)
	if got == "" {
		t.Error("expected non-empty path for existing local script")
	}
	if !strings.Contains(got, configDir+"/scripts/"+filename) {
		t.Errorf("expected path to contain %q, got %q", configDir+"/scripts/"+filename, got)
	}
}

// TestWasRecentlyInjected_EmptyPath returns false for empty path.
func TestWasRecentlyInjected_EmptyPath(t *testing.T) {
	if WasRecentlyInjected("") {
		t.Error("empty path should return false")
	}
}

// TestWasRecentlyInjected_NoMarker returns false when marker is absent.
func TestWasRecentlyInjected_NoMarker(t *testing.T) {
	f, err := os.CreateTemp("", "transcript-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("some content without the marker\n") //nolint:errcheck
	f.Close()

	if WasRecentlyInjected(f.Name()) {
		t.Error("file without marker should return false")
	}
}

// TestWasRecentlyInjected_MarkerPresent returns true when marker is in last 150 lines.
func TestWasRecentlyInjected_MarkerPresent(t *testing.T) {
	f, err := os.CreateTemp("", "transcript-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	// Write marker within the last 150 lines
	f.WriteString("[IMPORTANT] Consider Modularization\n") //nolint:errcheck
	f.Close()

	if !WasRecentlyInjected(f.Name()) {
		t.Error("file with marker should return true")
	}
}

// TestWasRecentlyInjected_MarkerBeyond150Lines returns false when marker is beyond last 150 lines.
func TestWasRecentlyInjected_MarkerBeyond150Lines(t *testing.T) {
	f, err := os.CreateTemp("", "transcript-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	// Write marker first, then 200 more lines to push it beyond the 150-line window
	f.WriteString("[IMPORTANT] Consider Modularization\n") //nolint:errcheck
	for i := 0; i < 200; i++ {
		f.WriteString("padding line\n") //nolint:errcheck
	}
	f.Close()

	if WasRecentlyInjected(f.Name()) {
		t.Error("marker beyond last 150 lines should return false")
	}
}
