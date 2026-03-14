package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildCompactionContextWithEmptyPlan tests BuildCompactionContext with empty planPath.
func TestBuildCompactionContextWithEmptyPlan(t *testing.T) {
	result := BuildCompactionContext("", "")
	if result != "" {
		t.Errorf("BuildCompactionContext with empty planPath should return empty string, got %q", result)
	}
}

// TestBuildCompactionContextWithNonexistentPlan tests BuildCompactionContext with non-existent plan.
func TestBuildCompactionContextWithNonexistentPlan(t *testing.T) {
	result := BuildCompactionContext("/nonexistent/plan/path", "session-123")
	if result != "" {
		t.Errorf("BuildCompactionContext with nonexistent plan should return empty string, got %q", result)
	}
}

// TestBuildCompactionContextWithEmptyPlanDir tests BuildCompactionContext with empty plan directory.
func TestBuildCompactionContextWithEmptyPlanDir(t *testing.T) {
	tmpdir := t.TempDir()
	result := BuildCompactionContext(tmpdir, "")
	if result != "" {
		t.Errorf("BuildCompactionContext with empty plan dir should return empty string, got %q", result)
	}
}

// TestBuildCompactionContextWithPhaseFiles tests BuildCompactionContext includes phase status.
func TestBuildCompactionContextWithPhaseFiles(t *testing.T) {
	tmpdir := t.TempDir()

	// Create phase files with todos
	phase1 := filepath.Join(tmpdir, "phase-01-setup.md")
	phase1Content := `# Setup

- [x] Install dependencies
- [x] Configure environment
- [ ] Run tests
`
	if err := os.WriteFile(phase1, []byte(phase1Content), 0644); err != nil {
		t.Fatalf("Failed to write phase file: %v", err)
	}

	phase2 := filepath.Join(tmpdir, "phase-02-implement.md")
	phase2Content := `# Implementation

- [x] Write code
- [x] Add tests
- [x] Document
`
	if err := os.WriteFile(phase2, []byte(phase2Content), 0644); err != nil {
		t.Fatalf("Failed to write phase file: %v", err)
	}

	result := BuildCompactionContext(tmpdir, "")
	if result == "" {
		t.Errorf("BuildCompactionContext should return content when phase files exist")
	}
	if !strings.Contains(result, "phase-01-setup.md") {
		t.Errorf("BuildCompactionContext should include phase-01 status")
	}
	if !strings.Contains(result, "phase-02-implement.md") {
		t.Errorf("BuildCompactionContext should include phase-02 status")
	}
	if !strings.Contains(result, "2/3") || !strings.Contains(result, "3/3") {
		t.Errorf("BuildCompactionContext should include todo counts")
	}
}

// TestBuildCompactionContextWithWisdom tests BuildCompactionContext includes wisdom.
func TestBuildCompactionContextWithWisdom(t *testing.T) {
	tmpdir := t.TempDir()

	// Create wisdom file
	wisdomPath := filepath.Join(tmpdir, ".wisdom.md")
	wisdomContent := "### Researcher (10:00)\nLearned something important\n### Debugger (11:00)\nFixed critical bug"
	if err := os.WriteFile(wisdomPath, []byte(wisdomContent), 0644); err != nil {
		t.Fatalf("Failed to write wisdom file: %v", err)
	}

	// Create a phase file
	phase1 := filepath.Join(tmpdir, "phase-01.md")
	if err := os.WriteFile(phase1, []byte("# Phase\n- [x] Done"), 0644); err != nil {
		t.Fatalf("Failed to write phase file: %v", err)
	}

	result := BuildCompactionContext(tmpdir, "")
	if !strings.Contains(result, "Accumulated Learnings") {
		t.Errorf("BuildCompactionContext should include wisdom section")
	}
	if !strings.Contains(result, "Learned something important") {
		t.Errorf("BuildCompactionContext should include wisdom content, got: %q", result)
	}
}

// TestBuildCompactionContextHeaderFormat tests BuildCompactionContext has proper header.
func TestBuildCompactionContextHeaderFormat(t *testing.T) {
	tmpdir := t.TempDir()

	// Create a phase file
	phase1 := filepath.Join(tmpdir, "phase-01.md")
	if err := os.WriteFile(phase1, []byte("# Phase\n- [x] Done"), 0644); err != nil {
		t.Fatalf("Failed to write phase file: %v", err)
	}

	result := BuildCompactionContext(tmpdir, "")
	if !strings.Contains(result, "POST-COMPACTION RECOVERY CONTEXT") {
		t.Errorf("BuildCompactionContext should include recovery context header")
	}
	if !strings.Contains(result, "Use this context to resume work") {
		t.Errorf("BuildCompactionContext should include usage instructions")
	}
}

// TestCountTodosBasic tests countTodos with basic todo format.
func TestCountTodosBasic(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "phase_*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	content := `# Phase

- [x] Completed task
- [ ] Incomplete task
- [x] Another done task
`
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}
	tmpfile.Close()

	total, done := countTodos(tmpfile.Name())
	if total != 3 {
		t.Errorf("countTodos total = %d, want 3", total)
	}
	if done != 2 {
		t.Errorf("countTodos done = %d, want 2", done)
	}
}

// TestCountTodosEmptyFile tests countTodos with file containing no todos.
func TestCountTodosEmptyFile(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "phase_*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	content := `# Phase

This file has no todos.
Just regular content.
`
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}
	tmpfile.Close()

	total, done := countTodos(tmpfile.Name())
	if total != 0 {
		t.Errorf("countTodos total = %d, want 0", total)
	}
	if done != 0 {
		t.Errorf("countTodos done = %d, want 0", done)
	}
}

// TestCountTodosWithCapitalX tests countTodos recognizes both [x] and [X].
func TestCountTodosWithCapitalX(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "phase_*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	content := `# Phase

- [x] lowercase x
- [X] UPPERCASE X
- [ ] incomplete
`
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}
	tmpfile.Close()

	total, done := countTodos(tmpfile.Name())
	if total != 3 {
		t.Errorf("countTodos total = %d, want 3", total)
	}
	if done != 2 {
		t.Errorf("countTodos done = %d, want 2 (both lowercase and uppercase X)", done)
	}
}

// TestCountTodosWithWhitespace tests countTodos handles leading whitespace.
func TestCountTodosWithWhitespace(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "phase_*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	content := `# Phase

  - [x] Indented completed
    - [x] Nested completed
- [ ] Non-indented incomplete
`
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}
	tmpfile.Close()

	total, done := countTodos(tmpfile.Name())
	if total != 3 {
		t.Errorf("countTodos total = %d, want 3", total)
	}
	if done != 2 {
		t.Errorf("countTodos done = %d, want 2", done)
	}
}

// TestCountTodosWithMissingFile tests countTodos gracefully handles missing file.
func TestCountTodosWithMissingFile(t *testing.T) {
	total, done := countTodos("/nonexistent/file.md")
	if total != 0 || done != 0 {
		t.Errorf("countTodos with missing file should return 0,0, got %d,%d", total, done)
	}
}

// TestReadPhaseStatus tests readPhaseStatus aggregates all phase files.
func TestReadPhaseStatus(t *testing.T) {
	tmpdir := t.TempDir()

	// Create multiple phase files
	phase1 := filepath.Join(tmpdir, "phase-01-setup.md")
	if err := os.WriteFile(phase1, []byte("# Phase 1\n- [x] Task1\n- [x] Task2\n- [ ] Task3"), 0644); err != nil {
		t.Fatalf("Failed to write phase file: %v", err)
	}

	phase2 := filepath.Join(tmpdir, "phase-02-implement.md")
	if err := os.WriteFile(phase2, []byte("# Phase 2\n- [x] Task1\n- [x] Task2\n- [x] Task3"), 0644); err != nil {
		t.Fatalf("Failed to write phase file: %v", err)
	}

	phase3 := filepath.Join(tmpdir, "phase-03-test.md")
	if err := os.WriteFile(phase3, []byte("# Phase 3\nNo todos here"), 0644); err != nil {
		t.Fatalf("Failed to write phase file: %v", err)
	}

	result := readPhaseStatus(tmpdir)
	if result == "" {
		t.Errorf("readPhaseStatus should return non-empty status")
	}
	if !strings.Contains(result, "phase-01-setup.md") {
		t.Errorf("readPhaseStatus should include phase-01")
	}
	if !strings.Contains(result, "2/3") {
		t.Errorf("readPhaseStatus should show phase-01 progress")
	}
	if !strings.Contains(result, "complete") {
		t.Errorf("readPhaseStatus should show complete status for phase-02")
	}
	if !strings.Contains(result, "no todos") {
		t.Errorf("readPhaseStatus should indicate no todos for phase-03")
	}
}

// TestReadPhaseStatusNoPhaseFiles tests readPhaseStatus with no phase files.
func TestReadPhaseStatusNoPhaseFiles(t *testing.T) {
	tmpdir := t.TempDir()
	result := readPhaseStatus(tmpdir)
	if result != "" {
		t.Errorf("readPhaseStatus with no phase files should return empty string, got %q", result)
	}
}

// TestBuildCompactionContextIntegration tests full integration.
func TestBuildCompactionContextIntegration(t *testing.T) {
	tmpdir := t.TempDir()

	// Create phase files
	phase1 := filepath.Join(tmpdir, "phase-01-research.md")
	if err := os.WriteFile(phase1, []byte("# Research\n- [x] Study requirements\n- [x] Review docs\n- [ ] Prototype"), 0644); err != nil {
		t.Fatalf("Failed to write phase file: %v", err)
	}

	phase2 := filepath.Join(tmpdir, "phase-02-implementation.md")
	if err := os.WriteFile(phase2, []byte("# Impl\n- [x] Write code\n- [x] Tests\n- [x] Docs"), 0644); err != nil {
		t.Fatalf("Failed to write phase file: %v", err)
	}

	// Create wisdom file
	wisdomPath := filepath.Join(tmpdir, ".wisdom.md")
	if err := os.WriteFile(wisdomPath, []byte("### Coder (14:30)\nPattern: use dependency injection\n### Tester (15:00)\nGotcha: watch memory leaks"), 0644); err != nil {
		t.Fatalf("Failed to write wisdom file: %v", err)
	}

	result := BuildCompactionContext(tmpdir, "")
	if result == "" {
		t.Errorf("BuildCompactionContext integration should produce output")
	}

	// Verify all sections present
	if !strings.Contains(result, "POST-COMPACTION RECOVERY CONTEXT") {
		t.Errorf("Missing recovery context header")
	}
	if !strings.Contains(result, "Plan Progress") {
		t.Errorf("Missing plan progress section")
	}
	if !strings.Contains(result, "Accumulated Learnings") {
		t.Errorf("Missing accumulated learnings section")
	}
	if !strings.Contains(result, "phase-01-research.md") {
		t.Errorf("Missing phase-01 status")
	}
	if !strings.Contains(result, "Pattern: use dependency injection") {
		t.Errorf("Missing wisdom content")
	}
}

// TestBuildCompactionContextSessionIDFallback tests wisdom resolution with sessionID.
func TestBuildCompactionContextSessionIDFallback(t *testing.T) {
	tmpdir := t.TempDir()
	sessionID := "test-session-789"

	// Create phase file
	phase1 := filepath.Join(tmpdir, "phase-01.md")
	if err := os.WriteFile(phase1, []byte("# Phase\n- [x] Done"), 0644); err != nil {
		t.Fatalf("Failed to write phase file: %v", err)
	}

	// Create temp wisdom file (not in plan dir)
	tempWisdomPath := filepath.Join(os.TempDir(), "sl-wisdom-"+sessionID+".md")
	defer os.Remove(tempWisdomPath)
	if err := os.WriteFile(tempWisdomPath, []byte("### Debugger (16:00)\nPattern: check logs first"), 0644); err != nil {
		t.Fatalf("Failed to write temp wisdom: %v", err)
	}

	result := BuildCompactionContext(tmpdir, sessionID)
	if result == "" {
		t.Errorf("BuildCompactionContext should fallback to sessionID wisdom")
	}
	if !strings.Contains(result, "Pattern: check logs first") {
		t.Errorf("BuildCompactionContext should include temp wisdom file content")
	}
}
