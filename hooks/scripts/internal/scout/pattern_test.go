package scout

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadPatterns tests pattern loading from .slignore.
func TestLoadPatterns(t *testing.T) {
	// Test default patterns (missing file)
	patterns := LoadPatterns("/nonexistent/.slignore")
	if len(patterns) == 0 {
		t.Errorf("LoadPatterns should return default patterns when file missing")
	}
	found := false
	for _, p := range patterns {
		if p == "node_modules" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("LoadPatterns defaults should contain 'node_modules'")
	}

	// Test loading from file
	tmpfile, err := os.CreateTemp("", ".slignore")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write patterns to file
	tmpfile.WriteString("custom-pattern\n")
	tmpfile.WriteString("# comment line\n")
	tmpfile.WriteString("another-pattern\n")
	tmpfile.Close()

	patterns = LoadPatterns(tmpfile.Name())
	if len(patterns) != 2 {
		t.Errorf("LoadPatterns should skip comments and empty lines, got %d patterns", len(patterns))
	}
	if patterns[0] != "custom-pattern" {
		t.Errorf("LoadPatterns should parse first pattern correctly")
	}
	if patterns[1] != "another-pattern" {
		t.Errorf("LoadPatterns should parse second pattern correctly")
	}

	// Test empty file (should return defaults)
	tmpfile2, err := os.CreateTemp("", ".slignore")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile2.Name())
	tmpfile2.Close()

	patterns = LoadPatterns(tmpfile2.Name())
	if len(patterns) == 0 {
		t.Errorf("LoadPatterns should return defaults for empty file")
	}
}

// TestCreateMatcher tests matcher creation.
func TestCreateMatcher(t *testing.T) {
	// Test simple pattern expansion
	patterns := []string{"node_modules", "dist"}
	matcher := CreateMatcher(patterns)

	if matcher == nil {
		t.Errorf("CreateMatcher should return non-nil matcher")
	}
	if len(matcher.original) != 2 {
		t.Errorf("CreateMatcher should preserve original patterns")
	}

	// Test negation patterns
	patterns = []string{"build", "!build/keep"}
	matcher = CreateMatcher(patterns)
	if len(matcher.patterns) == 0 {
		t.Errorf("CreateMatcher should expand negation patterns")
	}

	// Test patterns with wildcards
	patterns = []string{"src/**/*.test.js", "dist/*"}
	matcher = CreateMatcher(patterns)
	if len(matcher.patterns) == 0 {
		t.Errorf("CreateMatcher should handle wildcard patterns")
	}
}

// TestMatchPath tests path matching.
func TestMatchPath(t *testing.T) {
	patterns := []string{"node_modules", "dist", ".git"}
	matcher := CreateMatcher(patterns)

	tests := []struct {
		name     string
		path     string
		wantBlock bool
	}{
		{"blocks node_modules/foo.js", "node_modules/foo.js", true},
		{"blocks nested node_modules", "src/node_modules/pkg/index.js", true},
		{"allows src/main.js", "src/main.js", false},
		{"blocks dist/index.js", "dist/index.js", true},
		{"blocks .git/objects/abc", ".git/objects/abc", true},
		{"allows .github/workflows", ".github/workflows/test.yml", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchPath(matcher, tt.path)
			if result.Blocked != tt.wantBlock {
				t.Errorf("MatchPath(%q) blocked=%v, want %v", tt.path, result.Blocked, tt.wantBlock)
			}
		})
	}
}

// TestMatchPathNormalization tests path normalization in matching.
func TestMatchPathNormalization(t *testing.T) {
	patterns := []string{"node_modules"}
	matcher := CreateMatcher(patterns)

	tests := []struct {
		name     string
		path     string
		wantBlock bool
	}{
		{"normalizes backslashes", "node_modules\\package\\index.js", true},
		{"strips leading ./", "./node_modules/pkg", true},
		{"strips leading /", "/node_modules/pkg", true},
		{"strips leading ../", "../node_modules/pkg", true},
		{"handles empty after normalization", "./", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchPath(matcher, tt.path)
			if result.Blocked != tt.wantBlock {
				t.Errorf("MatchPath(%q) blocked=%v, want %v", tt.path, result.Blocked, tt.wantBlock)
			}
		})
	}
}

// TestMatchPathEmpty tests empty/nil cases.
func TestMatchPathEmpty(t *testing.T) {
	patterns := []string{"node_modules"}
	matcher := CreateMatcher(patterns)

	tests := []struct {
		name     string
		path     string
		wantBlock bool
	}{
		{"empty path", "", false},
		{"nil matcher", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchPath(matcher, tt.path)
			if result.Blocked != tt.wantBlock {
				t.Errorf("MatchPath(%q) blocked=%v, want %v", tt.path, result.Blocked, tt.wantBlock)
			}
		})
	}

	// Test nil matcher
	result := MatchPath(nil, "node_modules/pkg")
	if result.Blocked {
		t.Errorf("MatchPath(nil, ...) should not block")
	}
}

// TestMatchPathPattern verifies pattern name is returned.
func TestMatchPathPattern(t *testing.T) {
	patterns := []string{"node_modules"}
	matcher := CreateMatcher(patterns)

	result := MatchPath(matcher, "node_modules/pkg")
	if result.Blocked && result.Pattern == "" {
		t.Errorf("MatchPath should return matching pattern name")
	}
}

// TestDefaultPatterns verifies defaults are initialized.
func TestDefaultPatterns(t *testing.T) {
	if len(DefaultPatterns) == 0 {
		t.Errorf("DefaultPatterns should not be empty")
	}
	found := false
	for _, p := range DefaultPatterns {
		if p == "node_modules" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("DefaultPatterns should contain node_modules")
	}
}

// TestIntegrationScoutPatterns tests real-world pattern scenarios.
func TestIntegrationScoutPatterns(t *testing.T) {
	// Create a temporary directory structure
	tmpdir := t.TempDir()

	// Create some test directories
	os.MkdirAll(filepath.Join(tmpdir, "node_modules", "pkg"), 0755)
	os.MkdirAll(filepath.Join(tmpdir, "src"), 0755)
	os.MkdirAll(filepath.Join(tmpdir, ".git"), 0755)

	patterns := []string{"node_modules", ".git", "dist"}
	matcher := CreateMatcher(patterns)

	// These should be blocked
	blockedPaths := []string{
		"node_modules/pkg/index.js",
		"dist/bundle.js",
		".git/config",
	}
	for _, path := range blockedPaths {
		result := MatchPath(matcher, path)
		if !result.Blocked {
			t.Errorf("MatchPath(%q) should be blocked", path)
		}
	}

	// These should be allowed
	allowedPaths := []string{
		"src/main.js",
		"package.json",
		"README.md",
	}
	for _, path := range allowedPaths {
		result := MatchPath(matcher, path)
		if result.Blocked {
			t.Errorf("MatchPath(%q) should be allowed", path)
		}
	}
}
