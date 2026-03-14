package notify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSendSuccessWithMockServer verifies Send posts JSON and returns Success=true on 200.
func TestSendSuccessWithMockServer(t *testing.T) {
	var received []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Clear any throttle state for this provider name
	clearThrottle("test-provider")

	result := Send("test-provider", srv.URL, map[string]string{"key": "value"}, nil)
	if !result.Success {
		t.Errorf("expected Success=true, got Error=%q", result.Error)
	}
	if len(received) == 0 {
		t.Errorf("server should have received a JSON body")
	}
}

// TestSendFailsOn4xxAndRecordsError verifies Send returns an error on non-2xx responses.
func TestSendFailsOn4xxAndRecordsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer srv.Close()

	clearThrottle("test-4xx")
	result := Send("test-4xx", srv.URL, map[string]string{"x": "y"}, nil)
	if result.Success {
		t.Errorf("expected failure on 400, got Success=true")
	}
	if !strings.Contains(result.Error, "400") {
		t.Errorf("error should mention HTTP status, got %q", result.Error)
	}
}

// TestSendWithCustomHeaders verifies custom headers are forwarded to the server.
func TestSendWithCustomHeaders(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	clearThrottle("test-headers")
	Send("test-headers", srv.URL, map[string]string{}, map[string]string{
		"Authorization": "Bearer mytoken",
	})
	if authHeader != "Bearer mytoken" {
		t.Errorf("expected Authorization header forwarded, got %q", authHeader)
	}
}

// TestSendEmptyURLReturnsError verifies Send with empty/invalid URL returns an error gracefully.
func TestSendEmptyURLReturnsError(t *testing.T) {
	clearThrottle("test-badurl")
	result := Send("test-badurl", "http://127.0.0.1:0/unreachable", map[string]string{}, nil)
	if result.Success {
		t.Errorf("unreachable URL should not return Success=true")
	}
}

// TestDiscordIsEnabledWhenURLSet verifies Discord.IsEnabled returns true with webhook URL.
func TestDiscordIsEnabledWhenURLSet(t *testing.T) {
	env := map[string]string{"DISCORD_WEBHOOK_URL": "https://discord.com/api/webhooks/test"}
	if !Discord.IsEnabled(env) {
		t.Errorf("Discord should be enabled when DISCORD_WEBHOOK_URL is set")
	}
}

// TestDiscordIsDisabledWhenURLEmpty verifies Discord.IsEnabled returns false with no URL.
func TestDiscordIsDisabledWhenURLEmpty(t *testing.T) {
	env := map[string]string{}
	if Discord.IsEnabled(env) {
		t.Errorf("Discord should be disabled when DISCORD_WEBHOOK_URL is absent")
	}
}

// TestDiscordSendPostsEmbedPayload verifies Discord.Send sends an embeds array.
func TestDiscordSendPostsEmbedPayload(t *testing.T) {
	var body map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	clearThrottle("discord")
	env := map[string]string{"DISCORD_WEBHOOK_URL": srv.URL}
	input := map[string]interface{}{
		"hook_event_name": "Stop",
		"session_id":      "abc123xyz",
		"cwd":             "/home/user/myproject",
	}
	result, err := Discord.Send(input, env)
	if err != nil {
		t.Fatalf("Discord.Send returned error: %v", err)
	}
	if !result.Success {
		t.Errorf("Discord.Send expected Success=true, got Error=%q", result.Error)
	}
	if _, ok := body["embeds"]; !ok {
		t.Errorf("Discord payload should contain 'embeds' key, got: %v", body)
	}
}

// TestSlackIsEnabledWhenURLSet verifies Slack.IsEnabled returns true with webhook URL.
func TestSlackIsEnabledWhenURLSet(t *testing.T) {
	env := map[string]string{"SLACK_WEBHOOK_URL": "https://hooks.slack.com/services/test"}
	if !Slack.IsEnabled(env) {
		t.Errorf("Slack should be enabled when SLACK_WEBHOOK_URL is set")
	}
}

// TestSlackSendPostsBlocksPayload verifies Slack.Send includes Block Kit blocks.
func TestSlackSendPostsBlocksPayload(t *testing.T) {
	var body map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	clearThrottle("slack")
	env := map[string]string{"SLACK_WEBHOOK_URL": srv.URL}
	input := map[string]interface{}{
		"hook_event_name": "Stop",
		"session_id":      "sess123",
		"cwd":             "/projects/solon",
	}
	result, err := Slack.Send(input, env)
	if err != nil {
		t.Fatalf("Slack.Send returned error: %v", err)
	}
	if !result.Success {
		t.Errorf("Slack.Send expected Success=true, got Error=%q", result.Error)
	}
	if _, ok := body["blocks"]; !ok {
		t.Errorf("Slack payload should contain 'blocks' key, got: %v", body)
	}
}

// TestTelegramIsEnabledWhenBothTokenAndChatIDSet verifies Telegram requires both env vars.
func TestTelegramIsEnabledWhenBothTokenAndChatIDSet(t *testing.T) {
	full := map[string]string{
		"TELEGRAM_BOT_TOKEN": "123:abc",
		"TELEGRAM_CHAT_ID":   "-1001234",
	}
	if !Telegram.IsEnabled(full) {
		t.Errorf("Telegram should be enabled when both token and chat_id are set")
	}
	// Missing chat_id
	partial := map[string]string{"TELEGRAM_BOT_TOKEN": "123:abc"}
	if Telegram.IsEnabled(partial) {
		t.Errorf("Telegram should be disabled when TELEGRAM_CHAT_ID is absent")
	}
}

// TestParseEnvContentBasic verifies ParseEnvContent parses key=value pairs.
func TestParseEnvContentBasic(t *testing.T) {
	content := "FOO=bar\nBAZ=qux\n"
	result := ParseEnvContent(content)
	if result["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got %q", result["FOO"])
	}
	if result["BAZ"] != "qux" {
		t.Errorf("expected BAZ=qux, got %q", result["BAZ"])
	}
}

// TestParseEnvContentSkipsComments verifies lines starting with # are ignored.
func TestParseEnvContentSkipsComments(t *testing.T) {
	content := "# this is a comment\nKEY=value\n"
	result := ParseEnvContent(content)
	if _, ok := result["# this is a comment"]; ok {
		t.Errorf("comment lines should not be parsed as keys")
	}
	if result["KEY"] != "value" {
		t.Errorf("expected KEY=value, got %q", result["KEY"])
	}
}

// TestParseEnvContentStripsQuotes verifies quoted values have quotes stripped.
func TestParseEnvContentStripsQuotes(t *testing.T) {
	content := `QUOTED="hello world"` + "\n" + `SINGLE='another'` + "\n"
	result := ParseEnvContent(content)
	if result["QUOTED"] != "hello world" {
		t.Errorf("expected unquoted value, got %q", result["QUOTED"])
	}
	if result["SINGLE"] != "another" {
		t.Errorf("expected unquoted single-quoted value, got %q", result["SINGLE"])
	}
}

// TestStrFieldFallback verifies strField returns fallback for missing or non-string keys.
func TestStrFieldFallback(t *testing.T) {
	m := map[string]interface{}{"name": "alice"}
	if strField(m, "name", "default") != "alice" {
		t.Errorf("strField should return existing string value")
	}
	if strField(m, "missing", "fallback") != "fallback" {
		t.Errorf("strField should return fallback for missing key")
	}
}

// TestBuildDiscordEmbedStopEvent verifies buildDiscordEmbed produces correct title for Stop.
func TestBuildDiscordEmbedStopEvent(t *testing.T) {
	input := map[string]interface{}{
		"hook_event_name": "Stop",
		"session_id":      "abcdef1234",
		"cwd":             "/projects/solon",
	}
	embed := buildDiscordEmbed(input)
	if !strings.Contains(embed.Title, "Session Complete") {
		t.Errorf("Stop embed title should mention Session Complete, got %q", embed.Title)
	}
	if embed.Color != discordColors["Stop"] {
		t.Errorf("Stop embed color mismatch: want %d got %d", discordColors["Stop"], embed.Color)
	}
}

// TestFormatTelegramMessageSubagentStop verifies SubagentStop message includes agent type.
func TestFormatTelegramMessageSubagentStop(t *testing.T) {
	input := map[string]interface{}{
		"hook_event_name": "SubagentStop",
		"agent_type":      "tester",
		"session_id":      "sess001",
		"cwd":             "/projects/solon",
	}
	msg := formatTelegramMessage(input)
	if !strings.Contains(msg, "tester") {
		t.Errorf("SubagentStop message should include agent_type, got: %q", msg)
	}
	if !strings.Contains(msg, "Subagent") {
		t.Errorf("SubagentStop message should mention Subagent, got: %q", msg)
	}
}
