// Package hookio provides shared types and I/O helpers for all hook scripts.
package hookio

// HookInput is the base JSON received from stdin for every hook event.
type HookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	PermissionMode string `json:"permission_mode"`
	HookEventName  string `json:"hook_event_name"`
	AgentID        string `json:"agent_id,omitempty"`
	AgentType      string `json:"agent_type,omitempty"`
}

// SessionStartInput is the payload for the SessionStart hook event.
type SessionStartInput struct {
	HookInput
	Source string `json:"source"` // startup|resume|clear|compact
	Model  string `json:"model"`
}

// SubagentStartInput is the payload for the SubagentStart hook event.
type SubagentStartInput struct {
	HookInput
}

// SubagentStopInput is the payload for the SubagentStop hook event.
type SubagentStopInput struct {
	HookInput
}

// UserPromptSubmitInput is the payload for the UserPromptSubmit hook event.
type UserPromptSubmitInput struct {
	HookInput
	Prompt string `json:"prompt"`
}

// PreToolUseInput is the payload for the PreToolUse hook event.
type PreToolUseInput struct {
	HookInput
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
}

// PostToolUseInput is the payload for the PostToolUse hook event.
type PostToolUseInput struct {
	HookInput
	ToolName   string                 `json:"tool_name"`
	ToolInput  map[string]interface{} `json:"tool_input"`
	ToolOutput string                 `json:"tool_output,omitempty"`
}

// TaskCompletedInput is the payload for the TaskCompleted hook event.
type TaskCompletedInput struct {
	HookInput
	TaskID          string `json:"task_id"`
	TaskSubject     string `json:"task_subject"`
	TaskDescription string `json:"task_description,omitempty"`
	TeammateName    string `json:"teammate_name"`
	TeamName        string `json:"team_name"`
}

// TeammateIdleInput is the payload for the TeammateIdle hook event.
type TeammateIdleInput struct {
	HookInput
	TeammateName string `json:"teammate_name"`
	TeamName     string `json:"team_name"`
}

// StopInput is the payload for the Stop hook event.
type StopInput struct {
	HookInput
}

// HookOutput is the JSON written to stdout to control Claude's behavior.
type HookOutput struct {
	Continue           *bool                  `json:"continue,omitempty"`
	AdditionalContext  string                 `json:"additionalContext,omitempty"`
	HookSpecificOutput map[string]interface{} `json:"hookSpecificOutput,omitempty"`
}

// SLConfig mirrors the TypeScript SLConfig interface — the main config struct.
type SLConfig struct {
	Plan        PlanConfig        `json:"plan"`
	Paths       PathsConfig       `json:"paths"`
	Docs        DocsConfig        `json:"docs"`
	Locale      LocaleConfig      `json:"locale"`
	Trust       TrustConfig       `json:"trust"`
	Project     ProjectConfig     `json:"project"`
	Skills      SkillsConfig      `json:"skills"`
	Hooks       map[string]bool   `json:"hooks"`
	Assertions  []string          `json:"assertions"`
	CodingLevel int               `json:"codingLevel"`
	Statusline  string            `json:"statusline"`
	Subagent    *SubagentConfig   `json:"subagent,omitempty"`
}

// PlanConfig holds plan naming and resolution settings.
type PlanConfig struct {
	NamingFormat string              `json:"namingFormat"`
	DateFormat   string              `json:"dateFormat"`
	IssuePrefix  *string             `json:"issuePrefix"` // null = no prefix
	ReportsDir   string              `json:"reportsDir"`
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
	Path       string // empty string = not resolved
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

// ProjectInfo holds detected project and environment information.
type ProjectInfo struct {
	Type           string  `json:"type"`
	PackageManager *string `json:"packageManager"`
	Framework      *string `json:"framework"`
	PythonVersion  *string `json:"pythonVersion"`
	NodeVersion    string  `json:"nodeVersion"`
	GitBranch      *string `json:"gitBranch"`
	GitRoot        *string `json:"gitRoot"`
	GitURL         *string `json:"gitUrl"`
	OSPlatform     string  `json:"osPlatform"`
	User           string  `json:"user"`
	Locale         string  `json:"locale"`
	Timezone       string  `json:"timezone"`
}

// ConfigCounts holds counted configuration artifacts.
type ConfigCounts struct {
	ClaudeMdCount int `json:"claudeMdCount"`
	RulesCount    int `json:"rulesCount"`
	McpCount      int `json:"mcpCount"`
	HooksCount    int `json:"hooksCount"`
}

// ScoutBlockResult is the result of a scout-block check.
type ScoutBlockResult struct {
	Blocked          bool     `json:"blocked"`
	Path             string   `json:"path,omitempty"`
	Pattern          string   `json:"pattern,omitempty"`
	Reason           string   `json:"reason,omitempty"`
	IsBroadPattern   bool     `json:"isBroadPattern,omitempty"`
	Suggestions      []string `json:"suggestions,omitempty"`
	IsAllowedCommand bool     `json:"isAllowedCommand,omitempty"`
}

// PrivacyCheckResult is the result of a privacy-block check.
type PrivacyCheckResult struct {
	Blocked    bool                   `json:"blocked"`
	FilePath   string                 `json:"filePath,omitempty"`
	Reason     string                 `json:"reason,omitempty"`
	Approved   bool                   `json:"approved,omitempty"`
	IsBash     bool                   `json:"isBash,omitempty"`
	Suspicious bool                   `json:"suspicious,omitempty"`
	PromptData map[string]interface{} `json:"promptData,omitempty"`
}
