package wisdom

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadWisdomWithEmptyPath tests ReadWisdom when planPath is empty and no sessionID.
func TestReadWisdomWithEmptyPath(t *testing.T) {
	result := ReadWisdom("", "", 10)
	if result != "" {
		t.Errorf("ReadWisdom with empty path and sessionID should return empty string, got %q", result)
	}
}

// TestReadWisdomWithMissingFile tests ReadWisdom when wisdom file doesn't exist.
func TestReadWisdomWithMissingFile(t *testing.T) {
	tmpdir := t.TempDir()
	result := ReadWisdom(tmpdir, "", 10)
	if result != "" {
		t.Errorf("ReadWisdom with missing file should return empty string, got %q", result)
	}
}

// TestReadWisdomWithContent tests ReadWisdom reads existing file content.
func TestReadWisdomWithContent(t *testing.T) {
	tmpdir := t.TempDir()
	wisdomPath := filepath.Join(tmpdir, ".wisdom.md")

	content := "### Tester (10:00)\nFirst learning\nSecond learning\n### Tester (11:00)\nThird learning"
	if err := os.WriteFile(wisdomPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result := ReadWisdom(tmpdir, "", 10)
	if result == "" {
		t.Errorf("ReadWisdom should return content when file exists")
	}
	if !strings.Contains(result, "First learning") {
		t.Errorf("ReadWisdom should contain content, got: %q", result)
	}
}

// TestReadWisdomMaxLines tests that ReadWisdom respects maxLines limit.
func TestReadWisdomMaxLines(t *testing.T) {
	tmpdir := t.TempDir()
	wisdomPath := filepath.Join(tmpdir, ".wisdom.md")

	// Create content with many lines
	lines := make([]string, 20)
	for i := 0; i < 20; i++ {
		lines[i] = "Line " + string(rune('0'+i%10))
	}
	content := strings.Join(lines, "\n")

	if err := os.WriteFile(wisdomPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result := ReadWisdom(tmpdir, "", 5)
	resultLines := strings.Split(strings.TrimSpace(result), "\n")
	if len(resultLines) > 5 {
		t.Errorf("ReadWisdom(maxLines=5) returned %d lines, want max 5", len(resultLines))
	}
}

// TestReadWisdomFromTempFile tests ReadWisdom falls back to /tmp when planPath is empty.
func TestReadWisdomFromTempFile(t *testing.T) {
	sessionID := "test-session-123"
	wisdomPath := filepath.Join(os.TempDir(), "sl-wisdom-"+sessionID+".md")

	// Clean up if it exists
	defer os.Remove(wisdomPath)

	content := "### Researcher (14:00)\nLearned something important"
	if err := os.WriteFile(wisdomPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result := ReadWisdom("", sessionID, 10)
	if result == "" {
		t.Errorf("ReadWisdom should fall back to /tmp file when planPath is empty")
	}
	if !strings.Contains(result, "Learned something important") {
		t.Errorf("ReadWisdom should contain temp file content, got: %q", result)
	}
}

// TestReadWisdomPrefersPlan tests ReadWisdom prefers plan path over /tmp.
func TestReadWisdomPrefersPlan(t *testing.T) {
	tmpdir := t.TempDir()
	sessionID := "test-session-456"

	// Create plan wisdom file
	planWisdomPath := filepath.Join(tmpdir, ".wisdom.md")
	planContent := "Plan wisdom content"
	if err := os.WriteFile(planWisdomPath, []byte(planContent), 0644); err != nil {
		t.Fatalf("Failed to write plan wisdom file: %v", err)
	}

	// Create temp wisdom file
	tempWisdomPath := filepath.Join(os.TempDir(), "sl-wisdom-"+sessionID+".md")
	defer os.Remove(tempWisdomPath)
	tempContent := "Temp wisdom content"
	if err := os.WriteFile(tempWisdomPath, []byte(tempContent), 0644); err != nil {
		t.Fatalf("Failed to write temp wisdom file: %v", err)
	}

	result := ReadWisdom(tmpdir, sessionID, 10)
	if !strings.Contains(result, "Plan wisdom content") {
		t.Errorf("ReadWisdom should prefer plan wisdom over temp, got: %q", result)
	}
	if strings.Contains(result, "Temp wisdom content") {
		t.Errorf("ReadWisdom should not read temp when plan exists")
	}
}

// TestReadWisdomWithEmptyFile tests ReadWisdom with empty wisdom file.
func TestReadWisdomWithEmptyFile(t *testing.T) {
	tmpdir := t.TempDir()
	wisdomPath := filepath.Join(tmpdir, ".wisdom.md")

	if err := os.WriteFile(wisdomPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result := ReadWisdom(tmpdir, "", 10)
	if result != "" {
		t.Errorf("ReadWisdom with empty file should return empty string, got %q", result)
	}
}

// TestReadWisdomWithWhitespaceOnly tests ReadWisdom with whitespace-only file.
func TestReadWisdomWithWhitespaceOnly(t *testing.T) {
	tmpdir := t.TempDir()
	wisdomPath := filepath.Join(tmpdir, ".wisdom.md")

	if err := os.WriteFile(wisdomPath, []byte("   \n\n   \n"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result := ReadWisdom(tmpdir, "", 10)
	if result != "" {
		t.Errorf("ReadWisdom with whitespace-only file should return empty string")
	}
}

// TestResolveWisdomPath tests the resolution logic for wisdom file paths.
func TestResolveWisdomPath(t *testing.T) {
	tmpdir := t.TempDir()
	sessionID := "test-123"

	tests := []struct {
		name      string
		planPath  string
		sessionID string
		setup     func()
		wantEmpty bool
	}{
		{
			"empty params returns empty",
			"",
			"",
			func() {},
			true,
		},
		{
			"plan path with existing file returns plan path",
			tmpdir,
			"",
			func() {
				os.WriteFile(filepath.Join(tmpdir, ".wisdom.md"), []byte("content"), 0644)
			},
			false,
		},
		{
			"plan path without file falls back to session temp",
			tmpdir,
			sessionID,
			func() {
				// Don't create plan wisdom file
			},
			false, // Will try to resolve to temp path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result := resolveWisdomPath(tt.planPath, tt.sessionID)
			if tt.wantEmpty && result != "" {
				t.Errorf("resolveWisdomPath should return empty, got %q", result)
			}
			if !tt.wantEmpty && result == "" {
				t.Errorf("resolveWisdomPath should return non-empty path")
			}
		})
	}
}

// TestReadWisdomOrderingNewest tests that ReadWisdom returns newest entries first.
func TestReadWisdomOrderingNewest(t *testing.T) {
	tmpdir := t.TempDir()
	wisdomPath := filepath.Join(tmpdir, ".wisdom.md")

	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	if err := os.WriteFile(wisdomPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result := ReadWisdom(tmpdir, "", 2)
	resultLines := strings.Split(strings.TrimSpace(result), "\n")

	// Should get last 2 lines
	if len(resultLines) < 2 || resultLines[len(resultLines)-1] != "Line 5" {
		t.Errorf("ReadWisdom should return newest lines last, got: %v", resultLines)
	}
}
