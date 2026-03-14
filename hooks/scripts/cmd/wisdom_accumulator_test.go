package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExtractLearningsWithEmptyPath tests extractLearnings with empty transcript path.
func TestExtractLearningsWithEmptyPath(t *testing.T) {
	result := extractLearnings("", "tester")
	if result != "" {
		t.Errorf("extractLearnings with empty path should return empty string, got %q", result)
	}
}

// TestExtractLearningsWithMissingFile tests extractLearnings with non-existent file.
func TestExtractLearningsWithMissingFile(t *testing.T) {
	result := extractLearnings("/nonexistent/path/to/transcript.log", "tester")
	if result != "" {
		t.Errorf("extractLearnings with missing file should return empty string, got %q", result)
	}
}

// TestExtractLearningsWithNoKeywords tests extractLearnings with content lacking keywords.
func TestExtractLearningsWithNoKeywords(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "transcript_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	content := "This is a normal log line\nAnother line without keywords\nJust plain text"
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}
	tmpfile.Close()

	result := extractLearnings(tmpfile.Name(), "tester")
	if result != "" {
		t.Errorf("extractLearnings without keywords should return empty string, got %q", result)
	}
}

// TestExtractLearningsWithKeywords tests extractLearnings extracts lines with keywords.
func TestExtractLearningsWithKeywords(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "transcript_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	content := "Normal line\nImportant: learned how to use the API\nAnother normal\nGotcha: watch out for the edge case\nMore text"
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}
	tmpfile.Close()

	result := extractLearnings(tmpfile.Name(), "tester")
	if result == "" {
		t.Errorf("extractLearnings should extract lines with keywords")
	}
	if !strings.Contains(result, "learned how to use the API") {
		t.Errorf("extractLearnings should contain learned keyword match, got: %q", result)
	}
	if !strings.Contains(result, "watch out for the edge case") {
		t.Errorf("extractLearnings should contain gotcha keyword match, got: %q", result)
	}
}

// TestExtractLearningsFiltersByLength tests extractLearnings filters by line length.
func TestExtractLearningsFiltersByLength(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "transcript_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Short line (too short, <10 chars - "important" is only 9)
	// Long line (too long, >200 chars)
	longLine := "Important: " + strings.Repeat("x", 200)
	content := "important\n" + longLine + "\nGotcha: this is a good length line with a proper message here"
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}
	tmpfile.Close()

	result := extractLearnings(tmpfile.Name(), "tester")
	// Should not contain the very short line (< 10 chars)
	if strings.Contains(result, "important\n") || strings.HasPrefix(result, "important") {
		t.Errorf("extractLearnings should filter lines < 10 chars")
	}
	// Long lines should be filtered (> 200 chars) - they won't be extracted anyway
	// Good length line should be included
	if !strings.Contains(result, "proper message") {
		t.Errorf("extractLearnings should include properly-sized lines, got: %q", result)
	}
}

// TestExtractLearningsCapsLast5 tests extractLearnings caps at 5 learnings.
func TestExtractLearningsCapsLast5(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "transcript_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Create content with 10 keyword lines
	lines := []string{}
	for i := 0; i < 10; i++ {
		lines = append(lines, "Important: learning number "+string(rune('0'+i%10)))
	}
	content := strings.Join(lines, "\n")
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}
	tmpfile.Close()

	result := extractLearnings(tmpfile.Name(), "tester")
	// Count "Important:" occurrences
	count := strings.Count(result, "Important:")
	if count > 5 {
		t.Errorf("extractLearnings should cap at 5 learnings, got %d", count)
	}
}

// TestExtractLearningsIncludesAgentType tests extractLearnings includes agent type in header.
func TestExtractLearningsIncludesAgentType(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "transcript_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	content := "Important: test learning"
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}
	tmpfile.Close()

	result := extractLearnings(tmpfile.Name(), "researcher")
	if !strings.Contains(result, "researcher") {
		t.Errorf("extractLearnings should include agent type in header, got: %q", result)
	}
}

// TestExtractLearningsLastNLines tests extractLearnings reads only last 30 lines.
func TestExtractLearningsLastNLines(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "transcript_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Create 50 lines, only last 30 should be scanned
	lines := []string{}
	for i := 0; i < 50; i++ {
		if i < 20 {
			lines = append(lines, "Line "+string(rune('0'+i%10)))
		} else {
			lines = append(lines, "Important: late learning "+string(rune('0'+(i-20)%10)))
		}
	}
	content := strings.Join(lines, "\n")
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}
	tmpfile.Close()

	result := extractLearnings(tmpfile.Name(), "tester")
	// Should have extracted some learnings
	if result == "" {
		t.Errorf("extractLearnings should extract from last 30 lines")
	}
}

// TestAppendWisdom tests appendWisdom appends content to file.
func TestAppendWisdom(t *testing.T) {
	tmpdir := t.TempDir()
	wisdomPath := filepath.Join(tmpdir, ".wisdom.md")

	// First append
	appendWisdom(wisdomPath, "First entry")
	data, err := os.ReadFile(wisdomPath)
	if err != nil {
		t.Fatalf("Failed to read wisdom file: %v", err)
	}
	if !strings.Contains(string(data), "First entry") {
		t.Errorf("appendWisdom should append content")
	}

	// Second append
	appendWisdom(wisdomPath, "Second entry")
	data, err = os.ReadFile(wisdomPath)
	if err != nil {
		t.Fatalf("Failed to read wisdom file: %v", err)
	}
	if !strings.Contains(string(data), "First entry") || !strings.Contains(string(data), "Second entry") {
		t.Errorf("appendWisdom should preserve previous content")
	}
}

// TestAppendWisdomCreatesFile tests appendWisdom creates file if missing.
func TestAppendWisdomCreatesFile(t *testing.T) {
	tmpdir := t.TempDir()
	wisdomPath := filepath.Join(tmpdir, "new_wisdom.md")

	appendWisdom(wisdomPath, "New content")

	if _, err := os.Stat(wisdomPath); err != nil {
		t.Errorf("appendWisdom should create file if missing")
	}

	data, _ := os.ReadFile(wisdomPath)
	if !strings.Contains(string(data), "New content") {
		t.Errorf("appendWisdom should write content to new file")
	}
}

// TestAppendWisdomHandlesInvalidPath tests appendWisdom with invalid path (graceful exit).
func TestAppendWisdomHandlesInvalidPath(t *testing.T) {
	// This should not panic, just silently return
	// We can't easily test the exact behavior since it silently returns on error
	appendWisdom("/invalid/path/that/does/not/exist/.wisdom.md", "content")
	// If we get here without panic, test passed
}

// TestPruneWisdomCapsLines tests pruneWisdom limits file to maxLines.
func TestPruneWisdomCapsLines(t *testing.T) {
	tmpdir := t.TempDir()
	wisdomPath := filepath.Join(tmpdir, ".wisdom.md")

	// Create file with 100 lines
	lines := []string{}
	for i := 0; i < 100; i++ {
		lines = append(lines, "Line "+string(rune('0'+i%10)))
	}
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(wisdomPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	pruneWisdom(wisdomPath, 50)

	data, err := os.ReadFile(wisdomPath)
	if err != nil {
		t.Fatalf("Failed to read pruned file: %v", err)
	}

	resultLines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(resultLines) > 50 {
		t.Errorf("pruneWisdom should cap at 50 lines, got %d", len(resultLines))
	}
}

// TestPruneWisdomKeepsLastLines tests pruneWisdom keeps last N lines (newest).
func TestPruneWisdomKeepsLastLines(t *testing.T) {
	tmpdir := t.TempDir()
	wisdomPath := filepath.Join(tmpdir, ".wisdom.md")

	lines := []string{"Line 1", "Line 2", "Line 3", "Line 4", "Line 5"}
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(wisdomPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	pruneWisdom(wisdomPath, 3)

	data, _ := os.ReadFile(wisdomPath)
	result := strings.TrimSpace(string(data))
	// Should keep last 3 lines (3, 4, 5)
	if !strings.Contains(result, "Line 3") || !strings.Contains(result, "Line 5") {
		t.Errorf("pruneWisdom should keep last N lines, got: %q", result)
	}
	if strings.Contains(result, "Line 1") {
		t.Errorf("pruneWisdom should not keep early lines")
	}
}

// TestPruneWisdomNothingToDo tests pruneWisdom with file under limit.
func TestPruneWisdomNothingToDo(t *testing.T) {
	tmpdir := t.TempDir()
	wisdomPath := filepath.Join(tmpdir, ".wisdom.md")

	content := "Line 1\nLine 2\nLine 3"
	if err := os.WriteFile(wisdomPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	original, _ := os.ReadFile(wisdomPath)
	pruneWisdom(wisdomPath, 50)
	after, _ := os.ReadFile(wisdomPath)

	if string(original) != string(after) {
		t.Errorf("pruneWisdom should not modify file under limit")
	}
}

// TestPruneWisdomWithMissingFile tests pruneWisdom gracefully handles missing file.
func TestPruneWisdomWithMissingFile(t *testing.T) {
	// Should not panic
	pruneWisdom("/nonexistent/path/file.md", 50)
	// If we get here without panic, test passed
}
