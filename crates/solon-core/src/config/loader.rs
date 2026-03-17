/// Config loading with cascade: DEFAULT → global (~/.claude/.sl.json) → local (.claude/.sl.json)
/// Ported from Go solon-core/internal/config/loader.go
use serde_json::Value;
use solon_common::SLConfig;
use std::path::{Path, PathBuf};

const LOCAL_CONFIG_PATH: &str = ".claude/.sl.json";

/// Returns ~/.claude/.sl.json path
pub fn global_config_path() -> PathBuf {
    let home = dirs_home();
    PathBuf::from(home).join(".claude").join(".sl.json")
}

fn dirs_home() -> String {
    std::env::var("HOME").unwrap_or_else(|_| "/tmp".to_string())
}

/// Returns the factory default configuration.
pub fn default_config() -> SLConfig {
    use solon_common::*;
    use std::collections::HashMap;

    let mut hooks = HashMap::new();
    for key in &[
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
        hooks.insert(key.to_string(), true);
    }

    SLConfig {
        plan: PlanConfig::default(),
        paths: PathsConfig::default(),
        docs: DocsConfig { max_loc: 800 },
        locale: LocaleConfig::default(),
        trust: TrustConfig::default(),
        project: ProjectConfig::default(),
        skills: SkillsConfig::default(),
        hooks,
        assertions: vec![],
        coding_level: -1,
        statusline: "full".to_string(),
        subagent: None,
    }
}

/// Read and parse a JSON config file. Returns None if missing or unparseable.
pub fn load_config_from_path(path: &Path) -> Option<Value> {
    let data = std::fs::read_to_string(path).ok()?;
    serde_json::from_str(&data).ok()
}

/// Deep merge: source values override target. Empty objects {} are skipped.
/// Arrays are replaced entirely.
pub fn deep_merge_values(target: Value, source: Value) -> Value {
    match (target, source) {
        (Value::Object(mut t), Value::Object(s)) => {
            if s.is_empty() {
                return Value::Object(t);
            }
            for (k, sv) in s {
                if sv.is_null() {
                    continue;
                }
                let tv = t.remove(&k).unwrap_or(Value::Null);
                t.insert(k, deep_merge_values(tv, sv));
            }
            Value::Object(t)
        }
        // Arrays replaced entirely
        (_, s @ Value::Array(_)) => s,
        (_, s) => s,
    }
}

/// Sanitize a path value — prevents traversal.
pub fn sanitize_path(path_value: &str, project_root: &str) -> Option<String> {
    let normalized = normalize_path(path_value);
    if normalized.is_empty() {
        return None;
    }
    if normalized.contains('\0') {
        return None;
    }
    if Path::new(&normalized).is_absolute() {
        return Some(normalized);
    }
    let resolved = Path::new(project_root).join(&normalized);
    let prefix = format!("{}/", project_root);
    let resolved_str = resolved.to_string_lossy().to_string();
    if !resolved_str.starts_with(&prefix) && resolved_str != project_root {
        return None;
    }
    Some(normalized)
}

/// Trim whitespace and trailing slashes.
pub fn normalize_path(p: &str) -> String {
    let t = p.trim();
    t.trim_end_matches('/').trim_end_matches('\\').to_string()
}

/// Load config with cascade: DEFAULT → global → local.
pub fn load_config() -> SLConfig {
    let project_root = std::env::current_dir()
        .unwrap_or_default()
        .to_string_lossy()
        .to_string();

    let global_raw = load_config_from_path(&global_config_path());
    let local_path = Path::new(&project_root).join(LOCAL_CONFIG_PATH);
    let local_raw = load_config_from_path(&local_path);

    if global_raw.is_none() && local_raw.is_none() {
        return default_config();
    }

    let default = default_config();
    let mut merged = serde_json::to_value(&default).unwrap_or(Value::Object(Default::default()));

    if let Some(global) = global_raw {
        merged = deep_merge_values(merged, global);
    }
    if let Some(local) = local_raw {
        merged = deep_merge_values(merged, local);
    }

    let mut cfg: SLConfig = serde_json::from_value(merged).unwrap_or_else(|_| default_config());

    if cfg.hooks.is_empty() {
        cfg.hooks = default_config().hooks;
    }
    if cfg.statusline.is_empty() {
        cfg.statusline = "full".to_string();
    }

    sanitize_config(cfg, &project_root)
}

/// Validate config paths against the project root.
pub fn sanitize_config(mut cfg: SLConfig, project_root: &str) -> SLConfig {
    let default = default_config();

    if sanitize_path(&cfg.plan.reports_dir, project_root).is_none() {
        cfg.plan.reports_dir = default.plan.reports_dir.clone();
    }
    if cfg.plan.resolution.order.is_empty() {
        cfg.plan.resolution.order = default.plan.resolution.order.clone();
    }
    if cfg.plan.resolution.branch_pattern.is_empty() {
        cfg.plan.resolution.branch_pattern = default.plan.resolution.branch_pattern.clone();
    }
    if sanitize_path(&cfg.paths.docs, project_root).is_none() {
        cfg.paths.docs = default.paths.docs.clone();
    }
    if sanitize_path(&cfg.paths.plans, project_root).is_none() {
        cfg.paths.plans = default.paths.plans.clone();
    }
    cfg
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_normalize_path() {
        // Trailing slashes are stripped (matching Go behaviour)
        assert_eq!(normalize_path("  docs/  "), "docs");
        assert_eq!(normalize_path("plans/"), "plans");
        assert_eq!(normalize_path(""), "");
    }

    #[test]
    fn test_deep_merge_empty_object_skipped() {
        let t = serde_json::json!({"a": 1, "b": 2});
        let s = serde_json::json!({});
        let result = deep_merge_values(t.clone(), s);
        assert_eq!(result, t);
    }

    #[test]
    fn test_deep_merge_array_replaced() {
        let t = serde_json::json!({"arr": [1, 2, 3]});
        let s = serde_json::json!({"arr": [4, 5]});
        let result = deep_merge_values(t, s);
        assert_eq!(result["arr"], serde_json::json!([4, 5]));
    }

    #[test]
    fn test_default_config_has_hooks() {
        let cfg = default_config();
        assert!(cfg.hooks.contains_key("session-init"));
        assert_eq!(cfg.statusline, "full");
    }
}
