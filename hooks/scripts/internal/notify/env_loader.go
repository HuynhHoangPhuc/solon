// Package notify handles notification dispatch to external services.
// env_loader.go: .env file parser with cascade loading.
package notify

import (
	"os"
	"path/filepath"
	"strings"
)

// ParseEnvContent parses .env file content into a key-value map.
// Supports comments (#), quoted values (single/double), inline comments for unquoted values.
func ParseEnvContent(content string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		eqIdx := strings.Index(trimmed, "=")
		if eqIdx == -1 {
			continue
		}

		key := strings.TrimSpace(trimmed[:eqIdx])
		value := strings.TrimSpace(trimmed[eqIdx+1:])

		// Strip surrounding quotes
		if len(value) >= 2 &&
			((value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		} else {
			// Strip inline comments for unquoted values
			if cidx := strings.Index(value, "#"); cidx != -1 {
				value = strings.TrimSpace(value[:cidx])
			}
		}

		if key != "" {
			result[key] = value
		}
	}
	return result
}

// loadEnvFile reads and parses a .env file; returns empty map on error.
func loadEnvFile(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]string)
	}
	parsed := ParseEnvContent(string(data))
	if len(parsed) > 0 {
		os.Stderr.WriteString("[env-loader] Loaded: " + path + "\n")
	}
	return parsed
}

// LoadEnv loads environment with cascade: .claude/.env (low) → ~/.claude/.env → os.Environ (high).
// cwd is used to find the project-local .claude/.env file.
func LoadEnv(cwd string) map[string]string {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			cwd = "."
		}
	}

	home, _ := os.UserHomeDir()
	envFiles := []string{
		filepath.Join(cwd, ".claude", ".env"),
		filepath.Join(home, ".claude", ".env"),
	}

	merged := make(map[string]string)
	for _, f := range envFiles {
		for k, v := range loadEnvFile(f) {
			merged[k] = v
		}
	}

	// os.Environ overrides file values
	for _, kv := range os.Environ() {
		idx := strings.Index(kv, "=")
		if idx == -1 {
			continue
		}
		merged[kv[:idx]] = kv[idx+1:]
	}

	return merged
}
