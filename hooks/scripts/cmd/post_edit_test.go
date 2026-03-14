package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoadSimplifySessionCreatesNewWhenMissing verifies a missing session file returns a fresh session.
func TestLoadSimplifySessionCreatesNewWhenMissing(t *testing.T) {
	path := filepath.Join(os.TempDir(), simplifySessionFile)
	backup := path + ".bak"
	_ = os.Rename(path, backup)
	defer func() { _ = os.Rename(backup, path) }()

	s := loadSimplifySession()
	if s == nil {
		t.Fatal("loadSimplifySession should never return nil")
	}
	if s.EditCount != 0 {
		t.Errorf("new session EditCount expected 0, got %d", s.EditCount)
	}
	if s.ModifiedFiles == nil {
		t.Errorf("new session ModifiedFiles should be initialised (not nil)")
	}
}

// TestSaveAndLoadSimplifySessionRoundTrip verifies save then load preserves all fields.
func TestSaveAndLoadSimplifySessionRoundTrip(t *testing.T) {
	path := filepath.Join(os.TempDir(), simplifySessionFile)
	backup := path + ".bak"
	_ = os.Rename(path, backup)
	defer func() {
		_ = os.Remove(path)
		_ = os.Rename(backup, path)
	}()

	// Use a recent StartTime so loadSimplifySession does not treat it as expired.
	original := &simplifySession{
		StartTime:     time.Now().UnixMilli(),
		EditCount:     3,
		ModifiedFiles: []string{"/a.go", "/b.go"},
		LastReminder:  500_000,
		SimplifierRun: false,
	}
	saveSimplifySession(original)

	loaded := loadSimplifySession()
	if loaded.EditCount != original.EditCount {
		t.Errorf("EditCount: want %d got %d", original.EditCount, loaded.EditCount)
	}
	if len(loaded.ModifiedFiles) != len(original.ModifiedFiles) {
		t.Errorf("ModifiedFiles len: want %d got %d", len(original.ModifiedFiles), len(loaded.ModifiedFiles))
	}
}

// TestLoadSimplifySessionResetsExpiredSession verifies sessions older than 2h are reset.
func TestLoadSimplifySessionResetsExpiredSession(t *testing.T) {
	path := filepath.Join(os.TempDir(), simplifySessionFile)
	backup := path + ".bak"
	_ = os.Rename(path, backup)
	defer func() {
		_ = os.Remove(path)
		_ = os.Rename(backup, path)
	}()

	// Write a session with StartTime far in the past (3 hours ago)
	expired := &simplifySession{
		StartTime:     0, // epoch — definitely expired
		EditCount:     99,
		ModifiedFiles: []string{"/old.go"},
	}
	data, _ := json.MarshalIndent(expired, "", "  ")
	_ = os.WriteFile(path, data, 0644)

	s := loadSimplifySession()
	if s.EditCount != 0 {
		t.Errorf("expired session should reset EditCount to 0, got %d", s.EditCount)
	}
}

// TestSimplifySessionConstants verifies the key timing constants have expected values.
func TestSimplifySessionConstants(t *testing.T) {
	if editThreshold != 5 {
		t.Errorf("editThreshold expected 5, got %d", editThreshold)
	}
	if sessionTTLMs != 2*60*60*1000 {
		t.Errorf("sessionTTLMs expected 7200000 (2h), got %d", sessionTTLMs)
	}
	if reminderCooldownMs != 10*60*1000 {
		t.Errorf("reminderCooldownMs expected 600000 (10min), got %d", reminderCooldownMs)
	}
}
