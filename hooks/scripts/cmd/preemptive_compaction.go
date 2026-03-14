// PostToolUse hook: warns when context window is near capacity, suggesting /compact.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"solon-hooks/internal/config"
	slctx "solon-hooks/internal/context"
	"solon-hooks/internal/hookio"
)

const (
	compactionThreshold  = 78
	compactionCooldownMs = 5 * 60 * 1000 // 5 minutes
	compactionCooldownFile = "sl-compaction-reminded.json"
)

var preemptiveCompactionCmd = &cobra.Command{
	Use:   "preemptive-compaction",
	Short: "Suggest /compact when context usage exceeds threshold",
	RunE:  runPreemptiveCompaction,
}

func runPreemptiveCompaction(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("preemptive-compaction") {
		os.Exit(0)
	}

	var input hookio.PostToolUseInput
	if err := hookio.ReadInput(&input); err != nil {
		hookio.WriteOutput(map[string]interface{}{"continue": true})
		return nil
	}

	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = os.Getenv("SL_SESSION_ID")
	}
	if sessionID == "" {
		hookio.WriteOutput(map[string]interface{}{"continue": true})
		return nil
	}

	percent, ok := slctx.ReadContextPercent(sessionID)
	if !ok || percent < compactionThreshold {
		hookio.WriteOutput(map[string]interface{}{"continue": true})
		return nil
	}

	if !compactionCooldownExpired() {
		hookio.WriteOutput(map[string]interface{}{"continue": true})
		return nil
	}
	writeCompactionCooldown()

	hookio.WriteOutput(map[string]interface{}{
		"continue": true,
		"additionalContext": fmt.Sprintf(
			"\n[Context Warning] Context window at %d%%. Consider running /compact soon to avoid overflow. "+
				"Finish your current atomic task first, then compact.", percent),
	})
	return nil
}

type compactionCooldownState struct {
	Timestamp int64 `json:"timestamp"`
}

func compactionCooldownExpired() bool {
	path := filepath.Join(os.TempDir(), compactionCooldownFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	var state compactionCooldownState
	if err := json.Unmarshal(data, &state); err != nil {
		return true
	}
	return time.Now().UnixMilli()-state.Timestamp > compactionCooldownMs
}

func writeCompactionCooldown() {
	path := filepath.Join(os.TempDir(), compactionCooldownFile)
	data, _ := json.Marshal(compactionCooldownState{Timestamp: time.Now().UnixMilli()})
	_ = os.WriteFile(path, data, 0644)
}
