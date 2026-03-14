// renderer.go: Three statusline render modes — full (multi-line), compact (2-line), minimal (1-line).
package statusline

import (
	"fmt"
	"sort"

	"solon-hooks/internal/colors"
)

// RenderContext holds all data needed to render the statusline.
type RenderContext struct {
	ModelName      string
	CurrentDir     string
	GitBranch      string
	GitUnstaged    int
	GitStaged      int
	GitAhead       int
	GitBehind      int
	ContextPercent int
	SessionText    string
	UsagePercent   *int // nil = not available
	LinesAdded     int
	LinesRemoved   int
	Transcript     *TranscriptData
}

// buildUsageString returns the formatted usage/session reset string, or "".
func buildUsageString(ctx *RenderContext) string {
	if ctx.SessionText == "" || ctx.SessionText == "N/A" {
		return ""
	}
	str := replaceStr(ctx.SessionText, " until reset", " left")
	if ctx.UsagePercent != nil {
		str += fmt.Sprintf(" (%d%%)", *ctx.UsagePercent)
	}
	return str
}

// replaceStr is a simple single-occurrence string replacement.
func replaceStr(s, old, new string) string {
	for i := 0; i <= len(s)-len(old); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}

// renderSessionLines builds the top section: model + context bar + usage + dir + branch + stats.
// Responsive wrapping: tries to fit on one line up to 85% of terminal width.
func renderSessionLines(ctx *RenderContext) []string {
	termWidth := GetTerminalWidth()
	threshold := (termWidth * 85) / 100

	dirPart := "📁 " + ctx.CurrentDir

	branchPart := ""
	if ctx.GitBranch != "" {
		branchPart = "🌿 " + ctx.GitBranch
		var indicators []string
		if ctx.GitUnstaged > 0 {
			indicators = append(indicators, fmt.Sprintf("%d", ctx.GitUnstaged))
		}
		if ctx.GitStaged > 0 {
			indicators = append(indicators, fmt.Sprintf("+%d", ctx.GitStaged))
		}
		if ctx.GitAhead > 0 {
			indicators = append(indicators, fmt.Sprintf("%d↑", ctx.GitAhead))
		}
		if ctx.GitBehind > 0 {
			indicators = append(indicators, fmt.Sprintf("%d↓", ctx.GitBehind))
		}
		if len(indicators) > 0 {
			branchPart += " " + colors.YellowStr("("+joinStrings(indicators, ", ")+")")
		}
	}

	locationPart := dirPart
	if branchPart != "" {
		locationPart = dirPart + "  " + branchPart
	}

	sessionPart := "🤖 " + ctx.ModelName
	if ctx.ContextPercent > 0 {
		sessionPart += "  " + colors.ColoredBar(ctx.ContextPercent, 12) + fmt.Sprintf(" %d%%", ctx.ContextPercent)
	}
	if usageStr := buildUsageString(ctx); usageStr != "" {
		sessionPart += "  ⌛ " + replaceStr(usageStr, ")", " used)")
	}

	statsPart := ""
	if ctx.LinesAdded > 0 || ctx.LinesRemoved > 0 {
		statsPart = fmt.Sprintf("📝 %s %s",
			colors.GreenStr(fmt.Sprintf("+%d", ctx.LinesAdded)),
			colors.RedStr(fmt.Sprintf("-%d", ctx.LinesRemoved)))
	}

	statsLen := VisibleLength(statsPart)

	allOneLine := sessionPart + "  " + locationPart + "  " + statsPart
	sessionLocation := sessionPart + "  " + locationPart

	var lines []string
	switch {
	case VisibleLength(allOneLine) <= threshold && statsLen > 0:
		lines = append(lines, allOneLine)
	case VisibleLength(sessionLocation) <= threshold:
		lines = append(lines, sessionLocation)
		if statsLen > 0 {
			lines = append(lines, statsPart)
		}
	case VisibleLength(sessionPart) <= threshold:
		lines = append(lines, sessionPart)
		lines = append(lines, locationPart)
		if statsLen > 0 {
			lines = append(lines, statsPart)
		}
	default:
		lines = append(lines, sessionPart)
		lines = append(lines, dirPart)
		if branchPart != "" {
			lines = append(lines, branchPart)
		}
		if statsLen > 0 {
			lines = append(lines, statsPart)
		}
	}
	return lines
}

// agentGroup is a run of consecutive agents with same type+status for display collapsing.
type agentGroup struct {
	agentType string
	status    string
	count     int
	agents    []TranscriptAgent
}

// renderAgentsLines renders a compact chronological agent flow with duplicate collapsing.
func renderAgentsLines(transcript *TranscriptData) []string {
	if transcript == nil || len(transcript.Agents) == 0 {
		return nil
	}

	// Sort chronologically
	sorted := make([]TranscriptAgent, len(transcript.Agents))
	copy(sorted, transcript.Agents)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartTime.Before(sorted[j].StartTime)
	})

	// Separate running vs completed for display order (running first)
	var running, completed []TranscriptAgent
	for _, a := range sorted {
		if a.Status == "running" {
			running = append(running, a)
		} else {
			completed = append(completed, a)
		}
	}
	allAgents := append(running, completed...)
	if len(allAgents) == 0 {
		return nil
	}

	// Resort combined list by start time
	sort.Slice(allAgents, func(i, j int) bool {
		return allAgents[i].StartTime.Before(allAgents[j].StartTime)
	})

	// Collapse consecutive duplicates
	var collapsed []agentGroup
	for _, a := range allAgents {
		aType := a.Type
		if aType == "" {
			aType = "agent"
		}
		if len(collapsed) > 0 {
			last := &collapsed[len(collapsed)-1]
			if last.agentType == aType && last.status == a.Status {
				last.count++
				last.agents = append(last.agents, a)
				continue
			}
		}
		collapsed = append(collapsed, agentGroup{
			agentType: aType, status: a.Status, count: 1, agents: []TranscriptAgent{a},
		})
	}

	// Show last 4 groups
	toShow := collapsed
	if len(toShow) > 4 {
		toShow = toShow[len(toShow)-4:]
	}

	flowParts := make([]string, 0, len(toShow))
	for _, g := range toShow {
		icon := colors.DimStr("○")
		if g.status == "running" {
			icon = colors.YellowStr("●")
		}
		suffix := ""
		if g.count > 1 {
			suffix = fmt.Sprintf(" ×%d", g.count)
		}
		flowParts = append(flowParts, fmt.Sprintf("%s %s%s", icon, g.agentType, suffix))
	}

	completedCount := len(completed)
	flowSuffix := ""
	if completedCount > 2 {
		flowSuffix = " " + colors.DimStr(fmt.Sprintf("(%d done)", completedCount))
	}

	var lines []string
	lines = append(lines, joinStrings(flowParts, " → ")+flowSuffix)

	// Detail line for running agent (or last completed)
	var detailAgent *TranscriptAgent
	if len(running) > 0 {
		detailAgent = &running[0]
	} else if len(completed) > 0 {
		detailAgent = &completed[len(completed)-1]
	}
	if detailAgent != nil && detailAgent.Description != "" {
		desc := detailAgent.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		elapsed := FormatElapsed(detailAgent.StartTime, detailAgent.EndTime)
		icon := colors.DimStr("▸")
		if detailAgent.Status == "running" {
			icon = colors.YellowStr("▸")
		}
		lines = append(lines, fmt.Sprintf("   %s %s %s", icon, desc, colors.DimStr("("+elapsed+")")))
	}

	return lines
}

// renderTodosLine returns a single line summarising the current todo state, or "".
func renderTodosLine(transcript *TranscriptData) string {
	if transcript == nil || len(transcript.Todos) == 0 {
		return ""
	}
	todos := transcript.Todos
	total := len(todos)

	var inProgress *TranscriptTodo
	completedCount, pendingCount := 0, 0
	for i := range todos {
		switch todos[i].Status {
		case "in_progress":
			if inProgress == nil {
				inProgress = &todos[i]
			}
		case "completed":
			completedCount++
		case "pending":
			pendingCount++
		}
	}

	if inProgress == nil {
		if completedCount == total && total > 0 {
			return colors.GreenStr("✓") + fmt.Sprintf(" All %d todos complete", total)
		}
		if pendingCount > 0 {
			var nextPending *TranscriptTodo
			for i := range todos {
				if todos[i].Status == "pending" {
					nextPending = &todos[i]
					break
				}
			}
			nextTask := "Next task"
			if nextPending != nil {
				nextTask = nextPending.Content
			}
			if len(nextTask) > 40 {
				nextTask = nextTask[:37] + "..."
			}
			return fmt.Sprintf("%s Next: %s %s",
				colors.DimStr("○"),
				nextTask,
				colors.DimStr(fmt.Sprintf("(%d done, %d pending)", completedCount, pendingCount)))
		}
		return ""
	}

	displayText := inProgress.ActiveForm
	if displayText == "" {
		displayText = inProgress.Content
	}
	if len(displayText) > 50 {
		displayText = displayText[:47] + "..."
	}
	return fmt.Sprintf("%s %s %s",
		colors.YellowStr("▸"),
		displayText,
		colors.DimStr(fmt.Sprintf("(%d done, %d pending)", completedCount, pendingCount)))
}

// RenderFull renders the full multi-line statusline (session + agents + todos).
func RenderFull(ctx *RenderContext) []string {
	var lines []string
	lines = append(lines, renderSessionLines(ctx)...)
	lines = append(lines, renderAgentsLines(ctx.Transcript)...)
	if todo := renderTodosLine(ctx.Transcript); todo != "" {
		lines = append(lines, todo)
	}
	return lines
}

// RenderCompact renders a 2-line statusline: model/context on line 1, dir/branch on line 2.
func RenderCompact(ctx *RenderContext) []string {
	line1 := "🤖 " + ctx.ModelName
	if ctx.ContextPercent > 0 {
		line1 += "  " + colors.ColoredBar(ctx.ContextPercent, 12) + fmt.Sprintf(" %d%%", ctx.ContextPercent)
	}
	if usageStr := buildUsageString(ctx); usageStr != "" {
		line1 += "  ⌛ " + usageStr
	}

	line2 := "📁 " + ctx.CurrentDir
	if ctx.GitBranch != "" {
		line2 += "  🌿 " + ctx.GitBranch
	}

	return []string{line1, line2}
}

// RenderMinimal renders a single emoji-separated statusline.
func RenderMinimal(ctx *RenderContext) []string {
	parts := []string{"🤖 " + ctx.ModelName}
	if ctx.ContextPercent > 0 {
		batteryIcon := "🔋"
		if ctx.ContextPercent > 70 {
			batteryIcon = colors.RedStr("🔋")
		}
		parts = append(parts, fmt.Sprintf("%s %d%%", batteryIcon, ctx.ContextPercent))
	}
	if usageStr := buildUsageString(ctx); usageStr != "" {
		parts = append(parts, "⏰ "+usageStr)
	}
	if ctx.GitBranch != "" {
		parts = append(parts, "🌿 "+ctx.GitBranch)
	}
	parts = append(parts, "📁 "+ctx.CurrentDir)
	return []string{joinStrings(parts, "  ")}
}

// joinStrings joins a slice of strings with sep.
func joinStrings(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}
