use crate::hooks::{get_str, is_hook_enabled, read_context_percent};
/// PostToolUse hook: warn when context window is near capacity, suggesting /compact.
use anyhow::Result;

const WARN_THRESHOLD: i64 = 65;
const STRONG_THRESHOLD: i64 = 75;
const URGENT_THRESHOLD: i64 = 85;
const COOLDOWN_MS: i64 = 3 * 60 * 1000; // 3 minutes
const COOLDOWN_FILE: &str = "sl-compaction-reminded.json";

pub fn run() -> Result<()> {
    if !is_hook_enabled("preemptive-compaction") {
        std::process::exit(0);
    }

    let input = match crate::hooks::read_hook_input() {
        Ok(v) => v,
        Err(_) => {
            crate::hooks::write_output(&serde_json::json!({"continue": true}));
            return Ok(());
        }
    };

    let session_id = {
        let from_input = get_str(&input, "session_id").to_string();
        if from_input.is_empty() {
            std::env::var("SL_SESSION_ID").unwrap_or_default()
        } else {
            from_input
        }
    };

    if session_id.is_empty() {
        crate::hooks::write_output(&serde_json::json!({"continue": true}));
        return Ok(());
    }

    let percent = match read_context_percent(&session_id) {
        Some(p) if p >= WARN_THRESHOLD => p,
        _ => {
            crate::hooks::write_output(&serde_json::json!({"continue": true}));
            return Ok(());
        }
    };

    if !cooldown_expired() {
        crate::hooks::write_output(&serde_json::json!({"continue": true}));
        return Ok(());
    }
    write_cooldown();

    let msg = compaction_message(percent);
    crate::hooks::write_output(&serde_json::json!({
        "continue": true,
        "additionalContext": msg,
    }));
    Ok(())
}

fn cooldown_path() -> std::path::PathBuf {
    std::env::temp_dir().join(COOLDOWN_FILE)
}

fn cooldown_expired() -> bool {
    let data = match std::fs::read_to_string(cooldown_path()) {
        Ok(d) => d,
        Err(_) => return true,
    };
    let v: serde_json::Value = match serde_json::from_str(&data) {
        Ok(v) => v,
        Err(_) => return true,
    };
    let ts = v.get("timestamp").and_then(|t| t.as_i64()).unwrap_or(0);
    chrono::Utc::now().timestamp_millis() - ts > COOLDOWN_MS
}

fn write_cooldown() {
    let entry = serde_json::json!({ "timestamp": chrono::Utc::now().timestamp_millis() });
    let _ = std::fs::write(
        cooldown_path(),
        serde_json::to_string(&entry).unwrap_or_default(),
    );
}

/// Returns escalating urgency message based on context usage percentage.
fn compaction_message(percent: i64) -> String {
    if percent >= URGENT_THRESHOLD {
        format!(
            "\n[URGENT] Context at {}% — STOP current work. Run /compact NOW or session will overflow. \
             Save any critical context to files before compacting.", percent
        )
    } else if percent >= STRONG_THRESHOLD {
        format!(
            "\n[Context Warning] Context at {}%. Finish current atomic task, then run /compact. \
             Avoid spawning new subagents until compacted.",
            percent
        )
    } else {
        format!(
            "\n[Context Notice] Context at {}%. Plan to /compact soon. Keep responses concise.",
            percent
        )
    }
}
