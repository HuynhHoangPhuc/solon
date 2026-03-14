package logging

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// overrideLogFile redirects logFile/logDir to a temp location for test isolation.
// Returns restore func.
func overrideLogFile(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	origDir := logDir
	origFile := logFile
	logDir = tmpDir
	logFile = tmpDir + "/hook-log.jsonl"
	return logFile, func() {
		logDir = origDir
		logFile = origFile
	}
}

// TestLogHook_WritesEntry verifies LogHook appends a valid JSONL entry.
func TestLogHook_WritesEntry(t *testing.T) {
	lf, restore := overrideLogFile(t)
	defer restore()

	LogHook("test-hook", LogData{
		Tool:   "Bash",
		Dur:    42,
		Status: "ok",
	})

	data, err := os.ReadFile(lf)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	lines := filterNonEmpty(strings.Split(string(data), "\n"))
	if len(lines) == 0 {
		t.Fatal("expected at least one log line, got none")
	}

	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("log line is not valid JSON: %v\nline: %s", err, lines[0])
	}
	if entry["hook"] != "test-hook" {
		t.Errorf("hook = %v, want %q", entry["hook"], "test-hook")
	}
	if entry["tool"] != "Bash" {
		t.Errorf("tool = %v, want %q", entry["tool"], "Bash")
	}
	if entry["status"] != "ok" {
		t.Errorf("status = %v, want %q", entry["status"], "ok")
	}
}

// TestLogHook_DefaultStatus uses "ok" when Status is empty.
func TestLogHook_DefaultStatus(t *testing.T) {
	lf, restore := overrideLogFile(t)
	defer restore()

	LogHook("hook-default", LogData{}) // no Status set

	data, _ := os.ReadFile(lf)
	lines := filterNonEmpty(strings.Split(string(data), "\n"))
	if len(lines) == 0 {
		t.Fatal("expected log entry")
	}
	var entry map[string]interface{}
	json.Unmarshal([]byte(lines[0]), &entry) //nolint:errcheck
	if entry["status"] != "ok" {
		t.Errorf("default status = %v, want %q", entry["status"], "ok")
	}
}

// TestLogHook_TimestampFormat verifies ts field is a valid RFC3339 timestamp.
func TestLogHook_TimestampFormat(t *testing.T) {
	lf, restore := overrideLogFile(t)
	defer restore()

	LogHook("ts-hook", LogData{Status: "ok"})

	data, _ := os.ReadFile(lf)
	lines := filterNonEmpty(strings.Split(string(data), "\n"))
	if len(lines) == 0 {
		t.Fatal("expected log entry")
	}
	var entry map[string]interface{}
	json.Unmarshal([]byte(lines[0]), &entry) //nolint:errcheck

	tsVal, ok := entry["ts"].(string)
	if !ok || tsVal == "" {
		t.Fatal("ts field missing or not a string")
	}
	if _, err := time.Parse(time.RFC3339, tsVal); err != nil {
		t.Errorf("ts %q is not valid RFC3339: %v", tsVal, err)
	}
}

// TestCreateHookTimer_End verifies the timer logs a positive duration.
func TestCreateHookTimer_End(t *testing.T) {
	lf, restore := overrideLogFile(t)
	defer restore()

	timer := CreateHookTimer("timer-hook")
	time.Sleep(2 * time.Millisecond) // ensure measurable duration
	timer.End(LogData{Status: "ok"})

	data, _ := os.ReadFile(lf)
	lines := filterNonEmpty(strings.Split(string(data), "\n"))
	if len(lines) == 0 {
		t.Fatal("expected log entry from timer")
	}
	var entry map[string]interface{}
	json.Unmarshal([]byte(lines[0]), &entry) //nolint:errcheck

	if entry["hook"] != "timer-hook" {
		t.Errorf("hook = %v, want %q", entry["hook"], "timer-hook")
	}
	dur, _ := entry["dur"].(float64)
	if dur < 0 {
		t.Errorf("dur = %v, expected non-negative", dur)
	}
}

// TestLogHook_MultipleEntries appends multiple entries to the same file.
func TestLogHook_MultipleEntries(t *testing.T) {
	lf, restore := overrideLogFile(t)
	defer restore()

	LogHook("hook-a", LogData{Status: "ok"})
	LogHook("hook-b", LogData{Status: "error", Error: "something failed"})

	data, _ := os.ReadFile(lf)
	lines := filterNonEmpty(strings.Split(string(data), "\n"))
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(lines))
	}

	var entry1, entry2 map[string]interface{}
	json.Unmarshal([]byte(lines[0]), &entry1) //nolint:errcheck
	json.Unmarshal([]byte(lines[1]), &entry2) //nolint:errcheck

	if entry1["hook"] != "hook-a" {
		t.Errorf("first entry hook = %v, want %q", entry1["hook"], "hook-a")
	}
	if entry2["hook"] != "hook-b" {
		t.Errorf("second entry hook = %v, want %q", entry2["hook"], "hook-b")
	}
	if entry2["error"] != "something failed" {
		t.Errorf("second entry error = %v, want %q", entry2["error"], "something failed")
	}
}
