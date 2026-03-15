// Package logging provides structured JSONL logging for hook scripts.
// Log file: ~/.solon/logs/hook-log.jsonl
// Rotation: keep last 500 lines when file exceeds 1000 lines.
// All errors fail silently — never crash a hook.
package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxLines   = 1000
	truncateTo = 500
)

// logDir uses $HOME/.solon/logs/ for a stable, absolute log location.
// Previous approach used runtime.Caller(0) which returns module-relative
// paths in compiled binaries, causing spurious solon-hooks/ dirs in CWD.
var logDir string
var logFile string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		logDir = ".logs"
	} else {
		logDir = filepath.Join(home, ".solon", "logs")
	}
	logFile = filepath.Join(logDir, "hook-log.jsonl")
}

// LogData holds optional fields for a hook log entry.
type LogData struct {
	Tool   string
	Dur    int64
	Status string
	Exit   int
	Error  string
}

// logEntry is the JSON structure written per line.
type logEntry struct {
	Ts     string `json:"ts"`
	Hook   string `json:"hook"`
	Tool   string `json:"tool"`
	Dur    int64  `json:"dur"`
	Status string `json:"status"`
	Exit   int    `json:"exit"`
	Error  string `json:"error"`
}

func ensureLogDir() {
	_ = os.MkdirAll(logDir, 0755)
}

func rotateIfNeeded() {
	data, err := os.ReadFile(logFile)
	if err != nil {
		return
	}
	lines := filterNonEmpty(strings.Split(string(data), "\n"))
	if len(lines) >= maxLines {
		kept := lines
		if len(lines) > truncateTo {
			kept = lines[len(lines)-truncateTo:]
		}
		_ = os.WriteFile(logFile, []byte(strings.Join(kept, "\n")+"\n"), 0644)
	}
}

func filterNonEmpty(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

// LogHook appends a structured entry to the JSONL log file.
func LogHook(hookName string, data LogData) {
	defer func() { recover() }() // never crash

	ensureLogDir()
	rotateIfNeeded()

	status := data.Status
	if status == "" {
		status = "ok"
	}
	entry := logEntry{
		Ts:     time.Now().UTC().Format(time.RFC3339),
		Hook:   hookName,
		Tool:   data.Tool,
		Dur:    data.Dur,
		Status: status,
		Exit:   data.Exit,
		Error:  data.Error,
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return
	}

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(string(line) + "\n")
}

// HookTimer measures hook execution duration.
type HookTimer struct {
	hookName string
	start    time.Time
}

// CreateHookTimer starts a timer for the named hook.
func CreateHookTimer(hookName string) *HookTimer {
	return &HookTimer{hookName: hookName, start: time.Now()}
}

// End records the elapsed duration and logs the hook event.
func (t *HookTimer) End(data LogData) {
	data.Dur = time.Since(t.start).Milliseconds()
	LogHook(t.hookName, data)
}
