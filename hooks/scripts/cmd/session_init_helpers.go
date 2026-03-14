package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"solon-hooks/internal/config"
)

// writeEnvForced writes an env var to envFile even if value is empty string.
// Unlike config.WriteEnv which skips zero/empty values.
func writeEnvForced(envFile, key string, value interface{}) {
	if envFile == "" {
		return
	}
	escaped := config.EscapeShellValue(fmt.Sprintf("%v", value))
	line := fmt.Sprintf("export %s=\"%s\"\n", key, escaped)
	f, err := os.OpenFile(envFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line)
}

// shadowedCleanupResult holds results of orphaned .shadowed/ dir cleanup.
type shadowedCleanupResult struct {
	Restored []string
	Skipped  []string
	Kept     []string
}

// teamInfo holds detected agent team info.
type teamInfo struct {
	TeamName    string
	MemberCount int
}

// cleanupOrphanedShadowedSkills restores skills from an orphaned .shadowed/ dir (Issue #422).
func cleanupOrphanedShadowedSkills() shadowedCleanupResult {
	cwd, _ := os.Getwd()
	shadowedDir := filepath.Join(cwd, ".claude", "skills", ".shadowed")
	skillsDir := filepath.Join(cwd, ".claude", "skills")

	if _, err := os.Stat(shadowedDir); os.IsNotExist(err) {
		return shadowedCleanupResult{}
	}

	var result shadowedCleanupResult
	entries, err := os.ReadDir(shadowedDir)
	if err != nil {
		return result
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		src := filepath.Join(shadowedDir, entry.Name())
		dest := filepath.Join(skillsDir, entry.Name())

		if _, err := os.Stat(dest); os.IsNotExist(err) {
			// Destination doesn't exist — restore
			if err := os.Rename(src, dest); err == nil {
				result.Restored = append(result.Restored, entry.Name())
			} else {
				os.Stderr.WriteString("[session-init] Failed to restore \"" + entry.Name() + "\": " + err.Error() + "\n")
			}
			continue
		}

		// Destination exists — compare SKILL.md
		orphanSkill := filepath.Join(src, "SKILL.md")
		localSkill := filepath.Join(dest, "SKILL.md")
		orphanData, oErr := os.ReadFile(orphanSkill)
		localData, lErr := os.ReadFile(localSkill)

		if oErr == nil && lErr == nil && string(orphanData) == string(localData) {
			// Identical — safe to remove
			os.RemoveAll(src)
			result.Skipped = append(result.Skipped, entry.Name())
		} else if oErr != nil || lErr != nil {
			// Missing SKILL.md — remove
			os.RemoveAll(src)
			result.Skipped = append(result.Skipped, entry.Name())
		} else {
			// Content differs — keep for manual review
			result.Kept = append(result.Kept, entry.Name())
		}
	}

	// Clean up manifest and dir if empty
	manifestFile := filepath.Join(shadowedDir, ".dedup-manifest.json")
	os.Remove(manifestFile)
	remaining, _ := os.ReadDir(shadowedDir)
	if len(remaining) == 0 {
		os.Remove(shadowedDir)
	}

	return result
}

// detectAgentTeam scans ~/.claude/teams/ for an active team config.
func detectAgentTeam() *teamInfo {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	teamsDir := filepath.Join(home, ".claude", "teams")
	entries, err := os.ReadDir(teamsDir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		configPath := filepath.Join(teamsDir, entry.Name(), "config.json")
		data, err := os.ReadFile(configPath)
		if err != nil {
			continue
		}
		var cfg struct {
			Members []interface{} `json:"members"`
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			continue
		}
		if len(cfg.Members) > 0 {
			return &teamInfo{TeamName: entry.Name(), MemberCount: len(cfg.Members)}
		}
	}
	return nil
}
