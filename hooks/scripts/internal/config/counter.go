// Config counter: count CLAUDE.md files, rules, MCP servers, and hooks
// across user (~/.claude/) and project (.claude/) scopes.
// All fs errors fail silently.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"solon-hooks/internal/hookio"
)

// getMcpServerNames extracts MCP server names from a settings JSON file.
func getMcpServerNames(filePath string) map[string]struct{} {
	result := make(map[string]struct{})
	data, err := os.ReadFile(filePath)
	if err != nil {
		return result
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return result
	}
	servers, ok := cfg["mcpServers"].(map[string]interface{})
	if !ok {
		return result
	}
	for name := range servers {
		result[name] = struct{}{}
	}
	return result
}

// CountMcpServersInFile counts MCP servers in filePath, excluding those in excludeFrom.
func CountMcpServersInFile(filePath string, excludeFrom string) int {
	servers := getMcpServerNames(filePath)
	if excludeFrom != "" {
		exclude := getMcpServerNames(excludeFrom)
		for name := range exclude {
			delete(servers, name)
		}
	}
	return len(servers)
}

// CountHooksInFile counts hook event types in a settings JSON file.
func CountHooksInFile(filePath string) int {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return 0
	}
	hooks, ok := cfg["hooks"].(map[string]interface{})
	if !ok {
		return 0
	}
	return len(hooks)
}

// CountRulesInDir counts .md files recursively up to depth 5, skipping symlinks.
func CountRulesInDir(rulesDir string, depth int) int {
	if depth > 5 {
		return 0
	}
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		// Skip symlinks
		info, err := entry.Info()
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		fullPath := filepath.Join(rulesDir, entry.Name())
		if entry.IsDir() {
			count += CountRulesInDir(fullPath, depth+1)
		} else if entry.Type().IsRegular() && filepath.Ext(entry.Name()) == ".md" {
			count++
		}
	}
	return count
}

// CountConfigs counts all configuration artifacts across user and project scopes.
func CountConfigs(cwd string) hookio.ConfigCounts {
	var counts hookio.ConfigCounts

	home, _ := os.UserHomeDir()
	claudeDir := filepath.Join(home, ".claude")

	// User scope
	if fileExists(filepath.Join(claudeDir, "CLAUDE.md")) {
		counts.ClaudeMdCount++
	}
	counts.RulesCount += CountRulesInDir(filepath.Join(claudeDir, "rules"), 0)
	userSettings := filepath.Join(claudeDir, "settings.json")
	counts.McpCount += CountMcpServersInFile(userSettings, "")
	counts.HooksCount += CountHooksInFile(userSettings)
	counts.McpCount += CountMcpServersInFile(filepath.Join(home, ".claude.json"), userSettings)

	// Project scope
	if cwd != "" {
		for _, name := range []string{"CLAUDE.md", "CLAUDE.local.md"} {
			if fileExists(filepath.Join(cwd, name)) {
				counts.ClaudeMdCount++
			}
		}
		for _, name := range []string{"CLAUDE.md", "CLAUDE.local.md"} {
			if fileExists(filepath.Join(cwd, ".claude", name)) {
				counts.ClaudeMdCount++
			}
		}
		counts.RulesCount += CountRulesInDir(filepath.Join(cwd, ".claude", "rules"), 0)
		counts.McpCount += CountMcpServersInFile(filepath.Join(cwd, ".mcp.json"), "")
		projectSettings := filepath.Join(cwd, ".claude", "settings.json")
		counts.McpCount += CountMcpServersInFile(projectSettings, "")
		counts.HooksCount += CountHooksInFile(projectSettings)
		localSettings := filepath.Join(cwd, ".claude", "settings.local.json")
		counts.McpCount += CountMcpServersInFile(localSettings, "")
		counts.HooksCount += CountHooksInFile(localSettings)
	}

	return counts
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
