use crate::hooks::{get_str, is_hook_enabled};
/// PostToolUse(Edit/Write/MultiEdit): track edits, remind to run code-simplifier after threshold.
use anyhow::Result;
use serde_json::Value;

const SIMPLIFY_SESSION_FILE: &str = "sl-simplify-session.json";
const EDIT_THRESHOLD: usize = 5;
const SESSION_TTL_MS: i64 = 2 * 60 * 60 * 1000; // 2 hours
const REMINDER_COOLDOWN_MS: i64 = 10 * 60 * 1000; // 10 minutes

#[derive(serde::Serialize, serde::Deserialize)]
struct SimplifySession {
    start_time: i64,
    edit_count: usize,
    modified_files: Vec<String>,
    last_reminder: i64,
    simplifier_run: bool,
}

impl SimplifySession {
    fn new() -> Self {
        Self {
            start_time: chrono::Utc::now().timestamp_millis(),
            edit_count: 0,
            modified_files: Vec::new(),
            last_reminder: 0,
            simplifier_run: false,
        }
    }
}

pub fn run() -> Result<()> {
    if !is_hook_enabled("post-edit-simplify-reminder") {
        crate::hooks::write_output(&serde_json::json!({"continue": true}));
        return Ok(());
    }

    let input = match crate::hooks::read_hook_input() {
        Ok(v) => v,
        Err(_) => {
            crate::hooks::write_output(&serde_json::json!({"continue": true}));
            return Ok(());
        }
    };

    let tool_name = get_str(&input, "tool_name");
    if !matches!(tool_name, "Edit" | "Write" | "MultiEdit") {
        crate::hooks::write_output(&serde_json::json!({"continue": true}));
        return Ok(());
    }

    // Invalidate git cache for the cwd
    let cwd = {
        let c = get_str(&input, "cwd").to_string();
        if c.is_empty() {
            std::env::current_dir()
                .unwrap_or_default()
                .to_string_lossy()
                .to_string()
        } else {
            c
        }
    };
    invalidate_git_cache(&cwd);

    let mut session = load_session();
    session.edit_count += 1;

    // Extract file path
    let file_path = input
        .get("tool_input")
        .and_then(|ti| ti.get("file_path").or_else(|| ti.get("path")))
        .and_then(|v| v.as_str())
        .filter(|s| !s.is_empty())
        .map(|s| s.to_string());

    if let Some(fp) = file_path {
        if !session.modified_files.contains(&fp) {
            session.modified_files.push(fp);
        }
    }

    let now = chrono::Utc::now().timestamp_millis();
    let should_remind = session.edit_count >= EDIT_THRESHOLD
        && !session.simplifier_run
        && now - session.last_reminder > REMINDER_COOLDOWN_MS;

    let mut output = serde_json::json!({"continue": true});
    if should_remind {
        session.last_reminder = now;
        output["additionalContext"] = Value::String(format!(
            "\n\n[Code Simplification Reminder] You have modified {} files in this session. \
             Consider using the `code-simplifier` agent to refine recent changes before proceeding to code review. \
             This is a MANDATORY step in the workflow.",
            session.modified_files.len()
        ));
    }

    save_session(&session);
    crate::hooks::write_output(&output);
    Ok(())
}

fn session_path() -> std::path::PathBuf {
    std::env::temp_dir().join(SIMPLIFY_SESSION_FILE)
}

fn load_session() -> SimplifySession {
    let data = match std::fs::read_to_string(session_path()) {
        Ok(d) => d,
        Err(_) => return SimplifySession::new(),
    };
    let s: SimplifySession = match serde_json::from_str(&data) {
        Ok(s) => s,
        Err(_) => return SimplifySession::new(),
    };
    let now = chrono::Utc::now().timestamp_millis();
    if now - s.start_time > SESSION_TTL_MS {
        return SimplifySession::new();
    }
    s
}

fn save_session(s: &SimplifySession) {
    if let Ok(data) = serde_json::to_string_pretty(s) {
        let _ = std::fs::write(session_path(), data);
    }
}

fn invalidate_git_cache(cwd: &str) {
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};
    let mut hasher = DefaultHasher::new();
    cwd.hash(&mut hasher);
    let hash = format!("{:x}", hasher.finish());
    let short = &hash[..8.min(hash.len())];
    let cache = std::env::temp_dir().join(format!("sl-git-cache-{}.json", short));
    let _ = std::fs::remove_file(cache);
}
