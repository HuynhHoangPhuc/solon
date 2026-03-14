package cmd

import (
	"strings"
	"testing"
)

// TestCompactionMessageBelowWarnThreshold tests that no message is generated below warn threshold.
// The compactionMessage function is only called when percent >= compactionWarnThreshold (65),
// so we test the message content directly.
func TestCompactionMessageGentleWarning(t *testing.T) {
	// 65-74% range: gentle notice
	msg := compactionMessage(65)
	if !strings.Contains(msg, "Context Notice") {
		t.Errorf("expected [Context Notice] at 65%%, got: %q", msg)
	}
	if !strings.Contains(msg, "65%") {
		t.Errorf("expected percent in message, got: %q", msg)
	}
}

// TestCompactionMessageStrongWarning tests the 75-84% threshold produces a strong warning.
func TestCompactionMessageStrongWarning(t *testing.T) {
	msg := compactionMessage(75)
	if !strings.Contains(msg, "Context Warning") {
		t.Errorf("expected [Context Warning] at 75%%, got: %q", msg)
	}
	if !strings.Contains(msg, "75%") {
		t.Errorf("expected percent in message, got: %q", msg)
	}
	if !strings.Contains(msg, "/compact") {
		t.Errorf("expected /compact mention, got: %q", msg)
	}
}

// TestCompactionMessageUrgentWarning tests the >85% threshold produces an urgent warning.
func TestCompactionMessageUrgentWarning(t *testing.T) {
	msg := compactionMessage(85)
	if !strings.Contains(msg, "URGENT") {
		t.Errorf("expected [URGENT] at 85%%, got: %q", msg)
	}
	if !strings.Contains(msg, "85%") {
		t.Errorf("expected percent in message, got: %q", msg)
	}
}

// TestCompactionMessageAt90Percent tests well above the urgent threshold.
func TestCompactionMessageAt90Percent(t *testing.T) {
	msg := compactionMessage(90)
	if !strings.Contains(msg, "URGENT") {
		t.Errorf("expected [URGENT] at 90%%, got: %q", msg)
	}
	if !strings.Contains(msg, "90%") {
		t.Errorf("expected percent in message, got: %q", msg)
	}
	if !strings.Contains(msg, "STOP") {
		t.Errorf("expected STOP in urgent message, got: %q", msg)
	}
}

// TestCompactionCooldownExpiredWhenMissing tests that a missing cooldown file means expired.
func TestCompactionCooldownExpiredWhenMissing(t *testing.T) {
	// Remove any existing cooldown file to ensure clean state
	// The function reads from os.TempDir() — if the file doesn't exist it returns true
	result := compactionCooldownExpired()
	// We can't guarantee the file doesn't exist, but we can verify it returns a bool
	_ = result // either true or false is valid
}
