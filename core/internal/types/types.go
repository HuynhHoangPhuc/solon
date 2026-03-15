// Package types provides shared types for the sc binary.
// Ported from solon-hooks/internal/hookio/types.go — keep in sync.
package types

// SLConfig mirrors the .sl.json configuration structure.
type SLConfig struct {
	Plan        PlanConfig      `json:"plan"`
	Paths       PathsConfig     `json:"paths"`
	Docs        DocsConfig      `json:"docs"`
	Locale      LocaleConfig    `json:"locale"`
	Trust       TrustConfig     `json:"trust"`
	Project     ProjectConfig   `json:"project"`
	Skills      SkillsConfig    `json:"skills"`
	Hooks       map[string]bool `json:"hooks"`
	Assertions  []string        `json:"assertions"`
	CodingLevel int             `json:"codingLevel"`
	Statusline  string          `json:"statusline"`
	Subagent    *SubagentConfig `json:"subagent,omitempty"`
}

// PlanConfig holds plan naming and resolution settings.
type PlanConfig struct {
	NamingFormat string               `json:"namingFormat"`
	DateFormat   string               `json:"dateFormat"`
	IssuePrefix  *string              `json:"issuePrefix"`
	ReportsDir   string               `json:"reportsDir"`
	Resolution   PlanResolutionConfig `json:"resolution"`
	Validation   PlanValidationConfig `json:"validation"`
}

// PlanResolutionConfig defines how an active plan is resolved.
type PlanResolutionConfig struct {
	Order         []string `json:"order"`
	BranchPattern string   `json:"branchPattern"`
}

// PlanValidationConfig defines plan validation question settings.
type PlanValidationConfig struct {
	Mode         string   `json:"mode"`
	MinQuestions int      `json:"minQuestions"`
	MaxQuestions int      `json:"maxQuestions"`
	FocusAreas   []string `json:"focusAreas"`
}

// PathsConfig holds relative directory paths for docs and plans.
type PathsConfig struct {
	Docs  string `json:"docs"`
	Plans string `json:"plans"`
}

// DocsConfig holds documentation constraints.
type DocsConfig struct {
	MaxLoc int `json:"maxLoc"`
}

// LocaleConfig holds language preferences.
type LocaleConfig struct {
	ThinkingLanguage *string `json:"thinkingLanguage"`
	ResponseLanguage *string `json:"responseLanguage"`
}

// TrustConfig holds trust/passphrase settings.
type TrustConfig struct {
	Passphrase *string `json:"passphrase"`
	Enabled    bool    `json:"enabled"`
}

// ProjectConfig holds project type/framework overrides.
type ProjectConfig struct {
	Type           string `json:"type"`
	PackageManager string `json:"packageManager"`
	Framework      string `json:"framework"`
}

// SkillsConfig holds skill-specific settings.
type SkillsConfig struct {
	Research ResearchConfig `json:"research"`
}

// ResearchConfig holds research skill settings.
type ResearchConfig struct {
	UseGemini bool `json:"useGemini"`
}

// SubagentConfig holds per-agent context prefix overrides.
type SubagentConfig struct {
	Agents map[string]AgentConfig `json:"agents,omitempty"`
}

// AgentConfig holds per-agent configuration.
type AgentConfig struct {
	ContextPrefix string `json:"contextPrefix,omitempty"`
}

// PlanResolution is the result of resolving an active plan path.
type PlanResolution struct {
	Path       string // empty = not resolved
	ResolvedBy string // "session", "branch", or ""
}

// SessionState is persisted per-session in /tmp/sl-session-{id}.json.
type SessionState struct {
	SessionOrigin string  `json:"sessionOrigin"`
	ActivePlan    *string `json:"activePlan"`
	SuggestedPlan *string `json:"suggestedPlan"`
	Timestamp     int64   `json:"timestamp"`
	Source        string  `json:"source"`
}
