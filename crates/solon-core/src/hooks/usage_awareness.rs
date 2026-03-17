/// UserPromptSubmit/PostToolUse hook: fetch and cache Anthropic usage limits.
/// Always outputs {"continue":true}. Fetches API in background when cache is stale.
use anyhow::Result;
use serde_json::Value;
use std::time::{SystemTime, UNIX_EPOCH};

use crate::hooks::is_hook_enabled;

const USAGE_CACHE_FILE: &str = "sl-usage-limits-cache.json";
const CACHE_TTL_PROMPT_MS: u64 = 60_000; // 1 minute for user prompts
const CACHE_TTL_DEFAULT_MS: u64 = 300_000; // 5 minutes otherwise
const USAGE_API_URL: &str = "https://api.anthropic.com/api/oauth/usage";

pub fn run() -> Result<()> {
    if !is_hook_enabled("usage-context-awareness") {
        std::process::exit(0);
    }

    // Always output continue:true regardless of errors
    let _guard = ContinueGuard;

    let input: Value = crate::hooks::read_hook_input().unwrap_or(Value::Null);
    let is_user_prompt = input.get("prompt").and_then(|v| v.as_str()).is_some();

    if should_fetch(is_user_prompt) {
        fetch_and_cache_usage();
    }

    Ok(())
}

/// RAII guard that always prints {"continue":true} on drop.
struct ContinueGuard;
impl Drop for ContinueGuard {
    fn drop(&mut self) {
        print!("{{\"continue\":true}}");
    }
}

fn cache_path() -> std::path::PathBuf {
    std::env::temp_dir().join(USAGE_CACHE_FILE)
}

fn now_ms() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis() as u64
}

fn should_fetch(is_user_prompt: bool) -> bool {
    let ttl = if is_user_prompt {
        CACHE_TTL_PROMPT_MS
    } else {
        CACHE_TTL_DEFAULT_MS
    };
    let data = match std::fs::read_to_string(cache_path()) {
        Ok(d) => d,
        Err(_) => return true,
    };
    let v: Value = match serde_json::from_str(&data) {
        Ok(v) => v,
        Err(_) => return true,
    };
    let ts = v.get("timestamp").and_then(|t| t.as_u64()).unwrap_or(0);
    now_ms().saturating_sub(ts) >= ttl
}

fn write_cache(status: &str, data: Option<Value>) {
    let entry = serde_json::json!({
        "timestamp": now_ms(),
        "status": status,
        "data": data
    });
    let _ = std::fs::write(
        cache_path(),
        serde_json::to_string(&entry).unwrap_or_default(),
    );
}

fn get_credentials() -> Option<String> {
    // Try ~/.claude/.credentials.json
    let home = dirs::home_dir()?;
    let cred_path = home.join(".claude").join(".credentials.json");
    let data = std::fs::read_to_string(&cred_path).ok()?;
    let v: Value = serde_json::from_str(&data).ok()?;
    let token = v.get("claudeAiOauth")?.get("accessToken")?.as_str()?;
    if token.is_empty() {
        return None;
    }
    Some(token.to_string())
}

fn fetch_and_cache_usage() {
    let token = match get_credentials() {
        Some(t) => t,
        None => {
            write_cache("unavailable", None);
            return;
        }
    };

    // Use std::process to call curl — avoids reqwest/tokio dep
    let output = std::process::Command::new("curl")
        .args([
            "-s",
            "-m",
            "10",
            "-H",
            &format!("Authorization: Bearer {}", token),
            "-H",
            "anthropic-beta: oauth-2025-04-20",
            "-H",
            "User-Agent: solon/1.0",
            "-H",
            "Accept: application/json",
            USAGE_API_URL,
        ])
        .output();

    match output {
        Ok(o) if o.status.success() => {
            let body = String::from_utf8_lossy(&o.stdout);
            if let Ok(data) = serde_json::from_str::<Value>(&body) {
                write_cache("available", Some(data));
            } else {
                write_cache("unavailable", None);
            }
        }
        _ => write_cache("unavailable", None),
    }
}
