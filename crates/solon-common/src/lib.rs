/// Shared types and utilities for Solon crates.
pub use anyhow;
pub use serde;
pub use serde_json;

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// SLConfig mirrors the .sl.json configuration structure.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(rename_all = "camelCase")]
pub struct SLConfig {
    pub plan: PlanConfig,
    pub paths: PathsConfig,
    pub docs: DocsConfig,
    pub locale: LocaleConfig,
    pub trust: TrustConfig,
    pub project: ProjectConfig,
    pub skills: SkillsConfig,
    #[serde(default)]
    pub hooks: HashMap<String, bool>,
    #[serde(default)]
    pub assertions: Vec<String>,
    #[serde(default)]
    pub coding_level: i32,
    #[serde(default)]
    pub statusline: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub subagent: Option<SubagentConfig>,
}

/// PlanConfig holds plan naming and resolution settings.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PlanConfig {
    pub naming_format: String,
    pub date_format: String,
    pub issue_prefix: Option<String>,
    pub reports_dir: String,
    pub resolution: PlanResolutionConfig,
    pub validation: PlanValidationConfig,
}

impl Default for PlanConfig {
    fn default() -> Self {
        Self {
            naming_format: "{date}-{issue}-{slug}".to_string(),
            date_format: "YYMMDD-HHmm".to_string(),
            issue_prefix: None,
            reports_dir: "reports".to_string(),
            resolution: PlanResolutionConfig::default(),
            validation: PlanValidationConfig::default(),
        }
    }
}

/// PlanResolutionConfig defines how an active plan is resolved.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PlanResolutionConfig {
    pub order: Vec<String>,
    pub branch_pattern: String,
}

impl Default for PlanResolutionConfig {
    fn default() -> Self {
        Self {
            order: vec!["session".to_string(), "branch".to_string()],
            branch_pattern: r"(?:feat|fix|chore|refactor|docs)/(?:[^/]+/)?(.+)".to_string(),
        }
    }
}

/// PlanValidationConfig defines plan validation question settings.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PlanValidationConfig {
    pub mode: String,
    pub min_questions: i32,
    pub max_questions: i32,
    pub focus_areas: Vec<String>,
}

impl Default for PlanValidationConfig {
    fn default() -> Self {
        Self {
            mode: "prompt".to_string(),
            min_questions: 3,
            max_questions: 8,
            focus_areas: vec![
                "assumptions".to_string(),
                "risks".to_string(),
                "tradeoffs".to_string(),
                "architecture".to_string(),
            ],
        }
    }
}

/// PathsConfig holds relative directory paths for docs and plans.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PathsConfig {
    pub docs: String,
    pub plans: String,
}

impl Default for PathsConfig {
    fn default() -> Self {
        Self {
            docs: "docs".to_string(),
            plans: "plans".to_string(),
        }
    }
}

/// DocsConfig holds documentation constraints.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(rename_all = "camelCase")]
pub struct DocsConfig {
    pub max_loc: i32,
}

/// LocaleConfig holds language preferences.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(rename_all = "camelCase")]
pub struct LocaleConfig {
    pub thinking_language: Option<String>,
    pub response_language: Option<String>,
}

/// TrustConfig holds trust/passphrase settings.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct TrustConfig {
    pub passphrase: Option<String>,
    pub enabled: bool,
}

/// ProjectConfig holds project type/framework overrides.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProjectConfig {
    #[serde(rename = "type")]
    pub project_type: String,
    #[serde(rename = "packageManager")]
    pub package_manager: String,
    pub framework: String,
}

impl Default for ProjectConfig {
    fn default() -> Self {
        Self {
            project_type: "auto".to_string(),
            package_manager: "auto".to_string(),
            framework: "auto".to_string(),
        }
    }
}

/// SkillsConfig holds skill-specific settings.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct SkillsConfig {
    pub research: ResearchConfig,
}

/// ResearchConfig holds research skill settings.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ResearchConfig {
    pub use_gemini: bool,
}

impl Default for ResearchConfig {
    fn default() -> Self {
        Self { use_gemini: true }
    }
}

/// SubagentConfig holds per-agent context prefix overrides.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct SubagentConfig {
    pub agents: Option<HashMap<String, AgentConfig>>,
}

/// AgentConfig holds per-agent configuration.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(rename_all = "camelCase")]
pub struct AgentConfig {
    pub context_prefix: Option<String>,
}

/// PlanResolution is the result of resolving an active plan path.
#[derive(Debug, Clone, Default)]
pub struct PlanResolution {
    /// Empty string = not resolved
    pub path: String,
    /// "session", "branch", or ""
    pub resolved_by: String,
}

/// SessionState is persisted per-session in /tmp/sl-session-{id}.json.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(rename_all = "camelCase")]
pub struct SessionState {
    pub session_origin: String,
    pub active_plan: Option<String>,
    pub suggested_plan: Option<String>,
    pub timestamp: i64,
    pub source: String,
}
