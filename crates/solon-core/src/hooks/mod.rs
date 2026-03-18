/// Hook dispatcher and shared I/O helpers for all hook subcommands.
/// Each hook reads JSON from stdin, processes logic, writes result to stdout.
/// Exit code: 0=allow, 2=block (explicit deny).
pub mod comment_slop_checker;
pub mod descriptive_name;
pub mod dev_rules;
pub mod intent_gate;
pub mod post_edit;
pub mod preemptive_compaction;
pub mod privacy_block;
pub mod scout_block;
pub mod session_init;
pub mod ship_reminder;
pub mod statusline;
pub mod subagent_init;
pub mod task_completed;
pub mod team_context;
pub mod teammate_idle;
pub mod todo_continuation_enforcer;
pub mod tool_output_truncation;
pub mod usage_awareness;
pub mod version;
pub mod wisdom_accumulator;

use anyhow::Result;
use serde_json::Value;
use std::io::{self, Read};

/// Read JSON from stdin. Returns error if stdin is empty or invalid JSON.
pub fn read_hook_input() -> Result<Value> {
    let mut buf = String::new();
    io::stdin().read_to_string(&mut buf)?;
    if buf.trim().is_empty() {
        anyhow::bail!("empty stdin");
    }
    let v: Value = serde_json::from_str(&buf)?;
    Ok(v)
}

/// Get an env var value, returning None if unset or empty.
pub fn get_env(key: &str) -> Option<String> {
    std::env::var(key).ok().filter(|v| !v.is_empty())
}

/// Write JSON output to stdout.
pub fn write_output(v: &Value) {
    if let Ok(s) = serde_json::to_string(v) {
        println!("{}", s);
    }
}

/// Write plain text to stdout (context injection).
pub fn write_context(text: &str) {
    print!("{}", text);
}

/// Write message to stderr and exit with code 2 (explicit block).
pub fn block(message: &str) -> ! {
    eprint!("{}", message);
    std::process::exit(2);
}

/// Log a prefixed message to stderr (non-blocking).
pub fn log_hook(hook_name: &str, message: &str) {
    eprintln!("[{}] {}", hook_name, message);
}

/// Extract string field from a JSON object.
pub fn get_str<'a>(v: &'a Value, key: &str) -> &'a str {
    v.get(key).and_then(|x| x.as_str()).unwrap_or("")
}

/// Extract optional string field from a JSON object.
pub fn get_str_opt(v: &Value, key: &str) -> Option<String> {
    v.get(key)
        .and_then(|x| x.as_str())
        .filter(|s| !s.is_empty())
        .map(|s| s.to_string())
}

// ── Config helpers ──────────────────────────────────────────────────────────

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// SL configuration loaded from ~/.claude/.sl.json and .claude/.sl.json.
#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct SlConfig {
    #[serde(default)]
    pub plan: PlanConfig,
    #[serde(default)]
    pub paths: PathsConfig,
    #[serde(default)]
    pub docs: DocsConfig,
    #[serde(default)]
    pub locale: LocaleConfig,
    #[serde(default)]
    pub trust: TrustConfig,
    #[serde(default)]
    pub project: ProjectConfig,
    #[serde(default)]
    pub hooks: HashMap<String, bool>,
    #[serde(default)]
    pub assertions: Vec<String>,
    #[serde(default = "default_coding_level")]
    pub coding_level: i32,
    #[serde(default = "default_statusline")]
    pub statusline: String,
    #[serde(default)]
    pub subagent: Option<SubagentConfig>,
}

fn default_coding_level() -> i32 {
    -1
}
fn default_statusline() -> String {
    "full".to_string()
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PlanConfig {
    #[serde(default = "default_naming_format")]
    pub naming_format: String,
    #[serde(default = "default_date_format")]
    pub date_format: String,
    pub issue_prefix: Option<String>,
    #[serde(default = "default_reports_dir")]
    pub reports_dir: String,
    #[serde(default)]
    pub resolution: PlanResolutionConfig,
    #[serde(default)]
    pub validation: PlanValidationConfig,
}

impl Default for PlanConfig {
    fn default() -> Self {
        Self {
            naming_format: default_naming_format(),
            date_format: default_date_format(),
            issue_prefix: None,
            reports_dir: default_reports_dir(),
            resolution: PlanResolutionConfig::default(),
            validation: PlanValidationConfig::default(),
        }
    }
}

fn default_naming_format() -> String {
    "{date}-{issue}-{slug}".to_string()
}
fn default_date_format() -> String {
    "YYMMDD-HHmm".to_string()
}
fn default_reports_dir() -> String {
    "reports".to_string()
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PlanResolutionConfig {
    #[serde(default = "default_resolution_order")]
    pub order: Vec<String>,
    #[serde(default = "default_branch_pattern")]
    pub branch_pattern: String,
}

impl Default for PlanResolutionConfig {
    fn default() -> Self {
        Self {
            order: default_resolution_order(),
            branch_pattern: default_branch_pattern(),
        }
    }
}

fn default_resolution_order() -> Vec<String> {
    vec!["session".to_string(), "branch".to_string()]
}
fn default_branch_pattern() -> String {
    r"(?:feat|fix|chore|refactor|docs)/(?:[^/]+/)?(.+)".to_string()
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PlanValidationConfig {
    #[serde(default = "default_validation_mode")]
    pub mode: String,
    #[serde(default = "default_min_questions")]
    pub min_questions: u32,
    #[serde(default = "default_max_questions")]
    pub max_questions: u32,
    #[serde(default)]
    pub focus_areas: Vec<String>,
}

impl Default for PlanValidationConfig {
    fn default() -> Self {
        Self {
            mode: default_validation_mode(),
            min_questions: default_min_questions(),
            max_questions: default_max_questions(),
            focus_areas: vec![
                "assumptions".to_string(),
                "risks".to_string(),
                "tradeoffs".to_string(),
                "architecture".to_string(),
            ],
        }
    }
}

fn default_validation_mode() -> String {
    "prompt".to_string()
}
fn default_min_questions() -> u32 {
    3
}
fn default_max_questions() -> u32 {
    8
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PathsConfig {
    #[serde(default = "default_docs")]
    pub docs: String,
    #[serde(default = "default_plans")]
    pub plans: String,
}

impl Default for PathsConfig {
    fn default() -> Self {
        Self {
            docs: default_docs(),
            plans: default_plans(),
        }
    }
}

fn default_docs() -> String {
    "docs".to_string()
}
fn default_plans() -> String {
    "plans".to_string()
}

#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct DocsConfig {
    #[serde(default = "default_max_loc")]
    pub max_loc: u32,
}

fn default_max_loc() -> u32 {
    800
}

#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct LocaleConfig {
    pub thinking_language: Option<String>,
    pub response_language: Option<String>,
}

#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct TrustConfig {
    pub passphrase: Option<String>,
    #[serde(default)]
    pub enabled: bool,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct ProjectConfig {
    #[serde(default = "default_auto")]
    pub r#type: String,
    #[serde(default = "default_auto")]
    pub package_manager: String,
    #[serde(default = "default_auto")]
    pub framework: String,
}

impl Default for ProjectConfig {
    fn default() -> Self {
        Self {
            r#type: default_auto(),
            package_manager: default_auto(),
            framework: default_auto(),
        }
    }
}

fn default_auto() -> String {
    "auto".to_string()
}

#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct SubagentConfig {
    #[serde(default)]
    pub agents: HashMap<String, AgentConfig>,
}

#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct AgentConfig {
    #[serde(default)]
    pub context_prefix: String,
}

/// Session state persisted to /tmp/sl-session-{id}.json.
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct SessionState {
    #[serde(rename = "sessionOrigin")]
    pub session_origin: String,
    #[serde(rename = "activePlan")]
    pub active_plan: Option<String>,
    #[serde(rename = "suggestedPlan")]
    pub suggested_plan: Option<String>,
    pub timestamp: i64,
    pub source: String,
}

/// Plan resolution result.
#[derive(Debug, Clone, Default)]
pub struct PlanResolution {
    pub path: String,
    pub resolved_by: String, // "session", "branch", or ""
}

// ── Config loading ──────────────────────────────────────────────────────────

const LOCAL_CONFIG_PATH: &str = ".claude/.sl.json";

fn global_config_path() -> std::path::PathBuf {
    let home = dirs_home();
    home.join(".claude").join(".sl.json")
}

fn dirs_home() -> std::path::PathBuf {
    dirs::home_dir().unwrap_or_else(|| std::path::PathBuf::from("."))
}

fn load_raw_config(path: &std::path::Path) -> Option<serde_json::Map<String, Value>> {
    let data = std::fs::read_to_string(path).ok()?;
    let v: Value = serde_json::from_str(&data).ok()?;
    v.as_object().cloned()
}

fn deep_merge(
    target: &mut serde_json::Map<String, Value>,
    source: &serde_json::Map<String, Value>,
) {
    for (k, sv) in source {
        match sv {
            Value::Object(src_obj) if !src_obj.is_empty() => {
                let entry = target
                    .entry(k.clone())
                    .or_insert(Value::Object(Default::default()));
                if let Value::Object(tgt_obj) = entry {
                    deep_merge(tgt_obj, src_obj);
                } else {
                    *entry = sv.clone();
                }
            }
            Value::Null => {}
            _ => {
                target.insert(k.clone(), sv.clone());
            }
        }
    }
}

/// Default hooks enabled map.
fn default_hooks() -> HashMap<String, bool> {
    let mut m = HashMap::new();
    for k in &[
        "session-init",
        "subagent-init",
        "dev-rules-reminder",
        "usage-context-awareness",
        "scout-block",
        "privacy-block",
        "post-edit-simplify-reminder",
        "task-completed-handler",
        "teammate-idle-handler",
        "preemptive-compaction",
        "tool-output-truncation",
        "todo-continuation-enforcer",
        "comment-slop-checker",
        "wisdom-accumulation",
        "compaction-context-preservation",
        "intent-gate",
        "semantic-compression",
    ] {
        m.insert(k.to_string(), true);
    }
    m
}

pub fn load_config() -> SlConfig {
    let cwd = std::env::current_dir().unwrap_or_default();

    let global_raw = load_raw_config(&global_config_path());
    let local_raw = load_raw_config(&cwd.join(LOCAL_CONFIG_PATH));

    if global_raw.is_none() && local_raw.is_none() {
        return SlConfig {
            hooks: default_hooks(),
            ..Default::default()
        };
    }

    // Start with default serialized
    let default_cfg = SlConfig {
        hooks: default_hooks(),
        ..Default::default()
    };
    let mut merged: serde_json::Map<String, Value> = match serde_json::to_value(&default_cfg) {
        Ok(Value::Object(m)) => m,
        _ => Default::default(),
    };

    if let Some(g) = &global_raw {
        deep_merge(&mut merged, g);
    }
    if let Some(l) = &local_raw {
        deep_merge(&mut merged, l);
    }

    let mut cfg: SlConfig = serde_json::from_value(Value::Object(merged)).unwrap_or_default();

    if cfg.hooks.is_empty() {
        cfg.hooks = default_hooks();
    }
    if cfg.statusline.is_empty() {
        cfg.statusline = "full".to_string();
    }

    cfg
}

/// Check if a hook is enabled (default: true for unknown hooks).
pub fn is_hook_enabled(hook_name: &str) -> bool {
    let cfg = load_config();
    match cfg.hooks.get(hook_name) {
        Some(&v) => v,
        None => true,
    }
}

// ── Session state ───────────────────────────────────────────────────────────

pub fn session_temp_path(session_id: &str) -> std::path::PathBuf {
    let tmp = std::env::temp_dir();
    tmp.join(format!("sl-session-{}.json", session_id))
}

pub fn read_session_state(session_id: &str) -> Option<SessionState> {
    if session_id.is_empty() {
        return None;
    }
    let data = std::fs::read_to_string(session_temp_path(session_id)).ok()?;
    serde_json::from_str(&data).ok()
}

pub fn write_session_state(session_id: &str, state: &SessionState) -> bool {
    if session_id.is_empty() {
        return false;
    }
    let target = session_temp_path(session_id);
    let tmp = target.with_extension(format!("tmp.{}", rand_suffix()));
    let data = serde_json::to_string_pretty(state).unwrap_or_default();
    if std::fs::write(&tmp, data).is_err() {
        return false;
    }
    std::fs::rename(&tmp, &target).is_ok()
}

fn rand_suffix() -> String {
    use std::time::{SystemTime, UNIX_EPOCH};
    let t = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default();
    format!("{:x}", t.subsec_nanos())
}

// ── Plan helpers ────────────────────────────────────────────────────────────

pub fn normalize_path(p: &str) -> String {
    p.trim().trim_end_matches(['/', '\\']).to_string()
}

pub fn format_date(format: &str) -> String {
    let now = chrono::Local::now();
    format
        .replace("YYYY", &format!("{:04}", now.year()))
        .replace("YY", &format!("{:02}", now.year() % 100))
        .replace("MM", &format!("{:02}", now.month()))
        .replace("DD", &format!("{:02}", now.day()))
        .replace("HH", &format!("{:02}", now.hour()))
        .replace("mm", &format!("{:02}", now.minute()))
        .replace("ss", &format!("{:02}", now.second()))
}

use chrono::{Datelike, Timelike};

pub fn resolve_plan_path(session_id: &str, cfg: &SlConfig) -> PlanResolution {
    let plans_dir = normalize_path(&cfg.paths.plans);
    let plans_dir = if plans_dir.is_empty() {
        "plans".to_string()
    } else {
        plans_dir
    };
    let order = if cfg.plan.resolution.order.is_empty() {
        vec!["session".to_string(), "branch".to_string()]
    } else {
        cfg.plan.resolution.order.clone()
    };

    for method in &order {
        match method.as_str() {
            "session" => {
                if let Some(state) = read_session_state(session_id) {
                    if let Some(ap) = &state.active_plan {
                        if !ap.is_empty() {
                            let resolved = if std::path::Path::new(ap).is_absolute() {
                                ap.clone()
                            } else if !state.session_origin.is_empty() {
                                format!("{}/{}", state.session_origin, ap)
                            } else {
                                ap.clone()
                            };
                            return PlanResolution {
                                path: resolved,
                                resolved_by: "session".to_string(),
                            };
                        }
                    }
                }
            }
            "branch" => {
                let branch = exec_safe("git branch --show-current", "", 5000);
                let slug = extract_slug_from_branch(&branch, &cfg.plan.resolution.branch_pattern);
                if slug.is_empty() {
                    continue;
                }
                if let Ok(entries) = std::fs::read_dir(&plans_dir) {
                    let mut found = String::new();
                    for entry in entries.flatten() {
                        if entry.path().is_dir() {
                            let name = entry.file_name().to_string_lossy().to_string();
                            if name.contains(&slug) {
                                found = name;
                            }
                        }
                    }
                    if !found.is_empty() {
                        return PlanResolution {
                            path: format!("{}/{}", plans_dir, found),
                            resolved_by: "branch".to_string(),
                        };
                    }
                }
            }
            _ => {}
        }
    }
    PlanResolution::default()
}

pub fn get_reports_path(
    plan_path: &str,
    resolved_by: &str,
    plan_cfg: &PlanConfig,
    paths_cfg: &PathsConfig,
    base_dir: &str,
) -> String {
    let reports_dir = normalize_path(&plan_cfg.reports_dir);
    let reports_dir = if reports_dir.is_empty() {
        "reports".to_string()
    } else {
        reports_dir
    };
    let plans_dir = normalize_path(&paths_cfg.plans);
    let plans_dir = if plans_dir.is_empty() {
        "plans".to_string()
    } else {
        plans_dir
    };

    let report_path = if !plan_path.is_empty() && resolved_by == "session" {
        let np = normalize_path(plan_path);
        if !np.is_empty() {
            format!("{}/{}", np, reports_dir)
        } else {
            String::new()
        }
    } else {
        String::new()
    };
    let report_path = if report_path.is_empty() {
        format!("{}/{}", plans_dir, reports_dir)
    } else {
        report_path
    };

    if !base_dir.is_empty() {
        format!("{}/{}", base_dir, report_path)
    } else {
        format!("{}/", report_path)
    }
}

pub fn resolve_naming_pattern(plan_cfg: &PlanConfig, git_branch: &str) -> String {
    let formatted_date = format_date(&plan_cfg.date_format);
    let issue_id = extract_issue_from_branch(git_branch);
    let full_issue = if !issue_id.is_empty() {
        if let Some(prefix) = &plan_cfg.issue_prefix {
            if !prefix.is_empty() {
                format!("{}{}", prefix, issue_id)
            } else {
                String::new()
            }
        } else {
            String::new()
        }
    } else {
        String::new()
    };

    let mut pattern = plan_cfg.naming_format.replace("{date}", &formatted_date);
    if !full_issue.is_empty() {
        pattern = pattern.replace("{issue}", &full_issue);
    } else {
        let re = regex::Regex::new(r"-?\{issue\}-?").unwrap();
        pattern = re.replace_all(&pattern, "-").to_string();
        let re2 = regex::Regex::new(r"--+").unwrap();
        pattern = re2.replace_all(&pattern, "-").to_string();
    }
    // Clean up edges around {slug}
    pattern = pattern.trim_start_matches('-').to_string();
    pattern = pattern.trim_end_matches('-').to_string();
    let re3 = regex::Regex::new(r"-+(\{slug\})").unwrap();
    pattern = re3.replace_all(&pattern, "-$1").to_string();
    let re4 = regex::Regex::new(r"(\{slug\})-+").unwrap();
    pattern = re4.replace_all(&pattern, "$1-").to_string();
    let re5 = regex::Regex::new(r"--+").unwrap();
    pattern = re5.replace_all(&pattern, "-").to_string();
    pattern
}

fn extract_slug_from_branch(branch: &str, pattern: &str) -> String {
    if branch.is_empty() {
        return String::new();
    }
    let re = if pattern.is_empty() {
        regex::Regex::new(r"(?:feat|fix|chore|refactor|docs)/(?:[^/]+/)?(.+)").ok()
    } else {
        regex::Regex::new(pattern).ok()
    };
    if let Some(re) = re {
        if let Some(caps) = re.captures(branch) {
            if let Some(m) = caps.get(1) {
                return sanitize_slug(m.as_str());
            }
        }
    }
    String::new()
}

pub fn sanitize_slug(slug: &str) -> String {
    let re_invalid = regex::Regex::new(r#"[<>:"\\|?*\x00-\x1f\x7f]"#).unwrap();
    let s = re_invalid.replace_all(slug, "").to_string();
    let re_non_alnum = regex::Regex::new(r"[^a-zA-Z0-9\-]").unwrap();
    let s = re_non_alnum.replace_all(&s, "-").to_string();
    let re_multi = regex::Regex::new(r"-+").unwrap();
    let s = re_multi.replace_all(&s, "-").to_string();
    let s = s.trim_matches('-').to_string();
    if s.len() > 100 {
        s[..100].to_string()
    } else {
        s
    }
}

fn extract_issue_from_branch(branch: &str) -> String {
    if branch.is_empty() {
        return String::new();
    }
    let patterns = [
        r"(?i)(?:issue|gh|fix|feat|bug)[/\-]?(\d+)",
        r"[/\-](\d+)[/\-]",
        r"#(\d+)",
    ];
    for pat in &patterns {
        if let Ok(re) = regex::Regex::new(pat) {
            if let Some(caps) = re.captures(branch) {
                if let Some(m) = caps.get(1) {
                    return m.as_str().to_string();
                }
            }
        }
    }
    String::new()
}

pub fn extract_task_list_id(resolved: &PlanResolution) -> String {
    if resolved.resolved_by != "session" || resolved.path.is_empty() {
        return String::new();
    }
    std::path::Path::new(&resolved.path)
        .file_name()
        .map(|n| n.to_string_lossy().to_string())
        .unwrap_or_default()
}

// ── Git helpers ─────────────────────────────────────────────────────────────

/// Try running a command via a specific shell. Returns Some(output) on success.
fn try_shell(shell: &str, flag: &str, cmd: &str, cwd: &str) -> Option<String> {
    use std::process::Command;
    let mut c = Command::new(shell);
    c.arg(flag).arg(cmd);
    if !cwd.is_empty() {
        c.current_dir(cwd);
    }
    let output = c.output().ok()?;
    if output.status.success() {
        Some(String::from_utf8_lossy(&output.stdout).trim().to_string())
    } else {
        None
    }
}

pub fn exec_safe(cmd: &str, cwd: &str, timeout_ms: u64) -> String {
    let _timeout = if timeout_ms == 0 { 5000 } else { timeout_ms };

    // Try `sh` first — works on Linux/macOS and Windows with Git Bash in PATH
    if let Some(output) = try_shell("sh", "-c", cmd, cwd) {
        return output;
    }

    // Windows fallback: cmd.exe (strip POSIX shell redirections)
    if cfg!(target_os = "windows") {
        let clean_cmd = cmd.replace(" 2>/dev/null", "");
        if let Some(output) = try_shell("cmd", "/C", &clean_cmd, cwd) {
            return output;
        }
    }

    String::new()
}

pub fn get_git_branch() -> String {
    exec_safe("git branch --show-current", "", 5000)
}

pub fn get_git_root() -> String {
    exec_safe("git rev-parse --show-toplevel", "", 5000)
}

pub fn get_git_remote_url() -> String {
    exec_safe("git config --get remote.origin.url", "", 5000)
}

// ── Wisdom helpers ──────────────────────────────────────────────────────────

pub fn read_wisdom(plan_path: &str, session_id: &str, max_lines: usize) -> String {
    let path = if !plan_path.is_empty() {
        let p = std::path::PathBuf::from(plan_path).join(".wisdom.md");
        if p.exists() {
            Some(p)
        } else {
            None
        }
    } else {
        None
    };
    let path = path.unwrap_or_else(|| {
        if !session_id.is_empty() {
            std::env::temp_dir().join(format!("sl-wisdom-{}.md", session_id))
        } else {
            std::path::PathBuf::new()
        }
    });
    if path.as_os_str().is_empty() {
        return String::new();
    }
    let data = match std::fs::read_to_string(&path) {
        Ok(d) => d,
        Err(_) => return String::new(),
    };
    let trimmed = data.trim().to_string();
    if trimmed.is_empty() {
        return String::new();
    }
    let lines: Vec<&str> = trimmed.lines().collect();
    let start = if lines.len() > max_lines {
        lines.len() - max_lines
    } else {
        0
    };
    lines[start..].join("\n")
}

// ── Compaction context ──────────────────────────────────────────────────────

pub fn build_compaction_context(plan_path: &str, session_id: &str) -> String {
    if plan_path.is_empty() {
        return String::new();
    }
    let mut sections: Vec<String> = Vec::new();

    // Phase completion status
    let phase_status = read_phase_status(plan_path);
    if !phase_status.is_empty() {
        sections.push(format!("Plan Progress:\n{}", phase_status));
    }

    // Accumulated wisdom
    let wisdom = read_wisdom(plan_path, session_id, 10);
    if !wisdom.is_empty() {
        sections.push(format!("Accumulated Learnings:\n{}", wisdom));
    }

    if sections.is_empty() {
        return String::new();
    }

    format!(
        "POST-COMPACTION RECOVERY CONTEXT:\n{}\n\nUse this context to resume work without re-reading completed phases.",
        sections.join("\n\n")
    )
}

fn read_phase_status(plan_path: &str) -> String {
    let pattern = format!("{}/phase-*.md", plan_path);
    let mut phase_files: Vec<_> = glob::glob(&pattern)
        .map(|paths| paths.flatten().collect())
        .unwrap_or_default();
    if phase_files.is_empty() {
        return String::new();
    }
    phase_files.sort();

    let mut lines = Vec::new();
    for f in phase_files {
        let name = f
            .file_name()
            .map(|n| n.to_string_lossy().to_string())
            .unwrap_or_default();
        let (total, done) = count_todos(&f);
        if total == 0 {
            lines.push(format!("  - {}: no todos", name));
        } else {
            let status = if done == total {
                "complete"
            } else {
                "in-progress"
            };
            lines.push(format!(
                "  - {}: {}/{} todos ({})",
                name, done, total, status
            ));
        }
    }
    lines.join("\n")
}

fn count_todos(path: &std::path::Path) -> (usize, usize) {
    let data = match std::fs::read_to_string(path) {
        Ok(d) => d,
        Err(_) => return (0, 0),
    };
    let mut total = 0;
    let mut done = 0;
    for line in data.lines() {
        let t = line.trim();
        if t.starts_with("- [ ]") {
            total += 1;
        } else if t.starts_with("- [x]") || t.starts_with("- [X]") {
            total += 1;
            done += 1;
        }
    }
    (total, done)
}

// ── Write env var to file ────────────────────────────────────────────────────

pub fn write_env(env_file: &str, key: &str, value: &str) {
    if env_file.is_empty() || value.is_empty() {
        return;
    }
    let escaped = escape_shell_value(value);
    let line = format!("export {}=\"{}\"\n", key, escaped);
    use std::io::Write;
    if let Ok(mut f) = std::fs::OpenOptions::new()
        .append(true)
        .create(true)
        .open(env_file)
    {
        let _ = f.write_all(line.as_bytes());
    }
}

pub fn write_env_forced(env_file: &str, key: &str, value: &str) {
    if env_file.is_empty() {
        return;
    }
    let escaped = escape_shell_value(value);
    let line = format!("export {}=\"{}\"\n", key, escaped);
    use std::io::Write;
    if let Ok(mut f) = std::fs::OpenOptions::new()
        .append(true)
        .create(true)
        .open(env_file)
    {
        let _ = f.write_all(line.as_bytes());
    }
}

pub fn escape_shell_value(s: &str) -> String {
    s.replace('\\', "\\\\")
        .replace('"', "\\\"")
        .replace('$', "\\$")
        .replace('`', "\\`")
}

// ── Skills venv detection ────────────────────────────────────────────────────

pub fn resolve_skills_venv() -> String {
    let cwd = std::env::current_dir().unwrap_or_default();
    let home = dirs_home();
    let local_venv = cwd
        .join(".claude")
        .join("skills")
        .join(".venv")
        .join("bin")
        .join("python3");
    let global_venv = home
        .join(".claude")
        .join("skills")
        .join(".venv")
        .join("bin")
        .join("python3");
    if local_venv.exists() {
        return ".claude/skills/.venv/bin/python3".to_string();
    }
    if global_venv.exists() {
        return "~/.claude/skills/.venv/bin/python3".to_string();
    }
    String::new()
}

// ── Context cache ────────────────────────────────────────────────────────────

pub fn read_context_percent(session_id: &str) -> Option<i64> {
    if session_id.is_empty() {
        return None;
    }
    let path = std::env::temp_dir().join(format!("sl-context-{}.json", session_id));
    let data = std::fs::read_to_string(&path).ok()?;
    let v: Value = serde_json::from_str(&data).ok()?;
    let percent = v.get("percent")?.as_i64()?;
    let ts = v.get("timestamp")?.as_i64()?;
    let now_ms = chrono::Utc::now().timestamp_millis();
    if now_ms - ts > 300_000 {
        return None;
    } // 5 min stale
    Some(percent)
}

// ── Truncation helpers ───────────────────────────────────────────────────────

pub struct ToolBudget {
    pub max_lines: usize,
    pub head_lines: usize,
    pub tail_lines: usize,
}

pub fn budget_for_tool(tool_name: &str) -> ToolBudget {
    match tool_name {
        "Bash" => ToolBudget {
            max_lines: 500,
            head_lines: 80,
            tail_lines: 50,
        },
        "Grep" => ToolBudget {
            max_lines: 200,
            head_lines: 40,
            tail_lines: 20,
        },
        "Read" => ToolBudget {
            max_lines: 300,
            head_lines: 60,
            tail_lines: 30,
        },
        "Glob" => ToolBudget {
            max_lines: 150,
            head_lines: 30,
            tail_lines: 20,
        },
        _ => ToolBudget {
            max_lines: 200,
            head_lines: 50,
            tail_lines: 30,
        },
    }
}

pub fn truncate_output(
    output: &str,
    max_lines: usize,
    head_lines: usize,
    tail_lines: usize,
) -> (String, bool) {
    if output.is_empty() {
        return (output.to_string(), false);
    }
    let lines: Vec<&str> = output.split('\n').collect();
    if lines.len() <= max_lines {
        return (output.to_string(), false);
    }
    let dropped = lines.len().saturating_sub(head_lines + tail_lines);
    if dropped == 0 {
        return (output.to_string(), false);
    }
    let head = lines[..head_lines].join("\n");
    let tail = lines[lines.len() - tail_lines..].join("\n");
    let marker = format!(
        "\n\n... [{} lines truncated for context efficiency] ...\n\n",
        dropped
    );
    (format!("{}{}{}", head, marker, tail), true)
}

// ── Privacy helpers ──────────────────────────────────────────────────────────

const APPROVAL_PREFIX: &str = "APPROVED:";

pub fn is_safe_file(path: &str) -> bool {
    let base = std::path::Path::new(path)
        .file_name()
        .map(|n| n.to_string_lossy().to_lowercase())
        .unwrap_or_default();
    base.ends_with(".example") || base.ends_with(".sample") || base.ends_with(".template")
}

pub fn has_approval_prefix(path: &str) -> bool {
    path.starts_with(APPROVAL_PREFIX)
}

pub fn strip_approval_prefix(path: &str) -> &str {
    if has_approval_prefix(path) {
        &path[APPROVAL_PREFIX.len()..]
    } else {
        path
    }
}

pub fn is_privacy_sensitive(path: &str) -> bool {
    if path.is_empty() {
        return false;
    }
    let clean = strip_approval_prefix(path);
    let normalized = clean.replace('\\', "/");
    // URL decode best-effort
    let normalized = url_decode_simple(&normalized);
    if is_safe_file(&normalized) {
        return false;
    }
    let base = std::path::Path::new(&normalized)
        .file_name()
        .map(|n| n.to_string_lossy().to_string())
        .unwrap_or_default();
    // Match patterns
    let patterns: &[&str] = &[
        r"^\.env$",
        r"^\.env\.",
        r"\.env$",
        r"/\.env\.",
        r"(?i)credentials",
        r"(?i)secrets?\.ya?ml$",
        r"\.pem$",
        r"\.key$",
        r"id_rsa",
        r"id_ed25519",
    ];
    for pat in patterns {
        if let Ok(re) = regex::Regex::new(pat) {
            if re.is_match(&base) || re.is_match(&normalized) {
                return true;
            }
        }
    }
    false
}

fn url_decode_simple(s: &str) -> String {
    let mut result = String::with_capacity(s.len());
    let bytes = s.as_bytes();
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'%' && i + 2 < bytes.len() {
            if let Ok(hex) = std::str::from_utf8(&bytes[i + 1..i + 3]) {
                if let Ok(v) = u8::from_str_radix(hex, 16) {
                    result.push(v as char);
                    i += 3;
                    continue;
                }
            }
        }
        result.push(bytes[i] as char);
        i += 1;
    }
    result
}

pub fn build_privacy_prompt_data(file_path: &str) -> serde_json::Value {
    let base = std::path::Path::new(file_path)
        .file_name()
        .map(|n| n.to_string_lossy().to_string())
        .unwrap_or_else(|| file_path.to_string());
    serde_json::json!({
        "type": "PRIVACY_PROMPT",
        "file": file_path,
        "basename": base,
        "question": {
            "header": "File Access",
            "text": format!("I need to read {:?} which may contain sensitive data (API keys, passwords, tokens). Do you approve?", base),
            "options": [
                {"label": "Yes, approve access", "description": format!("Allow reading {} this time", base)},
                {"label": "No, skip this file", "description": "Continue without accessing this file"},
            ]
        }
    })
}

// ── Scout helpers ────────────────────────────────────────────────────────────

const DEFAULT_IGNORE_PATTERNS: &[&str] = &[
    "node_modules",
    "dist",
    "build",
    ".next",
    ".nuxt",
    "__pycache__",
    ".venv",
    "venv",
    "vendor",
    "target",
    ".git",
    "coverage",
];

pub fn load_slignore_patterns(slignore_path: &str) -> Vec<String> {
    if slignore_path.is_empty() {
        return DEFAULT_IGNORE_PATTERNS
            .iter()
            .map(|s| s.to_string())
            .collect();
    }
    let data = match std::fs::read_to_string(slignore_path) {
        Ok(d) => d,
        Err(_) => {
            return DEFAULT_IGNORE_PATTERNS
                .iter()
                .map(|s| s.to_string())
                .collect()
        }
    };
    let patterns: Vec<String> = data
        .lines()
        .map(|l| l.trim().to_string())
        .filter(|l| !l.is_empty() && !l.starts_with('#'))
        .collect();
    if patterns.is_empty() {
        DEFAULT_IGNORE_PATTERNS
            .iter()
            .map(|s| s.to_string())
            .collect()
    } else {
        patterns
    }
}

pub fn path_matches_patterns(test_path: &str, patterns: &[String]) -> Option<String> {
    if test_path.is_empty() {
        return None;
    }
    let normalized = test_path
        .replace('\\', "/")
        .trim_start_matches("./")
        .trim_start_matches('/')
        .to_string();
    let normalized = {
        let mut s = normalized.as_str();
        while s.starts_with("../") {
            s = &s[3..];
        }
        s.to_string()
    };
    if normalized.is_empty() {
        return None;
    }

    // Build globset from patterns
    let mut builder = globset::GlobSetBuilder::new();
    for p in patterns {
        if p.starts_with('!') {
            continue;
        } // skip negations for block check
        let pats_to_add: Vec<String> = if p.contains('/') || p.contains('*') {
            vec![p.clone()]
        } else {
            vec![
                format!("**/{}", p),
                format!("**/{p}/**"),
                p.clone(),
                format!("{p}/**"),
            ]
        };
        for pat in pats_to_add {
            if let Ok(g) = globset::Glob::new(&pat) {
                builder.add(g);
            }
        }
    }
    if let Ok(gs) = builder.build() {
        if gs.is_match(&normalized) {
            // Find which original pattern matched
            return Some(find_matching_pattern(&normalized, patterns));
        }
    }
    None
}

fn find_matching_pattern(path: &str, patterns: &[String]) -> String {
    for p in patterns {
        if p.starts_with('!') {
            continue;
        }
        let simplified = p.replace("**", "").replace('*', "");
        if !simplified.is_empty() && path.contains(&simplified) {
            return p.clone();
        }
    }
    patterns
        .iter()
        .find(|p| !p.starts_with('!'))
        .cloned()
        .unwrap_or_else(|| "unknown".to_string())
}

pub fn is_broad_pattern(pattern: &str) -> bool {
    if pattern.is_empty() {
        return false;
    }
    let normalized = pattern.trim();
    let broad_pats = [
        r"^\*\*$",
        r"^\*$",
        r"^\*\*/\*$",
        r"^\*\*/\.\*$",
        r"^\*\.\w+$",
        r"^\*\.\{[^}]+\}$",
        r"^\*\*/\*\.\w+$",
        r"^\*\*/\*\.\{[^}]+\}$",
    ];
    for pat in &broad_pats {
        if let Ok(re) = regex::Regex::new(pat) {
            if re.is_match(normalized) {
                return true;
            }
        }
    }
    false
}

pub fn has_specific_directory(pattern: &str) -> bool {
    if pattern.is_empty() {
        return false;
    }
    let specific_dirs = [
        "src",
        "lib",
        "app",
        "apps",
        "packages",
        "components",
        "pages",
        "api",
        "server",
        "client",
        "web",
        "mobile",
        "shared",
        "common",
        "utils",
        "helpers",
        "services",
        "hooks",
        "store",
        "routes",
        "models",
        "controllers",
        "views",
        "tests",
        "__tests__",
        "spec",
    ];
    for dir in &specific_dirs {
        if pattern.starts_with(&format!("{}/", dir)) || pattern.starts_with(&format!("./{}/", dir))
        {
            return true;
        }
    }
    // Non-wildcard first segment counts as specific
    let first_seg = pattern.split('/').next().unwrap_or("");
    !first_seg.is_empty() && !first_seg.contains('*') && first_seg != "."
}

pub fn is_high_level_path(base_path: &str) -> bool {
    if base_path.is_empty() {
        return true;
    }
    let normalized = base_path.replace('\\', "/");
    let high_risk = [r"/worktrees/[^/]+/?$", r"^\.?/?$", r"^[^/]+/?$"];
    for pat in &high_risk {
        if let Ok(re) = regex::Regex::new(pat) {
            if re.is_match(&normalized) {
                return true;
            }
        }
    }
    let segments: Vec<&str> = normalized
        .split('/')
        .filter(|s| !s.is_empty() && *s != ".")
        .collect();
    if segments.len() <= 1 {
        return true;
    }
    let specific_dirs = [
        "src",
        "lib",
        "app",
        "apps",
        "packages",
        "components",
        "pages",
        "api",
        "server",
        "client",
        "web",
        "mobile",
        "shared",
        "common",
    ];
    for dir in &specific_dirs {
        if normalized.contains(&format!("/{}/", dir))
            || normalized.contains(&format!("/{}", dir))
            || normalized.starts_with(&format!("{}/", dir))
            || normalized == *dir
        {
            return false;
        }
    }
    true
}

pub fn suggest_specific_patterns(pattern: &str) -> Vec<String> {
    let mut suggestions = Vec::new();
    let ext_re = regex::Regex::new(r"\*\.(\{[^}]+\}|\w+)$").unwrap();
    let ext = ext_re
        .captures(pattern)
        .and_then(|c| c.get(1))
        .map(|m| m.as_str().to_string())
        .unwrap_or_default();
    if pattern.contains(".ts") || pattern.contains("{ts") {
        suggestions.push("src/**/*.ts".to_string());
        suggestions.push("src/**/*.tsx".to_string());
    }
    if pattern.contains(".js") || pattern.contains("{js") {
        suggestions.push("src/**/*.js".to_string());
        suggestions.push("lib/**/*.js".to_string());
    }
    for dir in &["src", "lib", "app", "components"] {
        if !ext.is_empty() {
            suggestions.push(format!("{}/**/*.{}", dir, ext));
        } else {
            suggestions.push(format!("{}/**/*", dir));
        }
    }
    suggestions.truncate(4);
    suggestions
}

// ── Intent classifier ─────────────────────────────────────────────────────────

pub struct IntentCategory {
    pub name: &'static str,
    pub keywords: &'static [&'static str],
    pub strategy: &'static str,
}

pub static INTENT_CATEGORIES: &[IntentCategory] = &[
    IntentCategory {
        name: "DEBUG",
        keywords: &[
            "fix",
            "bug",
            "error",
            "broken",
            "failing",
            "crash",
            "issue",
            "wrong",
            "not working",
        ],
        strategy: "Reproduce first. Check logs/traces. Isolate root cause before fixing.",
    },
    IntentCategory {
        name: "TEST",
        keywords: &["test", "coverage", "spec", "assert", "verify", "validate"],
        strategy: "Write focused tests. Cover edge cases. Don't mock internals.",
    },
    IntentCategory {
        name: "DEPLOY",
        keywords: &[
            "deploy", "push", "release", "publish", "ship", "ci/cd", "pipeline",
        ],
        strategy: "Verify tests pass. Check CI config. Confirm with user before pushing.",
    },
    IntentCategory {
        name: "REFACTOR",
        keywords: &[
            "refactor",
            "clean up",
            "simplify",
            "reorganize",
            "extract",
            "restructure",
        ],
        strategy: "Preserve behavior. Run tests before and after. Small incremental changes.",
    },
    IntentCategory {
        name: "EXPLAIN",
        keywords: &[
            "explain",
            "how does",
            "what is",
            "why does",
            "describe",
            "walk through",
            "tell me about",
        ],
        strategy: "Use /preview for visuals. Tailor depth to user's coding level.",
    },
    IntentCategory {
        name: "RESEARCH",
        keywords: &[
            "explore",
            "investigate",
            "analyze",
            "compare",
            "evaluate",
            "look into",
            "research",
        ],
        strategy:
            "Gather info before proposing changes. Read docs, search code, compare approaches.",
    },
    IntentCategory {
        name: "IMPLEMENT",
        keywords: &[
            "build",
            "create",
            "add",
            "write",
            "implement",
            "develop",
            "set up",
            "feature",
            "make",
        ],
        strategy: "Follow active plan. Write tests. Run compile after edits.",
    },
];

pub fn classify_intent(prompt: &str) -> Option<&'static IntentCategory> {
    let lower = prompt.to_lowercase();
    for cat in INTENT_CATEGORIES {
        for kw in cat.keywords {
            if lower.contains(kw) {
                return Some(cat);
            }
        }
    }
    None
}

// ── Colors ───────────────────────────────────────────────────────────────────

pub fn supports_color() -> bool {
    if std::env::var("NO_COLOR").is_ok() {
        return false;
    }
    if std::env::var("FORCE_COLOR").is_ok() {
        return true;
    }
    true
}

pub fn colorize(text: &str, code: &str) -> String {
    if !supports_color() {
        return text.to_string();
    }
    format!("{}{}\x1b[0m", code, text)
}

pub fn colored_bar(percent: i64, width: usize) -> String {
    let clamped = percent.clamp(0, 100) as usize;
    let width = if width == 0 { 12 } else { width };
    let mut filled = (clamped * width) / 100;
    if (clamped * width) % 100 >= 50 {
        filled += 1;
    }
    filled = filled.min(width);
    let empty = width - filled;
    let filled_str = "▰".repeat(filled);
    let empty_str = "▱".repeat(empty);
    if !supports_color() {
        return format!("{}{}", filled_str, empty_str);
    }
    let color = if percent >= 85 {
        "\x1b[31m"
    } else if percent >= 70 {
        "\x1b[33m"
    } else {
        "\x1b[32m"
    };
    format!("{}{}\x1b[2m{}\x1b[0m", color, filled_str, empty_str)
}

pub fn collapse_home(path: &str) -> String {
    let home = dirs_home();
    let home_str = home.to_string_lossy();
    if path.starts_with(home_str.as_ref()) {
        format!("~{}", &path[home_str.len()..])
    } else {
        path.to_string()
    }
}

pub fn local_tz() -> String {
    if let Ok(link) = std::fs::read_link("/etc/localtime") {
        let s = link.to_string_lossy().to_string();
        const PREFIX: &str = "/usr/share/zoneinfo/";
        if let Some(idx) = s.find(PREFIX) {
            return s[idx + PREFIX.len()..].to_string();
        }
    }
    if let Ok(data) = std::fs::read_to_string("/etc/timezone") {
        return data.trim().to_string();
    }
    std::env::var("TZ").unwrap_or_else(|_| "UTC".to_string())
}

// ── Project detection ─────────────────────────────────────────────────────────

pub fn detect_project_type(override_val: &str) -> String {
    if !override_val.is_empty() && override_val != "auto" {
        return override_val.to_string();
    }
    if std::path::Path::new("pnpm-workspace.yaml").exists()
        || std::path::Path::new("lerna.json").exists()
    {
        return "monorepo".to_string();
    }
    if std::path::Path::new("package.json").exists() {
        if let Ok(data) = std::fs::read_to_string("package.json") {
            if let Ok(pkg) = serde_json::from_str::<Value>(&data) {
                if let Some(obj) = pkg.as_object() {
                    if obj.contains_key("workspaces") {
                        return "monorepo".to_string();
                    }
                    if obj.contains_key("main") || obj.contains_key("exports") {
                        return "library".to_string();
                    }
                }
            }
        }
    }
    "single-repo".to_string()
}

pub fn detect_package_manager(override_val: &str) -> String {
    if !override_val.is_empty() && override_val != "auto" {
        return override_val.to_string();
    }
    if std::path::Path::new("bun.lockb").exists() {
        return "bun".to_string();
    }
    if std::path::Path::new("pnpm-lock.yaml").exists() {
        return "pnpm".to_string();
    }
    if std::path::Path::new("yarn.lock").exists() {
        return "yarn".to_string();
    }
    if std::path::Path::new("package-lock.json").exists() {
        return "npm".to_string();
    }
    String::new()
}

pub fn get_coding_level_style_name(level: i32) -> String {
    match level {
        0 => "coding-level-0-eli5".to_string(),
        1 => "coding-level-1-junior".to_string(),
        2 => "coding-level-2-mid".to_string(),
        3 => "coding-level-3-senior".to_string(),
        4 => "coding-level-4-lead".to_string(),
        _ => "coding-level-5-god".to_string(),
    }
}

pub fn get_coding_level_guidelines(level: i32) -> String {
    if level == -1 {
        return String::new();
    }
    let style_name = get_coding_level_style_name(level);
    let cwd = std::env::current_dir().unwrap_or_default();
    let path = cwd
        .join(".claude")
        .join("output-styles")
        .join(format!("{}.md", style_name));
    let data = match std::fs::read_to_string(&path) {
        Ok(d) => d,
        Err(_) => return String::new(),
    };
    // Strip YAML frontmatter
    let re = regex::Regex::new(r"(?s)^---.*?---\n*").unwrap();
    re.replace(&data, "").trim().to_string()
}

/// Find sc binary in ~/.solon/bin/sc or PATH.
pub fn find_sc_binary() -> Option<String> {
    let home = dirs_home();
    let candidates = vec![
        home.join(".solon")
            .join("bin")
            .join("sc")
            .to_string_lossy()
            .to_string(),
        "sc".to_string(),
    ];
    candidates.into_iter().find(|c| {
        std::process::Command::new(c)
            .arg("--version")
            .output()
            .is_ok()
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sanitize_slug() {
        assert_eq!(sanitize_slug("hello world"), "hello-world");
        assert_eq!(sanitize_slug("feat/my-feature"), "feat-my-feature");
        assert_eq!(sanitize_slug("---hello---"), "hello");
    }

    #[test]
    fn test_is_privacy_sensitive() {
        assert!(is_privacy_sensitive(".env"));
        assert!(is_privacy_sensitive(".env.local"));
        assert!(is_privacy_sensitive("secrets.yaml"));
        assert!(is_privacy_sensitive("id_rsa"));
        assert!(!is_privacy_sensitive(".env.example"));
        assert!(!is_privacy_sensitive("README.md"));
    }

    #[test]
    fn test_has_approval_prefix() {
        assert!(has_approval_prefix("APPROVED:/path/.env"));
        assert!(!has_approval_prefix("/path/.env"));
    }

    #[test]
    fn test_is_broad_pattern() {
        assert!(is_broad_pattern("**"));
        assert!(is_broad_pattern("*"));
        assert!(is_broad_pattern("**/*"));
        assert!(is_broad_pattern("*.ts"));
        assert!(!is_broad_pattern("src/**/*.ts"));
    }

    #[test]
    fn test_has_specific_directory() {
        assert!(has_specific_directory("src/**/*.ts"));
        assert!(has_specific_directory("lib/utils.rs"));
        assert!(!has_specific_directory("*.ts"));
    }

    #[test]
    fn test_classify_intent() {
        assert_eq!(
            classify_intent("fix the bug").map(|c| c.name),
            Some("DEBUG")
        );
        assert_eq!(
            classify_intent("write tests for it").map(|c| c.name),
            Some("TEST")
        );
        assert_eq!(
            classify_intent("implement the feature").map(|c| c.name),
            Some("IMPLEMENT")
        );
        assert_eq!(
            classify_intent("hello world how are you").map(|c| c.name),
            None
        );
    }

    #[test]
    fn test_truncate_output() {
        let lines: Vec<String> = (0..250).map(|i| format!("line {}", i)).collect();
        let text = lines.join("\n");
        let (result, changed) = truncate_output(&text, 200, 50, 30);
        assert!(changed);
        assert!(result.contains("truncated"));
    }

    #[test]
    fn test_format_date_tokens() {
        let result = format_date("YYMMDD-HHmm");
        // Should be 11 chars: YYMMDD-HHmm
        assert_eq!(result.len(), 11);
        assert!(result.contains('-'));
    }

    #[test]
    fn test_try_shell_echo() {
        let result = try_shell("sh", "-c", "echo hello", "");
        assert_eq!(result, Some("hello".to_string()));
    }

    #[test]
    fn test_try_shell_bad_shell() {
        let result = try_shell("nonexistent_shell_xyz", "-c", "echo hi", "");
        assert_eq!(result, None);
    }

    #[test]
    fn test_exec_safe_basic() {
        let result = exec_safe("echo test123", "", 5000);
        assert_eq!(result, "test123");
    }

    #[test]
    fn test_exec_safe_bad_command() {
        let result = exec_safe("false", "", 5000);
        assert_eq!(result, "");
    }
}
