package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"solon-hooks/internal/hookio"
)

// TestGetSessionTempPath returns a path in temp dir with session ID embedded.
func TestGetSessionTempPath(t *testing.T) {
	sessionID := "test-session-abc"
	got := GetSessionTempPath(sessionID)

	if !strings.Contains(got, sessionID) {
		t.Errorf("path %q does not contain session ID %q", got, sessionID)
	}
	if !strings.HasPrefix(got, os.TempDir()) {
		t.Errorf("path %q does not start with temp dir %q", got, os.TempDir())
	}
	if !strings.HasSuffix(got, ".json") {
		t.Errorf("path %q does not end with .json", got)
	}

	expected := filepath.Join(os.TempDir(), "sl-session-test-session-abc.json")
	if got != expected {
		t.Errorf("GetSessionTempPath = %q, want %q", got, expected)
	}
}

// TestWriteReadRoundtrip writes state then reads it back and verifies equality.
func TestWriteReadRoundtrip(t *testing.T) {
	sessionID := "roundtrip-test-session"
	// Clean up after test
	defer os.Remove(GetSessionTempPath(sessionID))

	plan := "plans/260314-test/plan.md"
	state := &hookio.SessionState{
		SessionOrigin: "startup",
		ActivePlan:    &plan,
		SuggestedPlan: nil,
		Timestamp:     1234567890,
		Source:        "session-start",
	}

	ok := WriteSessionState(sessionID, state)
	if !ok {
		t.Fatal("WriteSessionState returned false, expected true")
	}

	got := ReadSessionState(sessionID)
	if got == nil {
		t.Fatal("ReadSessionState returned nil after write")
	}
	if got.SessionOrigin != state.SessionOrigin {
		t.Errorf("SessionOrigin = %q, want %q", got.SessionOrigin, state.SessionOrigin)
	}
	if got.ActivePlan == nil || *got.ActivePlan != plan {
		t.Errorf("ActivePlan = %v, want %q", got.ActivePlan, plan)
	}
	if got.Timestamp != state.Timestamp {
		t.Errorf("Timestamp = %d, want %d", got.Timestamp, state.Timestamp)
	}
	if got.Source != state.Source {
		t.Errorf("Source = %q, want %q", got.Source, state.Source)
	}
}

// TestWriteSessionState_EmptyID returns false for empty session ID.
func TestWriteSessionState_EmptyID(t *testing.T) {
	state := &hookio.SessionState{SessionOrigin: "test"}
	if WriteSessionState("", state) {
		t.Error("WriteSessionState with empty ID should return false")
	}
}

// TestWriteSessionState_NilState returns false for nil state.
func TestWriteSessionState_NilState(t *testing.T) {
	if WriteSessionState("some-session", nil) {
		t.Error("WriteSessionState with nil state should return false")
	}
}

// TestReadSessionState_EmptyID returns nil for empty session ID.
func TestReadSessionState_EmptyID(t *testing.T) {
	if ReadSessionState("") != nil {
		t.Error("ReadSessionState with empty ID should return nil")
	}
}

// TestReadSessionState_NonExistent returns nil for a session that was never written.
func TestReadSessionState_NonExistent(t *testing.T) {
	sessionID := "nonexistent-session-xyz-99999"
	// Ensure no leftover file
	os.Remove(GetSessionTempPath(sessionID))

	if ReadSessionState(sessionID) != nil {
		t.Error("ReadSessionState for non-existent session should return nil")
	}
}
