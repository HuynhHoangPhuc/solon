// Package config handles SL config loading with cascade:
// DEFAULT → global (~/.claude/.sl.json) → local (.claude/.sl.json)
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"solon-hooks/internal/hookio"
)

const localConfigPath = ".claude/.sl.json"

// globalConfigPath returns ~/.claude/.sl.json
func globalConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", ".sl.json")
}

// strPtr is a helper to make a *string from a literal.
func strPtr(s string) *string { return &s }

// DefaultConfig is the factory default configuration.
var DefaultConfig = hookio.SLConfig{
	Plan: hookio.PlanConfig{
		NamingFormat: "{date}-{issue}-{slug}",
		DateFormat:   "YYMMDD-HHmm",
		IssuePrefix:  nil,
		ReportsDir:   "reports",
		Resolution: hookio.PlanResolutionConfig{
			Order:         []string{"session", "branch"},
			BranchPattern: `(?:feat|fix|chore|refactor|docs)/(?:[^/]+/)?(.+)`,
		},
		Validation: hookio.PlanValidationConfig{
			Mode:         "prompt",
			MinQuestions: 3,
			MaxQuestions: 8,
			FocusAreas:   []string{"assumptions", "risks", "tradeoffs", "architecture"},
		},
	},
	Paths: hookio.PathsConfig{Docs: "docs", Plans: "plans"},
	Docs:  hookio.DocsConfig{MaxLoc: 800},
	Locale: hookio.LocaleConfig{
		ThinkingLanguage: nil,
		ResponseLanguage: nil,
	},
	Trust:   hookio.TrustConfig{Passphrase: nil, Enabled: false},
	Project: hookio.ProjectConfig{Type: "auto", PackageManager: "auto", Framework: "auto"},
	Skills:  hookio.SkillsConfig{Research: hookio.ResearchConfig{UseGemini: true}},
	Assertions: []string{},
	Statusline: "full",
	Hooks: map[string]bool{
		"session-init":               true,
		"subagent-init":              true,
		"dev-rules-reminder":         true,
		"usage-context-awareness":    true,
		"scout-block":                true,
		"privacy-block":              true,
		"post-edit-simplify-reminder": true,
		"task-completed-handler":     true,
		"teammate-idle-handler":      true,
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
// Uses reflect to handle nested structs via intermediate map representation.
func DeepMerge(target, source hookio.SLConfig) hookio.SLConfig {
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
			// Arrays: replace entirely
			result[k] = sv
		case map[string]interface{}:
			// Empty object = inherit (no override)
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

func mapToSLConfig(m map[string]interface{}, fallback hookio.SLConfig) hookio.SLConfig {
	data, err := json.Marshal(m)
	if err != nil {
		return fallback
	}
	var cfg hookio.SLConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fallback
	}
	return cfg
}

// SanitizeConfig validates config paths against the project root.
func SanitizeConfig(cfg hookio.SLConfig, projectRoot string) hookio.SLConfig {
	result := cfg

	// Validate plan paths
	if SanitizePath(result.Plan.ReportsDir, projectRoot) == "" {
		result.Plan.ReportsDir = DefaultConfig.Plan.ReportsDir
	}
	// Merge resolution/validation with defaults (shallow)
	if result.Plan.Resolution.Order == nil {
		result.Plan.Resolution.Order = DefaultConfig.Plan.Resolution.Order
	}
	if result.Plan.Resolution.BranchPattern == "" {
		result.Plan.Resolution.BranchPattern = DefaultConfig.Plan.Resolution.BranchPattern
	}

	// Validate paths
	if SanitizePath(result.Paths.Docs, projectRoot) == "" {
		result.Paths.Docs = DefaultConfig.Paths.Docs
	}
	if SanitizePath(result.Paths.Plans, projectRoot) == "" {
		result.Paths.Plans = DefaultConfig.Paths.Plans
	}

	return result
}

// SanitizePath validates a path value — prevents traversal, allows absolute paths.
// Returns empty string if the path is unsafe or invalid.
func SanitizePath(pathValue, projectRoot string) string {
	normalized := NormalizePath(pathValue)
	if normalized == "" {
		return ""
	}
	// Reject null bytes
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

// NormalizePath trims whitespace and trailing slashes from a path.
func NormalizePath(pathValue string) string {
	if pathValue == "" {
		return ""
	}
	normalized := strings.TrimSpace(pathValue)
	normalized = strings.TrimRight(normalized, "/\\")
	return normalized
}

// EscapeShellValue escapes shell special characters for use in env file values.
func EscapeShellValue(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, `$`, `\$`)
	s = strings.ReplaceAll(s, "`", "\\`")
	return s
}

// WriteEnv appends an exported env variable to envFile.
func WriteEnv(envFile, key string, value interface{}) {
	if envFile == "" || value == nil || isZero(value) {
		return
	}
	escaped := EscapeShellValue(fmt.Sprintf("%v", value))
	line := fmt.Sprintf("export %s=\"%s\"\n", key, escaped)
	f, err := os.OpenFile(envFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line)
}

func isZero(v interface{}) bool {
	if v == nil {
		return true
	}
	return reflect.DeepEqual(v, reflect.Zero(reflect.TypeOf(v)).Interface())
}

// LoadConfigOptions controls which sections to populate.
type LoadConfigOptions struct {
	IncludeProject    bool
	IncludeAssertions bool
	IncludeLocale     bool
}

// LoadConfig loads config with cascade: DEFAULT → global → local.
func LoadConfig(opts LoadConfigOptions) hookio.SLConfig {
	projectRoot, _ := os.Getwd()

	globalRaw := LoadConfigFromPath(globalConfigPath())
	localRaw := LoadConfigFromPath(filepath.Join(projectRoot, localConfigPath))

	if globalRaw == nil && localRaw == nil {
		return getDefaultConfig(opts)
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

	// Apply option filters
	if !opts.IncludeLocale {
		cfg.Locale = DefaultConfig.Locale
	}
	if !opts.IncludeProject {
		cfg.Project = DefaultConfig.Project
	}
	if !opts.IncludeAssertions {
		cfg.Assertions = []string{}
	}
	if cfg.Hooks == nil {
		cfg.Hooks = DefaultConfig.Hooks
	}
	if cfg.Statusline == "" {
		cfg.Statusline = "full"
	}

	return SanitizeConfig(cfg, projectRoot)
}

func getDefaultConfig(opts LoadConfigOptions) hookio.SLConfig {
	cfg := DefaultConfig
	if !opts.IncludeLocale {
		cfg.Locale = DefaultConfig.Locale
	}
	if !opts.IncludeProject {
		cfg.Project = DefaultConfig.Project
	}
	if !opts.IncludeAssertions {
		cfg.Assertions = []string{}
	}
	return cfg
}

// IsHookEnabled returns true if hookName is enabled in config (default: true).
func IsHookEnabled(hookName string) bool {
	cfg := LoadConfig(LoadConfigOptions{})
	hooks := cfg.Hooks
	if hooks == nil {
		return true
	}
	val, exists := hooks[hookName]
	if !exists {
		return true
	}
	return val
}
