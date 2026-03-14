package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCountIncompleteTodosWithNoTodos verifies a file with no todos returns 0.
func TestCountIncompleteTodosWithNoTodos(t *testing.T) {
	f, err := os.CreateTemp("", "phase_*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.WriteString("# Phase 1\n\n- [x] Completed task\n- [x] Another done\n")
	f.Close()

	count := countIncompleteTodos(f.Name())
	if count != 0 {
		t.Errorf("expected 0 incomplete todos, got %d", count)
	}
}

// TestCountIncompleteTodosWithMixedTodos verifies only unchecked boxes are counted.
func TestCountIncompleteTodosWithMixedTodos(t *testing.T) {
	f, err := os.CreateTemp("", "phase_*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.WriteString("# Phase 2\n\n- [x] Done\n- [ ] Incomplete one\n- [ ] Incomplete two\n- [x] Also done\n")
	f.Close()

	count := countIncompleteTodos(f.Name())
	if count != 2 {
		t.Errorf("expected 2 incomplete todos, got %d", count)
	}
}

// TestCountIncompleteTodosWithMissingFile verifies a missing file returns 0 gracefully.
func TestCountIncompleteTodosWithMissingFile(t *testing.T) {
	count := countIncompleteTodos("/nonexistent/path/phase-99.md")
	if count != 0 {
		t.Errorf("missing file should return 0, got %d", count)
	}
}

// TestCountIncompleteTodosAllIncomplete verifies all-incomplete file is counted correctly.
func TestCountIncompleteTodosAllIncomplete(t *testing.T) {
	f, err := os.CreateTemp("", "phase_*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.WriteString("- [ ] Task A\n- [ ] Task B\n- [ ] Task C\n")
	f.Close()

	count := countIncompleteTodos(f.Name())
	if count != 3 {
		t.Errorf("expected 3 incomplete todos, got %d", count)
	}
}

// TestTodoEnforcerCooldownExpiredWhenMissing verifies missing cooldown file returns true (expired).
func TestTodoEnforcerCooldownExpiredWhenMissing(t *testing.T) {
	// Rename any existing cooldown file temporarily
	path := filepath.Join(os.TempDir(), todoEnforcerCooldownFile)
	backup := path + ".bak"
	_ = os.Rename(path, backup)
	defer func() {
		_ = os.Rename(backup, path)
	}()

	if !todoEnforcerCooldownExpired() {
		t.Errorf("missing cooldown file should mean expired (return true)")
	}
}
