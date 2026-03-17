/// PreToolUse hook: blocks access to .slignore-listed paths and broad glob patterns.
/// Exit 0 = allow, exit 2 = block.
use anyhow::Result;
use serde_json::Value;

use crate::hooks::{
    get_str, has_specific_directory, is_broad_pattern, is_high_level_path, is_hook_enabled,
    load_slignore_patterns, path_matches_patterns, suggest_specific_patterns,
};

pub fn run() -> Result<()> {
    if !is_hook_enabled("scout-block") {
        std::process::exit(0);
    }

    let raw_input = match crate::hooks::read_hook_input() {
        Ok(v) => v,
        Err(_) => {
            eprintln!("ERROR: Empty input");
            std::process::exit(2);
        }
    };

    let tool_name = get_str(&raw_input, "tool_name");
    let tool_name = if tool_name.is_empty() {
        "unknown"
    } else {
        tool_name
    };

    let tool_input = match raw_input.get("tool_input").and_then(|v| v.as_object()) {
        Some(m) => m.clone(),
        None => {
            eprintln!("WARN: Invalid JSON structure, allowing operation");
            std::process::exit(0);
        }
    };

    let cwd = std::env::current_dir().unwrap_or_default();
    let claude_dir = cwd.join(".claude");
    let slignore_path = claude_dir.join(".slignore").to_string_lossy().to_string();

    // Check for allowed build/tool commands first
    if let Some(Value::String(cmd)) = tool_input.get("command") {
        let unwrapped = unwrap_shell_executor(cmd);
        let sub_cmds = split_compound_command(&unwrapped);
        if !sub_cmds.is_empty() {
            let non_allowed: Vec<&str> = sub_cmds
                .iter()
                .map(|s| s.trim())
                .filter(|s| !is_allowed_command(s))
                .collect();
            if non_allowed.is_empty() {
                // All sub-commands are allowed
                std::process::exit(0);
            }
        }
    }

    // Check for broad glob patterns
    if tool_name == "Glob" || tool_input.get("pattern").is_some() {
        let pattern = tool_input
            .get("pattern")
            .and_then(|v| v.as_str())
            .unwrap_or("");
        let base_path = tool_input
            .get("path")
            .and_then(|v| v.as_str())
            .unwrap_or("");
        if !pattern.is_empty()
            && !has_specific_directory(pattern)
            && is_broad_pattern(pattern)
            && is_high_level_path(base_path)
        {
            let suggestions = suggest_specific_patterns(pattern);
            let msg = format_broad_pattern_error(pattern, &suggestions);
            eprint!("{}", msg);
            std::process::exit(2);
        }
    }

    // Load .slignore and check extracted paths
    let patterns = load_slignore_patterns(&slignore_path);
    let extracted = extract_from_tool_input(&tool_input);

    if extracted.is_empty() {
        std::process::exit(0);
    }

    for path in &extracted {
        if let Some(matched_pattern) = path_matches_patterns(path, &patterns) {
            let msg = format_blocked_error(
                path,
                &matched_pattern,
                tool_name,
                &claude_dir.to_string_lossy(),
            );
            eprint!("{}", msg);
            std::process::exit(2);
        }
    }

    std::process::exit(0);
}

fn unwrap_shell_executor(cmd: &str) -> String {
    let re = regex::Regex::new(r#"^(?:(?:bash|sh|zsh)\s+-c|eval)\s+["'](.+)["']\s*$"#).unwrap();
    if let Some(caps) = re.captures(cmd.trim()) {
        if let Some(m) = caps.get(1) {
            return m.as_str().to_string();
        }
    }
    cmd.to_string()
}

fn split_compound_command(cmd: &str) -> Vec<String> {
    if cmd.is_empty() {
        return Vec::new();
    }
    let re = regex::Regex::new(r"\s*(?:&&|\|\||;)\s*").unwrap();
    re.split(cmd)
        .map(|s| s.trim().to_string())
        .filter(|s| !s.is_empty())
        .collect()
}

fn is_allowed_command(cmd: &str) -> bool {
    let stripped = strip_command_prefix(cmd);
    is_build_command(&stripped) || is_venv_executable(&stripped) || is_venv_creation(&stripped)
}

fn strip_command_prefix(cmd: &str) -> String {
    let mut s = cmd.trim().to_string();
    let env_var_re = regex::Regex::new(r"^(\w+=\S+\s+)+").unwrap();
    s = env_var_re.replace_all(&s, "").to_string();
    let wrapper_re = regex::Regex::new(r"^(sudo|env|nice|nohup|time|timeout)\s+").unwrap();
    s = wrapper_re.replace_all(&s, "").to_string();
    s = env_var_re.replace_all(&s, "").to_string();
    s.trim().to_string()
}

fn is_build_command(cmd: &str) -> bool {
    if cmd.is_empty() {
        return false;
    }
    let build_re = regex::Regex::new(
        r"^(npm|pnpm|yarn|bun)\s+([^\s]+\s+)*(run\s+)?(build|test|lint|dev|start|install|ci|add|remove|update|publish|pack|init|create|exec)"
    ).unwrap();
    let tool_re = regex::Regex::new(
        r"^(\./)?(?:npx|pnpx|bunx|tsc|esbuild|vite|webpack|rollup|turbo|nx|jest|vitest|mocha|eslint|prettier|go|cargo|make|mvn|mvnw|gradle|gradlew|dotnet|docker|podman|kubectl|helm|terraform|ansible|bazel|cmake|sbt|flutter|swift|ant|ninja|meson|python3?|pip|uv|deno|bundle|rake|gem|php|composer|ruby|mix|elixir)"
    ).unwrap();
    build_re.is_match(cmd.trim()) || tool_re.is_match(cmd.trim())
}

fn is_venv_executable(cmd: &str) -> bool {
    let re = regex::Regex::new(r"(^|[/\\])\.?venv[/\\](bin|Scripts)[/\\]").unwrap();
    re.is_match(cmd)
}

fn is_venv_creation(cmd: &str) -> bool {
    let re = regex::Regex::new(
        r"^(python3?|py)\s+(-[\w.]+\s+)*-m\s+venv\s+|^uv\s+venv(\s|$)|^virtualenv\s+",
    )
    .unwrap();
    re.is_match(cmd.trim())
}

fn extract_from_tool_input(tool_input: &serde_json::Map<String, Value>) -> Vec<String> {
    let mut paths = Vec::new();

    for param in &["file_path", "path", "pattern"] {
        if let Some(Value::String(v)) = tool_input.get(*param) {
            let n = normalize_extracted_path(v);
            if !n.is_empty() {
                paths.push(n);
            }
        }
    }

    if let Some(Value::String(cmd)) = tool_input.get("command") {
        paths.extend(extract_from_command(cmd));
    }

    paths.retain(|p| !p.is_empty());
    paths
}

fn extract_from_command(cmd: &str) -> Vec<String> {
    if cmd.is_empty() {
        return Vec::new();
    }
    let mut paths = Vec::new();

    let quoted_re = regex::Regex::new(r#"["']([^"']+)["']"#).unwrap();
    let sed_re = regex::Regex::new(r"^s[/|@#,]").unwrap();

    for caps in quoted_re.captures_iter(cmd) {
        let content = caps.get(1).map(|m| m.as_str()).unwrap_or("");
        if sed_re.is_match(content) {
            continue;
        }
        if looks_like_path(content) {
            paths.push(normalize_extracted_path(content));
        }
    }

    let without_quotes = quoted_re.replace_all(cmd, " ").to_string();
    let tokens: Vec<&str> = without_quotes.split_whitespace().collect();

    let fs_commands: std::collections::HashSet<&str> = [
        "cd", "ls", "cat", "head", "tail", "less", "more", "rm", "cp", "mv", "find", "touch",
        "mkdir", "rmdir", "stat", "file", "du", "tree",
    ]
    .iter()
    .copied()
    .collect();

    let command_keywords: std::collections::HashSet<&str> = [
        "echo", "cat", "ls", "cd", "rm", "cp", "mv", "find", "grep", "head", "tail", "npm", "pnpm",
        "yarn", "bun", "npx", "node", "git", "cargo", "go", "python", "python3", "pip", "run",
        "build", "lint", "dev", "start", "install",
    ]
    .iter()
    .copied()
    .collect();

    let mut command_name = "";

    for token in &tokens {
        if token.starts_with('-') {
            continue;
        }
        if *token == "&&" || *token == ";" || token.starts_with('|') {
            command_name = "";
            continue;
        }
        if command_name.is_empty() {
            command_name = token;
            let lower = command_name.to_lowercase();
            if command_keywords.contains(lower.as_str()) || fs_commands.contains(lower.as_str()) {
                continue;
            }
        }
        if command_keywords.contains(token.to_lowercase().as_str()) {
            continue;
        }
        if looks_like_path(token) {
            paths.push(normalize_extracted_path(token));
        }
    }

    paths
}

fn looks_like_path(s: &str) -> bool {
    if s.is_empty() || s.len() < 2 {
        return false;
    }
    if s.contains('/') || s.contains('\\') {
        return true;
    }
    if s.starts_with("./") || s.starts_with("../") {
        return true;
    }
    let ext_re = regex::Regex::new(r"\.\w{1,6}$").unwrap();
    if ext_re.is_match(s) {
        return true;
    }
    let seg_re = regex::Regex::new(r"^[a-zA-Z0-9_\-]+/").unwrap();
    seg_re.is_match(s)
}

fn normalize_extracted_path(p: &str) -> String {
    let mut s = p.trim().to_string();
    if s.len() >= 2 {
        let b = s.as_bytes();
        if (b[0] == b'"' && b[s.len() - 1] == b'"') || (b[0] == b'\'' && b[s.len() - 1] == b'\'') {
            s = s[1..s.len() - 1].to_string();
        }
    }
    s = s.trim_start_matches(['`', '(', '{', '[']).to_string();
    s = s.trim_end_matches(['`', ')', '}', ']', ';']).to_string();
    s = s.replace('\\', "/");
    if s.len() > 1 && s.ends_with('/') {
        s.pop();
    }
    s
}

fn format_blocked_error(path: &str, pattern: &str, tool: &str, claude_dir: &str) -> String {
    let config_path = if !claude_dir.is_empty() {
        format!("{}/.slignore", claude_dir)
    } else {
        ".claude/.slignore".to_string()
    };
    let display_path = if path.len() > 60 {
        format!("...{}", &path[path.len() - 57..])
    } else {
        path.to_string()
    };

    format!(
        "\n\x1b[36mNOTE:\x1b[0m This is not an error - this block is intentional to optimize context.\n\n\
         \x1b[31mBLOCKED\x1b[0m: Access to '{}' denied\n\n\
         \x1b[33mPattern:\x1b[0m  {}\n\
         \x1b[33mTool:\x1b[0m     {}\n\n\
         \x1b[34mTo allow, add to\x1b[0m {}:\n  !{}\n\n\
         \x1b[2mConfig:\x1b[0m {}\n\n",
        display_path, pattern, tool, config_path, pattern, config_path
    )
}

fn format_broad_pattern_error(pattern: &str, suggestions: &[String]) -> String {
    let mut lines = vec![
        "\n\x1b[36mNOTE:\x1b[0m This is not an error - this block is intentional to optimize context.\n".to_string(),
        "\x1b[31mBLOCKED\x1b[0m: Overly broad glob pattern detected\n".to_string(),
        format!("  \x1b[33mPattern:\x1b[0m  {}", pattern),
        "  \x1b[33mReason:\x1b[0m   Would return ALL matching files, filling context\n".to_string(),
        "  \x1b[34mUse more specific patterns:\x1b[0m".to_string(),
    ];
    for s in suggestions {
        lines.push(format!("    • {}", s));
    }
    lines.push(String::new());
    lines.push(
        "  \x1b[2mTip: Target specific directories to avoid context overflow\x1b[0m\n".to_string(),
    );
    lines.join("\n")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_is_build_command() {
        assert!(is_build_command("npm run build"));
        assert!(is_build_command("cargo test"));
        assert!(is_build_command("go build"));
        assert!(!is_build_command("rm -rf node_modules"));
    }

    #[test]
    fn test_is_venv_executable() {
        assert!(is_venv_executable(".venv/bin/python3"));
        assert!(is_venv_executable("venv/Scripts/python.exe"));
        assert!(!is_venv_executable("python3 script.py"));
    }

    #[test]
    fn test_split_compound_command() {
        let parts = split_compound_command("npm install && cargo build");
        assert_eq!(parts.len(), 2);
    }

    #[test]
    fn test_broad_pattern_blocked() {
        assert!(is_broad_pattern("**") && is_high_level_path(""));
        assert!(is_broad_pattern("*.ts") && is_high_level_path(""));
        assert!(!has_specific_directory("*.ts"));
    }
}
