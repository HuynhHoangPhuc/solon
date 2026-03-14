// telegram.go: Telegram Bot API notification provider.
package notify

import (
	"fmt"
	"path/filepath"
	"time"
)

// Telegram is the Telegram Bot API provider.
var Telegram = Provider{
	Name: "telegram",
	IsEnabled: func(env map[string]string) bool {
		return env["TELEGRAM_BOT_TOKEN"] != "" && env["TELEGRAM_CHAT_ID"] != ""
	},
	Send: func(input map[string]interface{}, env map[string]string) (*SendResult, error) {
		token := env["TELEGRAM_BOT_TOKEN"]
		chatID := env["TELEGRAM_CHAT_ID"]
		url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
		msg := formatTelegramMessage(input)
		body := map[string]interface{}{
			"chat_id":                  chatID,
			"text":                     msg,
			"parse_mode":               "Markdown",
			"disable_web_page_preview": true,
		}
		return Send("telegram", url, body, nil), nil
	},
}

// getTimestamp returns a formatted timestamp string (YYYY-MM-DD HH:MM:SS).
func getTimestamp() string {
	now := time.Now()
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second())
}

// formatTelegramMessage builds the Markdown message text for a given hook input.
func formatTelegramMessage(input map[string]interface{}) string {
	hookType := strField(input, "hook_event_name", "unknown")
	cwd := strField(input, "cwd", "")
	sessionID := strField(input, "session_id", "")
	projectName := "unknown"
	if cwd != "" {
		projectName = filepath.Base(cwd)
	}
	ts := getTimestamp()
	sessionDisplay := "N/A"
	if len(sessionID) >= 8 {
		sessionDisplay = sessionID[:8] + "..."
	} else if sessionID != "" {
		sessionDisplay = sessionID
	}

	switch hookType {
	case "Stop":
		return fmt.Sprintf(
			"🚀 *Project Task Completed*\n\n📅 *Time:* %s\n📁 *Project:* %s\n🆔 *Session:* %s\n\n📍 *Location:* `%s`",
			ts, projectName, sessionDisplay, cwd)
	case "SubagentStop":
		agentType := strField(input, "agent_type", "unknown")
		return fmt.Sprintf(
			"🤖 *Project Subagent Completed*\n\n📅 *Time:* %s\n📁 *Project:* %s\n🔧 *Agent Type:* %s\n🆔 *Session:* %s\n\nSpecialized agent completed its task.\n\n📍 *Location:* `%s`",
			ts, projectName, agentType, sessionDisplay, cwd)
	case "AskUserPrompt":
		return fmt.Sprintf(
			"💬 *User Input Needed*\n\n📅 *Time:* %s\n📁 *Project:* %s\n🆔 *Session:* %s\n\nClaude is waiting for your input.\n\n📍 *Location:* `%s`",
			ts, projectName, sessionDisplay, cwd)
	default:
		return fmt.Sprintf(
			"📝 *Project Code Event*\n\n📅 *Time:* %s\n📁 *Project:* %s\n📋 *Event:* %s\n🆔 *Session:* %s\n\n📍 *Location:* `%s`",
			ts, projectName, hookType, sessionDisplay, cwd)
	}
}
