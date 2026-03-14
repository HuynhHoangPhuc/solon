package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"solon-hooks/internal/config"

	"github.com/spf13/cobra"
)

var usageAwarenessCmd = &cobra.Command{
	Use:   "usage-awareness",
	Short: "Handle UserPromptSubmit/PostToolUse usage context awareness",
	RunE:  runUsageAwareness,
}

const (
	usageCacheFile        = "sl-usage-limits-cache.json"
	usageCacheTTLPrompt   = 60 * time.Second
	usageCacheTTLDefault  = 300 * time.Second
	usageAPIURL           = "https://api.anthropic.com/api/oauth/usage"
	usageHTTPTimeout      = 10 * time.Second
)

type usageCache struct {
	Timestamp int64       `json:"timestamp"`
	Status    string      `json:"status"`
	Data      interface{} `json:"data,omitempty"`
}

func runUsageAwareness(cmd *cobra.Command, args []string) error {
	if !config.IsHookEnabled("usage-context-awareness") {
		os.Exit(0)
	}

	// Always output continue:true regardless of errors
	defer func() {
		fmt.Fprint(os.Stdout, `{"continue":true}`)
	}()

	var inputData map[string]interface{}
	if err := json.NewDecoder(os.Stdin).Decode(&inputData); err != nil {
		inputData = map[string]interface{}{}
	}

	_, isUserPrompt := inputData["prompt"].(string)

	if usageShouldFetch(isUserPrompt) {
		fetchAndCacheUsage()
	}

	return nil
}

func usageCachePath() string {
	return filepath.Join(os.TempDir(), usageCacheFile)
}

func usageShouldFetch(isUserPrompt bool) bool {
	ttl := usageCacheTTLDefault
	if isUserPrompt {
		ttl = usageCacheTTLPrompt
	}
	data, err := os.ReadFile(usageCachePath())
	if err != nil {
		return true
	}
	var cache usageCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return true
	}
	age := time.Duration(time.Now().UnixMilli()-cache.Timestamp) * time.Millisecond
	return age >= ttl
}

func writeUsageCache(status string, data interface{}) {
	entry := usageCache{
		Timestamp: time.Now().UnixMilli(),
		Status:    status,
		Data:      data,
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_ = os.WriteFile(usageCachePath(), b, 0644)
}

func getClaudeCredentials() string {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("security", "find-generic-password", "-s", "Claude Code-credentials", "-w").Output()
		if err == nil {
			raw := strings.TrimSpace(string(out))
			var parsed struct {
				ClaudeAiOauth struct {
					AccessToken string `json:"accessToken"`
				} `json:"claudeAiOauth"`
			}
			if json.Unmarshal([]byte(raw), &parsed) == nil && parsed.ClaudeAiOauth.AccessToken != "" {
				return parsed.ClaudeAiOauth.AccessToken
			}
		}
	}

	home, _ := os.UserHomeDir()
	credPath := filepath.Join(home, ".claude", ".credentials.json")
	data, err := os.ReadFile(credPath)
	if err != nil {
		return ""
	}
	var creds struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(data, &creds); err != nil {
		return ""
	}
	return creds.ClaudeAiOauth.AccessToken
}

func fetchAndCacheUsage() {
	token := getClaudeCredentials()
	if token == "" {
		writeUsageCache("unavailable", nil)
		return
	}

	client := &http.Client{Timeout: usageHTTPTimeout}
	req, err := http.NewRequest(http.MethodGet, usageAPIURL, nil)
	if err != nil {
		writeUsageCache("unavailable", nil)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("User-Agent", "solon/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		writeUsageCache("unavailable", nil)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		writeUsageCache("unavailable", nil)
		return
	}

	var respData interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		writeUsageCache("unavailable", nil)
		return
	}
	writeUsageCache("available", respData)
}
