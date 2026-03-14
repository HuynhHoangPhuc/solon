// slack.go: Slack Block Kit notification provider via incoming webhooks.
package notify

import (
	"path/filepath"
	"time"
)

// Slack is the Slack incoming webhook provider.
var Slack = Provider{
	Name: "slack",
	IsEnabled: func(env map[string]string) bool {
		return env["SLACK_WEBHOOK_URL"] != ""
	},
	Send: func(input map[string]interface{}, env map[string]string) (*SendResult, error) {
		webhookURL := env["SLACK_WEBHOOK_URL"]
		hookType := strField(input, "hook_event_name", "unknown")
		cwd := strField(input, "cwd", "")
		sessionID := strField(input, "session_id", "")

		projectName := "Unknown"
		if cwd != "" {
			if b := filepath.Base(cwd); b != "" {
				projectName = b
			}
		}
		sessionShort := sessionID
		if len(sessionShort) > 8 {
			sessionShort = sessionShort[:8]
		}

		payload := map[string]interface{}{
			"text":   "Claude Code: " + hookType + " in " + projectName,
			"blocks": buildSlackBlocks(input, hookType, projectName, sessionShort),
		}
		return Send("slack", webhookURL, payload, nil), nil
	},
}

// slackTitle returns the notification title for a hook type.
func slackTitle(hookType string) string {
	switch hookType {
	case "Stop":
		return "Claude Code Session Complete"
	case "SubagentStop":
		return "Claude Code Subagent Complete"
	case "AskUserPrompt":
		return "Claude Code Needs Input"
	default:
		return "Claude Code Event"
	}
}

// buildSlackBlocks constructs Block Kit blocks for the notification payload.
func buildSlackBlocks(input map[string]interface{}, hookType, projectName, sessionID string) []interface{} {
	ts := time.Now().Format("2006-01-02 15:04:05")
	cwd := strField(input, "cwd", "Unknown")

	blocks := []interface{}{
		map[string]interface{}{
			"type": "header",
			"text": map[string]interface{}{
				"type": "plain_text",
				"text": slackTitle(hookType),
			},
		},
		map[string]interface{}{
			"type": "section",
			"fields": []interface{}{
				map[string]interface{}{"type": "mrkdwn", "text": "*Project:*\n" + projectName},
				map[string]interface{}{"type": "mrkdwn", "text": "*Time:*\n" + ts},
				map[string]interface{}{"type": "mrkdwn", "text": "*Session:*\n`" + sessionID + "...`"},
				map[string]interface{}{"type": "mrkdwn", "text": "*Event:*\n" + hookType},
			},
		},
		map[string]interface{}{"type": "divider"},
		map[string]interface{}{
			"type": "context",
			"elements": []interface{}{
				map[string]interface{}{"type": "mrkdwn", "text": "📍 `" + cwd + "`"},
			},
		},
	}

	// Insert agent type block for SubagentStop
	if hookType == "SubagentStop" {
		agentType := strField(input, "agent_type", "unknown")
		agentBlock := map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": "*Agent Type:* " + agentType,
			},
		}
		// Insert before divider (index 2)
		newBlocks := make([]interface{}, 0, len(blocks)+1)
		newBlocks = append(newBlocks, blocks[:2]...)
		newBlocks = append(newBlocks, agentBlock)
		newBlocks = append(newBlocks, blocks[2:]...)
		blocks = newBlocks
	}

	return blocks
}
