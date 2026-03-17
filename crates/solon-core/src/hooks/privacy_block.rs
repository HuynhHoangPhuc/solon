/// PreToolUse hook: blocks sensitive file access, requires user approval.
/// Exit 0 = allow, exit 2 = block with prompt data on stderr.
use anyhow::Result;
use serde_json::Value;

use crate::hooks::{
    build_privacy_prompt_data, get_str, has_approval_prefix, is_hook_enabled, is_privacy_sensitive,
    strip_approval_prefix,
};

pub fn run() -> Result<()> {
    if !is_hook_enabled("privacy-block") {
        std::process::exit(0);
    }

    let input: Value = match crate::hooks::read_hook_input() {
        Ok(v) => v,
        Err(_) => std::process::exit(0),
    };

    let tool_name = get_str(&input, "tool_name");
    let tool_input = match input.get("tool_input").and_then(|v| v.as_object()) {
        Some(m) => m.clone(),
        None => std::process::exit(0),
    };
    if tool_name.is_empty() {
        std::process::exit(0);
    }

    let result = check_privacy(tool_name, &tool_input, true);

    if result.approved {
        if result.suspicious {
            eprintln!(
                "\x1b[33mWARN:\x1b[0m Approved path is outside project: {}",
                result.file_path
            );
        }
        let base = std::path::Path::new(&result.file_path)
            .file_name()
            .map(|n| n.to_string_lossy().to_string())
            .unwrap_or_default();
        eprintln!("\x1b[32m✓\x1b[0m Privacy: User-approved access to {}", base);
        std::process::exit(0);
    }

    if result.is_bash {
        eprintln!("\x1b[33mWARN:\x1b[0m {}", result.reason);
        std::process::exit(0);
    }

    if result.blocked {
        let prompt_json = serde_json::to_string_pretty(&result.prompt_data).unwrap_or_default();
        eprint!(
            "{}",
            format_privacy_block_msg(&result.file_path, &prompt_json)
        );
        std::process::exit(2);
    }

    std::process::exit(0);
}

pub struct PrivacyCheckResult {
    pub blocked: bool,
    pub file_path: String,
    pub reason: String,
    pub approved: bool,
    pub is_bash: bool,
    pub suspicious: bool,
    pub prompt_data: Value,
}

/// Extract privacy-relevant paths from tool input.
fn extract_paths(tool_input: &serde_json::Map<String, Value>) -> Vec<(String, String)> {
    let mut paths: Vec<(String, String)> = Vec::new();

    for field in &["file_path", "path", "pattern"] {
        if let Some(Value::String(v)) = tool_input.get(*field) {
            if !v.is_empty() {
                paths.push((v.clone(), field.to_string()));
            }
        }
    }

    if let Some(Value::String(cmd)) = tool_input.get("command") {
        // Check for APPROVED: prefixed paths first
        let approved_re = regex::Regex::new(r"APPROVED:[^\s]+").unwrap();
        let approved: Vec<String> = approved_re
            .find_iter(cmd)
            .map(|m| m.as_str().to_string())
            .collect();
        if !approved.is_empty() {
            for p in approved {
                paths.push((p, "command".to_string()));
            }
        } else {
            // Extract .env references
            let env_re = regex::Regex::new(r"\.env[^\s]*").unwrap();
            for m in env_re.find_iter(cmd) {
                paths.push((m.as_str().to_string(), "command".to_string()));
            }
            // Variable assignments
            let var_re = regex::Regex::new(r"\w+=[^\s]*\.env[^\s]*").unwrap();
            for m in var_re.find_iter(cmd) {
                let parts: Vec<&str> = m.as_str().splitn(2, '=').collect();
                if parts.len() == 2 && !parts[1].is_empty() {
                    paths.push((parts[1].to_string(), "command".to_string()));
                }
            }
        }
    }

    paths.retain(|(v, _)| !v.is_empty());
    paths
}

pub fn check_privacy(
    tool_name: &str,
    tool_input: &serde_json::Map<String, Value>,
    allow_bash: bool,
) -> PrivacyCheckResult {
    let is_bash = tool_name == "Bash";
    let paths = extract_paths(tool_input);

    for (path, _field) in &paths {
        if !is_privacy_sensitive(path) {
            continue;
        }
        if has_approval_prefix(path) {
            let stripped = strip_approval_prefix(path).to_string();
            let suspicious =
                stripped.contains("..") || std::path::Path::new(&stripped).is_absolute();
            return PrivacyCheckResult {
                blocked: false,
                approved: true,
                file_path: stripped,
                suspicious,
                reason: String::new(),
                is_bash: false,
                prompt_data: Value::Null,
            };
        }
        if is_bash && allow_bash {
            return PrivacyCheckResult {
                blocked: false,
                is_bash: true,
                file_path: path.clone(),
                reason: format!("Bash command accesses sensitive file: {}", path),
                approved: false,
                suspicious: false,
                prompt_data: Value::Null,
            };
        }
        return PrivacyCheckResult {
            blocked: true,
            file_path: path.clone(),
            reason: "Sensitive file access requires user approval".to_string(),
            prompt_data: build_privacy_prompt_data(path),
            approved: false,
            is_bash: false,
            suspicious: false,
        };
    }

    PrivacyCheckResult {
        blocked: false,
        file_path: String::new(),
        reason: String::new(),
        approved: false,
        is_bash: false,
        suspicious: false,
        prompt_data: Value::Null,
    }
}

fn format_privacy_block_msg(file_path: &str, prompt_json: &str) -> String {
    format!(
        "\n\x1b[36mNOTE:\x1b[0m This is not an error - this block protects sensitive data.\n\n\
         \x1b[33mPRIVACY BLOCK\x1b[0m: Sensitive file access requires user approval\n\n\
         \x1b[33mFile:\x1b[0m  {}\n\n\
         This file may contain secrets (API keys, passwords, tokens).\n\n\
         \x1b[90m@@PRIVACY_PROMPT_START@@\x1b[0m\n{}\n\x1b[90m@@PRIVACY_PROMPT_END@@\x1b[0m\n\n\
         \x1b[34mClaude:\x1b[0m Use AskUserQuestion tool with the JSON above, then:\n\
         \x1b[32mIf \"Yes\":\x1b[0m Use bash to read: cat \"{}\"\n\
         \x1b[31mIf \"No\":\x1b[0m  Continue without this file.\n",
        file_path, prompt_json, file_path
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    fn make_input(file_path: &str) -> serde_json::Map<String, Value> {
        let mut m = serde_json::Map::new();
        m.insert(
            "file_path".to_string(),
            Value::String(file_path.to_string()),
        );
        m
    }

    #[test]
    fn test_env_file_blocked() {
        let input = make_input(".env");
        let r = check_privacy("Read", &input, true);
        assert!(r.blocked);
        assert_eq!(r.file_path, ".env");
    }

    #[test]
    fn test_env_example_allowed() {
        let input = make_input(".env.example");
        let r = check_privacy("Read", &input, true);
        assert!(!r.blocked);
    }

    #[test]
    fn test_approved_prefix() {
        let input = make_input("APPROVED:.env.local");
        let r = check_privacy("Read", &input, true);
        assert!(!r.blocked);
        assert!(r.approved);
    }

    #[test]
    fn test_bash_tool_warns_not_blocks() {
        let mut input = serde_json::Map::new();
        input.insert("command".to_string(), Value::String("cat .env".to_string()));
        let r = check_privacy("Bash", &input, true);
        assert!(!r.blocked);
        assert!(r.is_bash);
    }
}
