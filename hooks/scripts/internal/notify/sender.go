// sender.go: HTTP POST with 5-minute per-provider error throttling.
package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const throttleFile = "sl-noti-throttle.json"
const throttleDuration = 5 * time.Minute

// SendResult holds the outcome of a notification send attempt.
type SendResult struct {
	Success   bool
	Error     string
	Throttled bool
}

// throttlePath returns the path to the throttle state file.
func throttlePath() string {
	return filepath.Join(os.TempDir(), throttleFile)
}

// loadThrottleState reads the throttle state map (provider -> last error timestamp ms).
func loadThrottleState() map[string]int64 {
	data, err := os.ReadFile(throttlePath())
	if err != nil {
		return make(map[string]int64)
	}
	var state map[string]int64
	if err := json.Unmarshal(data, &state); err != nil {
		os.Stderr.WriteString("[sender] Throttle file corrupted, resetting\n")
		return make(map[string]int64)
	}
	return state
}

// saveThrottleState writes the throttle state map to disk.
func saveThrottleState(state map[string]int64) {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(throttlePath(), data, 0644); err != nil {
		os.Stderr.WriteString("[sender] Failed to save throttle state: " + err.Error() + "\n")
	}
}

// isThrottled returns true if provider had an error within the throttle window.
func isThrottled(provider string) bool {
	state := loadThrottleState()
	last, ok := state[provider]
	if !ok {
		return false
	}
	elapsed := time.Since(time.UnixMilli(last))
	return elapsed < throttleDuration
}

// recordError records the current time as the last error for provider.
func recordError(provider string) {
	state := loadThrottleState()
	state[provider] = time.Now().UnixMilli()
	saveThrottleState(state)
}

// clearThrottle removes the throttle entry for provider on success.
func clearThrottle(provider string) {
	state := loadThrottleState()
	if _, ok := state[provider]; ok {
		delete(state, provider)
		saveThrottleState(state)
	}
}

// Send performs an HTTP POST to url with JSON body and optional headers.
// Applies per-provider error throttling to avoid rapid-fire retries.
func Send(provider, url string, body interface{}, headers map[string]string) *SendResult {
	if isThrottled(provider) {
		return &SendResult{Throttled: true}
	}

	raw, err := json.Marshal(body)
	if err != nil {
		recordError(provider)
		return &SendResult{Error: "marshal: " + err.Error()}
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(raw))
	if err != nil {
		recordError(provider)
		return &SendResult{Error: "request: " + err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		recordError(provider)
		os.Stderr.WriteString(fmt.Sprintf("[sender] %s network error: %s\n", provider, err.Error()))
		return &SendResult{Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 100))
		errMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
		recordError(provider)
		os.Stderr.WriteString(fmt.Sprintf("[sender] %s failed: %s\n", provider, errMsg))
		return &SendResult{Error: errMsg}
	}

	clearThrottle(provider)
	return &SendResult{Success: true}
}
