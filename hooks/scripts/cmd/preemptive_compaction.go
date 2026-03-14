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
	compactionWarnThreshold   = 65
	compactionStrongThreshold = 75
	compactionUrgentThreshold = 85
	compactionCooldownMs      = 3 * 60 * 1000 // 3 minutes
	compactionCooldownFile    = "sl-compaction-reminded.json"
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
	if !ok || percent < compactionWarnThreshold {
		hookio.WriteOutput(map[string]interface{}{"continue": true})
		return nil
	}

	if !compactionCooldownExpired() {
		hookio.WriteOutput(map[string]interface{}{"continue": true})
		return nil
	}
	writeCompactionCooldown()

	msg := compactionMessage(percent)
	hookio.WriteOutput(map[string]interface{}{
		"continue":          true,
		"additionalContext": msg,
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

// compactionMessage returns escalating urgency messages based on context %.
func compactionMessage(percent int) string {
	switch {
	case percent >= compactionUrgentThreshold:
		return fmt.Sprintf(
			"\n[URGENT] Context at %d%% — STOP current work. Run /compact NOW or session will overflow. "+
				"Save any critical context to files before compacting.", percent)
	case percent >= compactionStrongThreshold:
		return fmt.Sprintf(
			"\n[Context Warning] Context at %d%%. Finish current atomic task, then run /compact. "+
				"Avoid spawning new subagents until compacted.", percent)
	default:
		return fmt.Sprintf(
			"\n[Context Notice] Context at %d%%. Plan to /compact soon. Keep responses concise.", percent)
	}
}
