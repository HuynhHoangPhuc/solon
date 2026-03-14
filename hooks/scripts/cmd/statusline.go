// statusline.go: statusline subcommand — renders Claude Code status display.
// Reads JSON from stdin, builds render context, outputs lines to stdout.
// Falls back to "📁 dir" on any error so Claude Code always gets a valid line.
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"solon-hooks/internal/config"
	"solon-hooks/internal/git"
	"solon-hooks/internal/statusline"
)

// autocompactBuffer matches TS constant: 22.5% of 200k context.
const autocompactBuffer = 45000

var statuslineCmd = &cobra.Command{
	Use:   "statusline",
	Short: "Render Claude Code status line",
	RunE:  runStatusline,
}

func runStatusline(cmd *cobra.Command, args []string) error {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil || len(raw) == 0 {
		printFallback()
		return nil
	}
	if err := renderStatusline(raw); err != nil {
		printFallback()
	}
	return nil
}

// printFallback prints the minimal fallback line when anything goes wrong.
func printFallback() {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}
	fmt.Println("📁 " + statusline.CollapseHome(cwd))
}

// renderStatusline parses stdin JSON and renders the full statusline output.
func renderStatusline(raw []byte) error {
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}

	// Extract working directory
	rawDir := ""
	if ws, ok := data["workspace"].(map[string]interface{}); ok {
		rawDir, _ = ws["current_dir"].(string)
	}
	if rawDir == "" {
		rawDir, _ = data["cwd"].(string)
	}
	if rawDir == "" {
		rawDir, _ = os.Getwd()
	}
	currentDir := statusline.CollapseHome(rawDir)

	// Model display name
	modelName := "Claude"
	if m, ok := data["model"].(map[string]interface{}); ok {
		if v, ok := m["display_name"].(string); ok && v != "" {
			modelName = v
		}
	}

	// Git info (30s cached)
	gitInfo := git.GetGitInfo(rawDir)

	// Context window calculation
	contextPercent := 0
	totalTokens := 0
	contextSize := 0
	if cw, ok := data["context_window"].(map[string]interface{}); ok {
		if v, ok := jsonInt(cw, "context_window_size"); ok {
			contextSize = v
		}
		if usage, ok := cw["current_usage"].(map[string]interface{}); ok && contextSize > autocompactBuffer {
			inp, _ := jsonInt(usage, "input_tokens")
			cacheCreate, _ := jsonInt(usage, "cache_creation_input_tokens")
			cacheRead, _ := jsonInt(usage, "cache_read_input_tokens")
			totalTokens = inp + cacheCreate + cacheRead
			contextPercent = int(math.Min(100, math.Round(
				float64(totalTokens+autocompactBuffer)/float64(contextSize)*100,
			)))
		}
	}

	// Write context data to temp file for other hooks
	sessionID, _ := data["session_id"].(string)
	if sessionID != "" && contextSize > 0 {
		writeContextCache(sessionID, contextPercent, totalTokens, contextSize, data)
	}

	// Parse transcript
	transcriptPath, _ := data["transcript_path"].(string)
	transcript := statusline.ParseTranscript(transcriptPath)

	// Read usage limits cache
	sessionText, usagePercent := readUsageLimitsCache()

	// Lines changed
	linesAdded, linesRemoved := 0, 0
	if cost, ok := data["cost"].(map[string]interface{}); ok {
		linesAdded, _ = jsonInt(cost, "total_lines_added")
		linesRemoved, _ = jsonInt(cost, "total_lines_removed")
	}

	// Build render context
	ctx := &statusline.RenderContext{
		ModelName:      modelName,
		CurrentDir:     currentDir,
		ContextPercent: contextPercent,
		SessionText:    sessionText,
		UsagePercent:   usagePercent,
		LinesAdded:     linesAdded,
		LinesRemoved:   linesRemoved,
		Transcript:     transcript,
	}
	if gitInfo != nil {
		ctx.GitBranch = gitInfo.Branch
		ctx.GitUnstaged = gitInfo.Unstaged
		ctx.GitStaged = gitInfo.Staged
		ctx.GitAhead = gitInfo.Ahead
		ctx.GitBehind = gitInfo.Behind
	}

	// Load config for statusline mode
	cfg := config.LoadConfig(config.LoadConfigOptions{})
	mode := cfg.Statusline
	if mode == "" {
		mode = "full"
	}

	// Render and print
	var lines []string
	switch mode {
	case "none":
		lines = []string{""}
	case "minimal":
		lines = statusline.RenderMinimal(ctx)
	case "compact":
		lines = statusline.RenderCompact(ctx)
	default: // "full"
		lines = statusline.RenderFull(ctx)
	}

	for _, line := range lines {
		fmt.Println(line)
	}
	return nil
}

// jsonInt extracts an int from a JSON-decoded map (float64 is the default numeric type).
func jsonInt(m map[string]interface{}, key string) (int, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	}
	return 0, false
}

// writeContextCache writes context window data to /tmp/sl-context-{sessionId}.json.
func writeContextCache(sessionID string, percent, tokens, size int, data map[string]interface{}) {
	path := filepath.Join(os.TempDir(), "sl-context-"+sessionID+".json")
	usage := map[string]interface{}{}
	if cw, ok := data["context_window"].(map[string]interface{}); ok {
		if u, ok := cw["current_usage"].(map[string]interface{}); ok {
			usage = u
		}
	}
	payload := map[string]interface{}{
		"percent":   percent,
		"tokens":    tokens,
		"size":      size,
		"usage":     usage,
		"timestamp": time.Now().UnixMilli(),
	}
	if b, err := json.Marshal(payload); err == nil {
		_ = os.WriteFile(path, b, 0644)
	}
}

// readUsageLimitsCache reads /tmp/sl-usage-limits-cache.json written by the usage-awareness hook.
// Returns (sessionText, usagePercent) where usagePercent is nil if unavailable.
func readUsageLimitsCache() (string, *int) {
	path := filepath.Join(os.TempDir(), "sl-usage-limits-cache.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", nil
	}
	var cache map[string]interface{}
	if err := json.Unmarshal(raw, &cache); err != nil {
		return "", nil
	}

	if status, _ := cache["status"].(string); status == "unavailable" {
		return "N/A", nil
	}

	cacheData, _ := cache["data"].(map[string]interface{})
	if cacheData == nil {
		return "", nil
	}
	fiveHour, _ := cacheData["five_hour"].(map[string]interface{})
	if fiveHour == nil {
		return "", nil
	}

	var usagePercent *int
	if util, ok := fiveHour["utilization"].(float64); ok {
		v := int(math.Round(util))
		usagePercent = &v
	}

	sessionText := ""
	if resetAt, ok := fiveHour["resets_at"].(string); ok && resetAt != "" {
		remaining := parseResetRemaining(resetAt)
		if remaining > 0 && remaining < 18000 {
			rh := remaining / 3600
			rm := (remaining % 3600) / 60
			sessionText = fmt.Sprintf("%dh %dm until reset", rh, rm)
		}
	}

	return sessionText, usagePercent
}

// parseResetRemaining parses an ISO 8601 timestamp and returns seconds until that time.
// Returns 0 on parse error or if the time is in the past.
func parseResetRemaining(resetAt string) int {
	t, err := time.Parse(time.RFC3339, resetAt)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, resetAt)
		if err != nil {
			return 0
		}
	}
	remaining := int(time.Until(t).Seconds())
	if remaining < 0 {
		return 0
	}
	return remaining
}
