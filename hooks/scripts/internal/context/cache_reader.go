// Context cache reader — reads context window % from statusline temp file.
package context

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// ReadContextPercent reads context usage % from /tmp/sl-context-{sessionID}.json.
// Returns (percent, true) on success; (0, false) if unavailable or stale (>5 min).
func ReadContextPercent(sessionID string) (int, bool) {
	path := filepath.Join(os.TempDir(), "sl-context-"+sessionID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	var cache struct {
		Percent   int   `json:"percent"`
		Timestamp int64 `json:"timestamp"`
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return 0, false
	}
	// Treat as stale if older than 5 minutes
	if time.Now().UnixMilli()-cache.Timestamp > 300_000 {
		return 0, false
	}
	return cache.Percent, true
}
