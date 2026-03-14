package hookio

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

// redirectStdin replaces os.Stdin with a pipe fed by content, returns restore func.
func redirectStdin(t *testing.T, content string) func() {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdin
	os.Stdin = r
	go func() {
		w.WriteString(content) //nolint:errcheck
		w.Close()
	}()
	return func() { os.Stdin = orig; r.Close() }
}

// captureStdout replaces os.Stdout with a pipe, returns reader and restore func.
func captureStdout(t *testing.T) (*os.File, func() string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	return r, func() string {
		w.Close()
		os.Stdout = orig
		var buf bytes.Buffer
		io.Copy(&buf, r) //nolint:errcheck
		r.Close()
		return buf.String()
	}
}

// captureStderr replaces os.Stderr with a pipe, returns restore func that returns captured output.
func captureStderr(t *testing.T) func() string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	return func() string {
		w.Close()
		os.Stderr = orig
		var buf bytes.Buffer
		io.Copy(&buf, r) //nolint:errcheck
		r.Close()
		return buf.String()
	}
}

// TestReadInput_ValidJSON unmarshals valid JSON from stdin correctly.
func TestReadInput_ValidJSON(t *testing.T) {
	payload := `{"session_id":"abc123","hook_event_name":"PreToolUse"}`
	restore := redirectStdin(t, payload)
	defer restore()

	var input HookInput
	if err := ReadInput(&input); err != nil {
		t.Fatalf("ReadInput returned error: %v", err)
	}
	if input.SessionID != "abc123" {
		t.Errorf("SessionID = %q, want %q", input.SessionID, "abc123")
	}
	if input.HookEventName != "PreToolUse" {
		t.Errorf("HookEventName = %q, want %q", input.HookEventName, "PreToolUse")
	}
}

// TestReadInput_EmptyStdin returns error for empty stdin.
func TestReadInput_EmptyStdin(t *testing.T) {
	restore := redirectStdin(t, "")
	defer restore()

	var input HookInput
	err := ReadInput(&input)
	if err == nil {
		t.Error("expected error for empty stdin, got nil")
	}
}

// TestWriteOutput encodes value as JSON to stdout.
func TestWriteOutput(t *testing.T) {
	_, read := captureStdout(t)

	cont := true
	WriteOutput(HookOutput{Continue: &cont, AdditionalContext: "hello"})

	out := read()
	if !strings.Contains(out, `"additionalContext"`) {
		t.Errorf("stdout missing additionalContext field, got: %s", out)
	}

	var decoded HookOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &decoded); err != nil {
		t.Errorf("stdout is not valid JSON: %v", err)
	}
	if decoded.AdditionalContext != "hello" {
		t.Errorf("AdditionalContext = %q, want %q", decoded.AdditionalContext, "hello")
	}
}

// TestWriteContext writes plain text to stdout.
func TestWriteContext(t *testing.T) {
	_, read := captureStdout(t)

	WriteContext("plain context text")

	out := read()
	if out != "plain context text" {
		t.Errorf("WriteContext output = %q, want %q", out, "plain context text")
	}
}

// TestLog writes formatted message to stderr.
func TestLog(t *testing.T) {
	read := captureStderr(t)

	Log("my-hook", "something happened")

	out := read()
	expected := "[my-hook] something happened\n"
	if out != expected {
		t.Errorf("Log output = %q, want %q", out, expected)
	}
}
