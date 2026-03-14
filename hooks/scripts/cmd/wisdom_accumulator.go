package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"solon-hooks/internal/config"
	"solon-hooks/internal/hookio"
)

const maxWisdomLines = 50

var wisdomAccumulatorCmd = &cobra.Command{
	Use:   "wisdom-accumulator",
	Short: "Extract and store learnings from completed subagents",
	RunE:  runWisdomAccumulator,
}

func runWisdomAccumulator(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("wisdom-accumulation") {
		os.Exit(0)
	}

	defer func() {
		if r := recover(); r != nil {
			os.Exit(0)
		}
	}()

	var input hookio.SubagentStopInput
	_ = hookio.ReadInput(&input)

	planPath := os.Getenv("SL_ACTIVE_PLAN")
	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = os.Getenv("SL_SESSION_ID")
	}

	// Resolve wisdom file path
	var wisdomPath string
	if planPath != "" {
		wisdomPath = filepath.Join(planPath, ".wisdom.md")
	} else if sessionID != "" {
		wisdomPath = filepath.Join(os.TempDir(), "sl-wisdom-"+sessionID+".md")
	} else {
		os.Exit(0)
	}

	// Extract learnings from transcript tail
	learnings := extractLearnings(input.TranscriptPath, input.AgentType)
	if learnings == "" {
		os.Exit(0)
	}

	// Append to wisdom file and prune if needed
	appendWisdom(wisdomPath, learnings)
	pruneWisdom(wisdomPath, maxWisdomLines)

	return nil
}

// extractLearnings reads last 30 lines of transcript and extracts actionable items.
func extractLearnings(transcriptPath, agentType string) string {
	if transcriptPath == "" {
		return ""
	}

	f, err := os.Open(transcriptPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	// Read all lines, keep last 30
	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) > 30 {
		lines = lines[len(lines)-30:]
	}

	// Extract lines containing actionable keywords
	keywords := []string{
		"learned", "gotcha", "important", "note:", "warning:",
		"convention", "pattern", "decision", "discovered",
		"must", "should", "avoid", "don't", "careful",
		"works", "doesn't work", "failed", "succeeded",
	}

	var learnings []string
	for _, line := range lines {
		lower := strings.ToLower(line)
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				trimmed := strings.TrimSpace(line)
				if len(trimmed) > 10 && len(trimmed) < 200 {
					learnings = append(learnings, trimmed)
				}
				break
			}
		}
	}

	if len(learnings) == 0 {
		return ""
	}

	// Cap at 5 learnings per subagent
	if len(learnings) > 5 {
		learnings = learnings[len(learnings)-5:]
	}

	header := fmt.Sprintf("### %s (%s)", agentType, time.Now().Format("15:04"))
	return header + "\n" + strings.Join(learnings, "\n")
}

// appendWisdom appends content to the wisdom file.
func appendWisdom(path, content string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "\n%s\n", content)
}

// pruneWisdom keeps only the last maxLines of the wisdom file.
func pruneWisdom(path string, maxLines int) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) <= maxLines {
		return
	}

	pruned := strings.Join(lines[len(lines)-maxLines:], "\n")
	os.WriteFile(path, []byte(pruned), 0644)
}
