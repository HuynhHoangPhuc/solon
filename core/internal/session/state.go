// Package session manages per-session state persisted to /tmp/sl-session-{id}.json.
// Writes are atomic (temp file + rename) to prevent corruption.
// Ported from solon-hooks/internal/session/state.go — keep in sync.
package session

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"solon-core/internal/types"
)

// GetSessionTempPath returns the temp file path for a session ID.
func GetSessionTempPath(sessionID string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("sl-session-%s.json", sessionID))
}

// ReadSessionState reads session state from the temp file.
// Returns nil if sessionID is empty, file does not exist, or parse fails.
func ReadSessionState(sessionID string) *types.SessionState {
	if sessionID == "" {
		return nil
	}
	data, err := os.ReadFile(GetSessionTempPath(sessionID))
	if err != nil {
		return nil
	}
	var state types.SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}
	return &state
}

// WriteSessionState atomically writes session state to the temp file.
// Returns true on success.
func WriteSessionState(sessionID string, state *types.SessionState) bool {
	if sessionID == "" || state == nil {
		return false
	}
	target := GetSessionTempPath(sessionID)
	tmp := fmt.Sprintf("%s.%x", target, rand.Int63())

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return false
	}
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return false
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return false
	}
	return true
}
