// discord.go: Discord webhook notification provider using rich embeds.
package notify

import (
	"path/filepath"
	"time"
)

var discordColors = map[string]int{
	"Stop":          5763719,  // Green
	"SubagentStop":  3447003,  // Blue
	"AskUserPrompt": 15844367, // Yellow
	"default":       10070709, // Gray
}

// Discord is the Discord webhook provider.
var Discord = Provider{
	Name: "discord",
	IsEnabled: func(env map[string]string) bool {
		return env["DISCORD_WEBHOOK_URL"] != ""
	},
	Send: func(input map[string]interface{}, env map[string]string) (*SendResult, error) {
		webhookURL := env["DISCORD_WEBHOOK_URL"]
		if webhookURL == "" {
			return &SendResult{Error: "DISCORD_WEBHOOK_URL not configured"}, nil
		}
		embed := buildDiscordEmbed(input)
		body := map[string]interface{}{
			"embeds": []interface{}{embed},
		}
		return Send("discord", webhookURL, body, nil), nil
	},
}

// discordField is a Discord embed field.
type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// discordEmbed is a Discord message embed.
type discordEmbed struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Color       int            `json:"color"`
	Timestamp   string         `json:"timestamp"`
	Footer      map[string]string `json:"footer"`
	Fields      []discordField `json:"fields"`
}

// buildDiscordEmbed constructs the embed payload for a hook input.
func buildDiscordEmbed(input map[string]interface{}) discordEmbed {
	hookType := strField(input, "hook_event_name", "unknown")
	cwd := strField(input, "cwd", "")
	sessionID := strField(input, "session_id", "")

	projectName := "Unknown"
	if cwd != "" {
		if b := filepath.Base(cwd); b != "" {
			projectName = b
		}
	}

	color, ok := discordColors[hookType]
	if !ok {
		color = discordColors["default"]
	}

	sessionDisplay := "`N/A`"
	if len(sessionID) >= 8 {
		sessionDisplay = "`" + sessionID[:8] + "...`"
	} else if sessionID != "" {
		sessionDisplay = "`" + sessionID + "`"
	}

	ts := time.Now().Format("15:04:05")
	isoTS := time.Now().UTC().Format(time.RFC3339)
	cwdDisplay := cwd
	if cwdDisplay == "" {
		cwdDisplay = "Unknown"
	}
	locationField := discordField{Name: "📍 Location", Value: "`" + cwdDisplay + "`", Inline: false}

	var title, description string
	var fields []discordField

	switch hookType {
	case "Stop":
		title = "Claude Code Session Complete"
		description = "Session completed successfully"
		fields = []discordField{
			{Name: "⏰ Time", Value: ts, Inline: true},
			{Name: "🆔 Session", Value: sessionDisplay, Inline: true},
			locationField,
		}
	case "SubagentStop":
		agentType := strField(input, "agent_type", "unknown")
		title = "Claude Code Subagent Complete"
		description = "Specialized agent completed its task"
		fields = []discordField{
			{Name: "⏰ Time", Value: ts, Inline: true},
			{Name: "🔧 Agent Type", Value: agentType, Inline: true},
			{Name: "🆔 Session", Value: sessionDisplay, Inline: true},
			locationField,
		}
	case "AskUserPrompt":
		title = "Claude Code Needs Input"
		description = "Claude is waiting for user input"
		fields = []discordField{
			{Name: "⏰ Time", Value: ts, Inline: true},
			{Name: "🆔 Session", Value: sessionDisplay, Inline: true},
			locationField,
		}
	default:
		title = "Claude Code Event"
		description = "Claude Code event triggered"
		fields = []discordField{
			{Name: "⏰ Time", Value: ts, Inline: true},
			{Name: "📋 Event", Value: hookType, Inline: true},
			{Name: "🆔 Session", Value: sessionDisplay, Inline: true},
			locationField,
		}
	}

	return discordEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Timestamp:   isoTS,
		Footer:      map[string]string{"text": "Project • " + projectName},
		Fields:      fields,
	}
}
