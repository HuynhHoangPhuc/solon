use crate::hooks::{budget_for_tool, get_str, is_hook_enabled, truncate_output};
/// PostToolUse hook: truncate large tool outputs to save context window space.
use anyhow::Result;

/// Tools whose output must never be truncated — Claude needs full content to verify file changes.
const TRUNCATION_WHITELIST: &[&str] = &["Edit", "Write", "MultiEdit", "NotebookEdit"];

pub fn run() -> Result<()> {
    if !is_hook_enabled("tool-output-truncation") {
        std::process::exit(0);
    }

    let input = match crate::hooks::read_hook_input() {
        Ok(v) => v,
        Err(_) => {
            crate::hooks::write_output(&serde_json::json!({"continue": true}));
            return Ok(());
        }
    };

    let tool_name = get_str(&input, "tool_name").to_string();
    if TRUNCATION_WHITELIST.contains(&tool_name.as_str()) {
        crate::hooks::write_output(&serde_json::json!({"continue": true}));
        return Ok(());
    }

    let tool_output = input
        .get("tool_output")
        .and_then(|v| v.as_str())
        .unwrap_or("")
        .to_string();

    let budget = budget_for_tool(&tool_name);
    let (result, changed) = truncate_output(
        &tool_output,
        budget.max_lines,
        budget.head_lines,
        budget.tail_lines,
    );

    if !changed {
        crate::hooks::write_output(&serde_json::json!({"continue": true}));
        return Ok(());
    }

    crate::hooks::write_output(&serde_json::json!({
        "continue": true,
        "hookSpecificOutput": {
            "tool_output": result,
        }
    }));
    Ok(())
}
