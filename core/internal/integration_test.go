// Package internal provides integration tests for the sc binary subsystems.
// Tests cover the full plan lifecycle: scaffold → hydrate → sync → status → validate.
package internal_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"solon-core/internal/config"
	"solon-core/internal/plan"
	"solon-core/internal/session"
	"solon-core/internal/task"
	"solon-core/internal/types"
	"solon-core/internal/workflow"
)

// makeTempConfig returns a minimal SLConfig pointing plans dir at tmpDir.
func makeTempConfig(plansDir string) types.SLConfig {
	cfg := config.DefaultConfig
	cfg.Paths.Plans = plansDir
	return cfg
}

// TestScaffoldHydrateSyncStatus covers the full plan lifecycle in sequence.
func TestScaffoldHydrateSyncStatus(t *testing.T) {
	// 1. Create temp dir
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	if err := os.MkdirAll(plansDir, 0755); err != nil {
		t.Fatalf("create plans dir: %v", err)
	}

	// 2. Set up minimal config
	cfg := makeTempConfig(plansDir)

	// 3. Scaffold a plan
	result, err := plan.ScaffoldPlan("test-feature", plan.ModeFast, 0, &cfg)
	if err != nil {
		t.Fatalf("ScaffoldPlan: %v", err)
	}
	planDir := result.PlanDir

	// 4. Verify plan.md and 3 phase files created
	if !contains(result.FilesCreated, "plan.md") {
		t.Errorf("plan.md not in FilesCreated: %v", result.FilesCreated)
	}
	phaseFiles := filterPhaseFiles(result.FilesCreated)
	if len(phaseFiles) != 3 {
		t.Errorf("expected 3 phase files for ModeFast, got %d: %v", len(phaseFiles), phaseFiles)
	}
	// Verify files exist on disk
	for _, f := range result.FilesCreated {
		fpath := filepath.Join(planDir, f)
		if _, err := os.Stat(fpath); os.IsNotExist(err) {
			t.Errorf("file not found on disk: %s", f)
		}
	}

	// 5. Hydrate plan — verify 3 tasks returned, correct blocking chain
	hydrateResult, err := task.HydratePlan(planDir)
	if err != nil {
		t.Fatalf("HydratePlan: %v", err)
	}
	if hydrateResult.Skipped {
		t.Fatalf("HydratePlan skipped: %s", hydrateResult.SkipReason)
	}
	if hydrateResult.TaskCount != 3 {
		t.Errorf("expected 3 tasks, got %d", hydrateResult.TaskCount)
	}
	// Verify sequential blocking chain
	tasks := hydrateResult.Tasks
	if len(tasks[0].BlockedBy) != 0 {
		t.Errorf("phase 1 should have no blockers, got %v", tasks[0].BlockedBy)
	}
	if len(tasks[1].BlockedBy) != 1 || tasks[1].BlockedBy[0] != tasks[0].Phase {
		t.Errorf("phase 2 should be blocked by phase 1, got %v", tasks[1].BlockedBy)
	}
	if len(tasks[2].BlockedBy) != 1 || tasks[2].BlockedBy[0] != tasks[1].Phase {
		t.Errorf("phase 3 should be blocked by phase 2, got %v", tasks[2].BlockedBy)
	}

	// 6. Workflow status — verify 3 pending phases, 0% progress
	status, err := workflow.GetStatus(planDir)
	if err != nil {
		t.Fatalf("GetStatus initial: %v", err)
	}
	if status.Phases.Total != 3 {
		t.Errorf("expected 3 total phases, got %d", status.Phases.Total)
	}
	if status.Phases.Completed != 0 {
		t.Errorf("expected 0 completed phases initially, got %d", status.Phases.Completed)
	}
	if status.Progress != 0 {
		t.Errorf("expected 0%% progress initially, got %d%%", status.Progress)
	}

	// Add a TODO item to phase 1 so sync has something to tick
	phase1File := filepath.Join(planDir, phaseFiles[0])
	existingContent, err := os.ReadFile(phase1File)
	if err != nil {
		t.Fatalf("read phase1: %v", err)
	}
	withTodo := string(existingContent) + "\n## TODO\n\n- [ ] implement step 1\n"
	if err := os.WriteFile(phase1File, []byte(withTodo), 0644); err != nil {
		t.Fatalf("write phase1 with todo: %v", err)
	}

	// 7. SyncCompletions — mark phase 1 done
	phase1Num := tasks[0].Phase
	syncResult, err := task.SyncCompletions(planDir, []int{phase1Num})
	if err != nil {
		t.Fatalf("SyncCompletions: %v", err)
	}
	if syncResult.CheckboxesUpdated == 0 {
		t.Errorf("expected at least 1 checkbox updated, got 0")
	}

	// 8. Workflow status — verify 1 completed, progress ~33%
	status2, err := workflow.GetStatus(planDir)
	if err != nil {
		t.Fatalf("GetStatus after sync: %v", err)
	}
	if status2.Phases.Completed != 1 {
		t.Errorf("expected 1 completed phase after sync, got %d", status2.Phases.Completed)
	}
	expectedProgress := (1 * 100) / 3 // 33
	if status2.Progress != expectedProgress {
		t.Errorf("expected ~%d%% progress, got %d%%", expectedProgress, status2.Progress)
	}

	// 9. ValidatePlan — verify valid
	vResult := plan.ValidatePlan(planDir)
	if !vResult.Valid {
		t.Errorf("plan should be valid, errors: %v", vResult.Errors)
	}
	if vResult.Stats.PhaseCount != 3 {
		t.Errorf("expected 3 phases in validation, got %d", vResult.Stats.PhaseCount)
	}
}

// TestResolverWithSession verifies session-based plan resolution.
func TestResolverWithSession(t *testing.T) {
	// 1. Create temp plans dir with a plan
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	planDir := filepath.Join(plansDir, "260315-1200-my-feature")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("create plan dir: %v", err)
	}
	// Write minimal plan.md
	planMd := "---\nstatus: in-progress\n---\n# Test Plan\n"
	if err := os.WriteFile(filepath.Join(planDir, "plan.md"), []byte(planMd), 0644); err != nil {
		t.Fatalf("write plan.md: %v", err)
	}

	// 2. Write session state with ActivePlan pointing to the plan dir
	sessionID := "test-session-" + strings.ReplaceAll(time.Now().Format("150405.000"), ".", "")
	activePlan := planDir
	state := &types.SessionState{
		SessionOrigin: tmpDir,
		ActivePlan:    &activePlan,
		Timestamp:     time.Now().Unix(),
		Source:        "test",
	}
	if !session.WriteSessionState(sessionID, state) {
		t.Fatal("WriteSessionState returned false")
	}
	// Clean up session temp file after test
	t.Cleanup(func() { os.Remove(session.GetSessionTempPath(sessionID)) })

	// 3. Call ResolvePlanPath with session ID
	cfg := makeTempConfig(plansDir)
	resolution := plan.ResolvePlanPath(sessionID, &cfg)

	// 4. Verify resolves correctly
	if resolution.Path == "" {
		t.Fatal("expected non-empty resolution path")
	}
	if resolution.ResolvedBy != "session" {
		t.Errorf("expected resolvedBy=session, got %q", resolution.ResolvedBy)
	}
	if resolution.Path != planDir {
		t.Errorf("expected path %q, got %q", planDir, resolution.Path)
	}
}

// TestNamingPatterns verifies SanitizeSlug, FormatDate, and BuildPlanDirName.
func TestNamingPatterns(t *testing.T) {
	// SanitizeSlug normal case
	tests := []struct {
		input    string
		wantEmpty bool
		desc     string
	}{
		{"my-feature", false, "normal kebab slug"},
		{"My Feature", false, "spaces converted to dashes"},
		{"", true, "empty slug returns empty"},
		{"hello@world!", false, "special chars stripped"},
		{strings.Repeat("a", 120), false, "long slug truncated to 100"},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got := plan.SanitizeSlug(tc.input)
			if tc.wantEmpty && got != "" {
				t.Errorf("SanitizeSlug(%q) = %q, want empty", tc.input, got)
			}
			if !tc.wantEmpty && got == "" {
				t.Errorf("SanitizeSlug(%q) = empty, want non-empty", tc.input)
			}
			if len(got) > 100 {
				t.Errorf("SanitizeSlug result too long: %d > 100", len(got))
			}
		})
	}

	// FormatDate produces non-empty output
	t.Run("FormatDate basic", func(t *testing.T) {
		result := plan.FormatDate("YYMMDD-HHmm")
		if result == "" {
			t.Error("FormatDate returned empty string")
		}
		if strings.Contains(result, "YY") || strings.Contains(result, "MM") {
			t.Errorf("FormatDate did not substitute tokens: %q", result)
		}
	})

	// BuildPlanDirName produces a non-empty dir name containing the slug
	t.Run("BuildPlanDirName with slug", func(t *testing.T) {
		cfg := config.DefaultConfig
		dirName := plan.BuildPlanDirName(cfg.Plan, "", "my-feature")
		if dirName == "" {
			t.Error("BuildPlanDirName returned empty string")
		}
		if !strings.Contains(dirName, "my-feature") {
			t.Errorf("BuildPlanDirName %q does not contain slug 'my-feature'", dirName)
		}
	})

	// BuildPlanDirName with empty slug falls back to "untitled"
	t.Run("BuildPlanDirName empty slug", func(t *testing.T) {
		cfg := config.DefaultConfig
		dirName := plan.BuildPlanDirName(cfg.Plan, "", "")
		if !strings.Contains(dirName, "untitled") {
			t.Errorf("BuildPlanDirName with empty slug should contain 'untitled', got %q", dirName)
		}
	})

	// BuildPlanDirName with special chars in slug
	t.Run("BuildPlanDirName special chars", func(t *testing.T) {
		cfg := config.DefaultConfig
		dirName := plan.BuildPlanDirName(cfg.Plan, "", "feat: add <new> feature!")
		if dirName == "" {
			t.Error("BuildPlanDirName returned empty for special char slug")
		}
		// Should not contain raw special chars that are invalid in filenames
		invalidChars := []string{"<", ">", ":", "\"", "\\", "|", "?", "*"}
		for _, ch := range invalidChars {
			if strings.Contains(dirName, ch) {
				t.Errorf("BuildPlanDirName contains invalid filename char %q in %q", ch, dirName)
			}
		}
	})
}

// --- helpers ---

// contains returns true if slice contains target.
func contains(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

// filterPhaseFiles returns only phase-*.md entries from a slice.
func filterPhaseFiles(files []string) []string {
	var out []string
	for _, f := range files {
		if strings.HasPrefix(f, "phase-") && strings.HasSuffix(f, ".md") {
			out = append(out, f)
		}
	}
	return out
}
