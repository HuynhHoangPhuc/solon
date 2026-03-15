// Package config handles SL config loading with cascade:
// DEFAULT → global (~/.claude/.sl.json) → local (.claude/.sl.json)
// Ported from solon-hooks/internal/config/config.go — keep in sync.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"solon-core/internal/types"
)

const localConfigPath = ".claude/.sl.json"

// globalConfigPath returns ~/.claude/.sl.json
func globalConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", ".sl.json")
}

// DefaultConfig is the factory default configuration.
var DefaultConfig = types.SLConfig{
	Plan: types.PlanConfig{
		NamingFormat: "{date}-{issue}-{slug}",
		DateFormat:   "YYMMDD-HHmm",
		IssuePrefix:  nil,
		ReportsDir:   "reports",
		Resolution: types.PlanResolutionConfig{
			Order:         []string{"session", "branch"},
			BranchPattern: `(?:feat|fix|chore|refactor|docs)/(?:[^/]+/)?(.+)`,
		},
		Validation: types.PlanValidationConfig{
			Mode:         "prompt",
			MinQuestions: 3,
			MaxQuestions: 8,
			FocusAreas:   []string{"assumptions", "risks", "tradeoffs", "architecture"},
		},
	},
	Paths:   types.PathsConfig{Docs: "docs", Plans: "plans"},
	Docs:    types.DocsConfig{MaxLoc: 800},
	Locale:  types.LocaleConfig{},
	Trust:   types.TrustConfig{Enabled: false},
	Project: types.ProjectConfig{Type: "auto", PackageManager: "auto", Framework: "auto"},
	Skills:  types.SkillsConfig{Research: types.ResearchConfig{UseGemini: true}},
	Assertions: []string{},
	Statusline: "full",
	Hooks: map[string]bool{
		"session-init":                     true,
		"subagent-init":                    true,
		"dev-rules-reminder":               true,
		"usage-context-awareness":          true,
		"scout-block":                      true,
		"privacy-block":                    true,
		"post-edit-simplify-reminder":      true,
		"task-completed-handler":           true,
		"teammate-idle-handler":            true,
		"preemptive-compaction":            true,
		"tool-output-truncation":           true,
		"todo-continuation-enforcer":       true,
		"comment-slop-checker":             true,
		"wisdom-accumulation":              true,
		"compaction-context-preservation":  true,
		"intent-gate":                      true,
		"semantic-compression":             true,
	},
	CodingLevel: -1,
}

// LoadConfigFromPath reads and parses a JSON config file.
// Returns nil if the file does not exist or cannot be parsed.
func LoadConfigFromPath(path string) map[string]interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

// DeepMerge merges source into target. Arrays are replaced entirely.
// Empty objects {} are treated as "inherit from parent" (no override).
func DeepMerge(target, source types.SLConfig) types.SLConfig {
	targetMap := structToMap(target)
	sourceMap := structToMap(source)
	merged := deepMergeMaps(targetMap, sourceMap)
	return mapToSLConfig(merged, target)
}

func structToMap(v interface{}) map[string]interface{} {
	data, _ := json.Marshal(v)
	var m map[string]interface{}
	_ = json.Unmarshal(data, &m)
	return m
}

func deepMergeMaps(target, source map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range target {
		result[k] = v
	}
	for k, sourceVal := range source {
		if sourceVal == nil {
			continue
		}
		targetVal := result[k]
		switch sv := sourceVal.(type) {
		case []interface{}:
			result[k] = sv
		case map[string]interface{}:
			if len(sv) == 0 {
				continue
			}
			if tv, ok := targetVal.(map[string]interface{}); ok {
				result[k] = deepMergeMaps(tv, sv)
			} else {
				result[k] = sv
			}
		default:
			result[k] = sourceVal
		}
	}
	return result
}

func mapToSLConfig(m map[string]interface{}, fallback types.SLConfig) types.SLConfig {
	data, err := json.Marshal(m)
	if err != nil {
		return fallback
	}
	var cfg types.SLConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fallback
	}
	return cfg
}

// SanitizeConfig validates config paths against the project root.
func SanitizeConfig(cfg types.SLConfig, projectRoot string) types.SLConfig {
	result := cfg
	if sanitizePath(result.Plan.ReportsDir, projectRoot) == "" {
		result.Plan.ReportsDir = DefaultConfig.Plan.ReportsDir
	}
	if result.Plan.Resolution.Order == nil {
		result.Plan.Resolution.Order = DefaultConfig.Plan.Resolution.Order
	}
	if result.Plan.Resolution.BranchPattern == "" {
		result.Plan.Resolution.BranchPattern = DefaultConfig.Plan.Resolution.BranchPattern
	}
	if sanitizePath(result.Paths.Docs, projectRoot) == "" {
		result.Paths.Docs = DefaultConfig.Paths.Docs
	}
	if sanitizePath(result.Paths.Plans, projectRoot) == "" {
		result.Paths.Plans = DefaultConfig.Paths.Plans
	}
	return result
}

// sanitizePath validates a path value — prevents traversal, allows absolute paths.
func sanitizePath(pathValue, projectRoot string) string {
	normalized := normalizePath(pathValue)
	if normalized == "" {
		return ""
	}
	if strings.ContainsRune(normalized, 0) {
		return ""
	}
	if filepath.IsAbs(normalized) {
		return normalized
	}
	resolved := filepath.Join(projectRoot, normalized)
	sep := string(filepath.Separator)
	if !strings.HasPrefix(resolved, projectRoot+sep) && resolved != projectRoot {
		return ""
	}
	return normalized
}

// normalizePath trims whitespace and trailing slashes from a path.
func normalizePath(pathValue string) string {
	if pathValue == "" {
		return ""
	}
	normalized := strings.TrimSpace(pathValue)
	normalized = strings.TrimRight(normalized, "/\\")
	return normalized
}

// LoadConfig loads config with cascade: DEFAULT → global → local.
func LoadConfig() types.SLConfig {
	projectRoot, _ := os.Getwd()
	globalRaw := LoadConfigFromPath(globalConfigPath())
	localRaw := LoadConfigFromPath(filepath.Join(projectRoot, localConfigPath))

	if globalRaw == nil && localRaw == nil {
		return DefaultConfig
	}

	defer func() { recover() }()

	merged := structToMap(DefaultConfig)
	if globalRaw != nil {
		merged = deepMergeMaps(merged, globalRaw)
	}
	if localRaw != nil {
		merged = deepMergeMaps(merged, localRaw)
	}

	cfg := mapToSLConfig(merged, DefaultConfig)
	if cfg.Hooks == nil {
		cfg.Hooks = DefaultConfig.Hooks
	}
	if cfg.Statusline == "" {
		cfg.Statusline = "full"
	}

	return SanitizeConfig(cfg, projectRoot)
}
