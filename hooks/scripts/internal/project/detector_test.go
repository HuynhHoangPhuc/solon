package project

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDetectProjectType_Override returns the override value when not "auto".
func TestDetectProjectType_Override(t *testing.T) {
	tests := []struct {
		override string
		want     string
	}{
		{"monorepo", "monorepo"},
		{"library", "library"},
		{"single-repo", "single-repo"},
	}
	for _, tt := range tests {
		got := DetectProjectType(tt.override)
		if got != tt.want {
			t.Errorf("DetectProjectType(%q) = %q, want %q", tt.override, got, tt.want)
		}
	}
}

// TestDetectProjectType_AutoFallback returns "single-repo" when no indicators found.
func TestDetectProjectType_AutoFallback(t *testing.T) {
	// Use a temp dir with no package.json or workspace files
	tmpDir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig) //nolint:errcheck
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	got := DetectProjectType("auto")
	if got != "single-repo" {
		t.Errorf("DetectProjectType(auto) in empty dir = %q, want %q", got, "single-repo")
	}
}

// TestDetectProjectType_PnpmWorkspace returns "monorepo" when pnpm-workspace.yaml exists.
func TestDetectProjectType_PnpmWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig) //nolint:errcheck
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(tmpDir, "pnpm-workspace.yaml"), []byte("packages:\n  - 'packages/*'\n"), 0644) //nolint:errcheck

	got := DetectProjectType("")
	if got != "monorepo" {
		t.Errorf("DetectProjectType with pnpm-workspace.yaml = %q, want %q", got, "monorepo")
	}
}

// TestDetectPackageManager_Override returns override value when not "auto".
func TestDetectPackageManager_Override(t *testing.T) {
	tests := []struct {
		override string
		want     string
	}{
		{"bun", "bun"},
		{"pnpm", "pnpm"},
		{"yarn", "yarn"},
		{"npm", "npm"},
	}
	for _, tt := range tests {
		got := DetectPackageManager(tt.override)
		if got != tt.want {
			t.Errorf("DetectPackageManager(%q) = %q, want %q", tt.override, got, tt.want)
		}
	}
}

// TestDetectPackageManager_NoLockfile returns empty string when no lockfile found.
func TestDetectPackageManager_NoLockfile(t *testing.T) {
	tmpDir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig) //nolint:errcheck
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	got := DetectPackageManager("auto")
	if got != "" {
		t.Errorf("DetectPackageManager(auto) in empty dir = %q, want %q", got, "")
	}
}

// TestDetectPackageManager_BunLockfile returns "bun" when bun.lockb exists.
func TestDetectPackageManager_BunLockfile(t *testing.T) {
	tmpDir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig) //nolint:errcheck
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(tmpDir, "bun.lockb"), []byte(""), 0644) //nolint:errcheck

	got := DetectPackageManager("")
	if got != "bun" {
		t.Errorf("DetectPackageManager with bun.lockb = %q, want %q", got, "bun")
	}
}

// TestDetectFramework_Override returns override value when not "auto".
func TestDetectFramework_Override(t *testing.T) {
	got := DetectFramework("next")
	if got != "next" {
		t.Errorf("DetectFramework(%q) = %q, want %q", "next", got, "next")
	}
}

// TestDetectFramework_NoPackageJson returns empty string when no package.json.
func TestDetectFramework_NoPackageJson(t *testing.T) {
	tmpDir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig) //nolint:errcheck
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	got := DetectFramework("auto")
	if got != "" {
		t.Errorf("DetectFramework(auto) with no package.json = %q, want %q", got, "")
	}
}

// TestIsGitRepo_NonGitDir returns false for a directory without .git.
func TestIsGitRepo_NonGitDir(t *testing.T) {
	tmpDir := t.TempDir()
	if IsGitRepo(tmpDir) {
		t.Errorf("IsGitRepo(%q) should be false for non-git dir", tmpDir)
	}
}

// TestIsGitRepo_WithGitDir returns true for a directory containing .git.
func TestIsGitRepo_WithGitDir(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if !IsGitRepo(tmpDir) {
		t.Errorf("IsGitRepo(%q) should be true when .git exists", tmpDir)
	}
}

// TestGetCodingLevelStyleName maps known levels to correct names.
func TestGetCodingLevelStyleName(t *testing.T) {
	tests := []struct {
		level int
		want  string
	}{
		{0, "coding-level-0-eli5"},
		{1, "coding-level-1-junior"},
		{2, "coding-level-2-mid"},
		{3, "coding-level-3-senior"},
		{4, "coding-level-4-lead"},
		{5, "coding-level-5-god"},
		{99, "coding-level-5-god"}, // unknown falls back to god
	}
	for _, tt := range tests {
		got := GetCodingLevelStyleName(tt.level)
		if got != tt.want {
			t.Errorf("GetCodingLevelStyleName(%d) = %q, want %q", tt.level, got, tt.want)
		}
	}
}

// TestGetCodingLevelGuidelines_MinusOne returns empty for level -1.
func TestGetCodingLevelGuidelines_MinusOne(t *testing.T) {
	got := GetCodingLevelGuidelines(-1, "")
	if got != "" {
		t.Errorf("GetCodingLevelGuidelines(-1) should be empty, got %q", got)
	}
}

// TestGetCodingLevelGuidelines_MissingFile returns empty when style file does not exist.
func TestGetCodingLevelGuidelines_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	got := GetCodingLevelGuidelines(3, tmpDir)
	if got != "" {
		t.Errorf("GetCodingLevelGuidelines with missing file should return empty, got %q", got)
	}
}
