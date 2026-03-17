use crate::hooks::{get_str, is_hook_enabled};
/// PostToolUse hook: detect AI-generated comment patterns in code edits and warn.
use anyhow::Result;
use serde_json::Value;

// Patterns matching common AI-generated comment anti-patterns.
// These match // # and /* styles at start of lines.
const SLOP_PATTERNS: &[&str] = &[
    // "This function/method/class handles/does/performs..."
    r"(?m)^\s*(?://|#|/\*)\s*This (?:function|method|class|module|component|hook|helper) (?:handles|does|performs|is responsible|takes care|manages|implements|provides|creates|returns|checks)",
    // "Updated to..." / "Modified to..." / "Changed to..."
    r"(?m)^\s*(?://|#|/\*)\s*(?:Updated|Modified|Changed|Refactored|Fixed|Added|Removed) (?:to |the |for |by )",
    // "Helper function/method that..."
    r"(?m)^\s*(?://|#|/\*)\s*Helper (?:function|method|class|utility) (?:that|which|to|for)",
    // "We need to..." / "We use..." / "Here we..."
    r"(?m)^\s*(?://|#|/\*)\s*(?:We|Here we) (?:need to|use|create|define|implement|check|handle|call)",
    // "The following..." / "Below is..." / "This is the..."
    r"(?m)^\s*(?://|#|/\*)\s*(?:The following|Below is|Above is|This is the)",
];

pub fn run() -> Result<()> {
    if !is_hook_enabled("comment-slop-checker") {
        std::process::exit(0);
    }

    let input = match crate::hooks::read_hook_input() {
        Ok(v) => v,
        Err(_) => {
            crate::hooks::write_output(&serde_json::json!({"continue": true}));
            return Ok(());
        }
    };

    let tool_name = get_str(&input, "tool_name");
    let content = extract_written_content(tool_name, input.get("tool_input"));
    if content.is_empty() {
        crate::hooks::write_output(&serde_json::json!({"continue": true}));
        return Ok(());
    }

    let match_count: usize = SLOP_PATTERNS
        .iter()
        .filter_map(|pat| regex::Regex::new(pat).ok())
        .map(|re| re.find_iter(&content).count())
        .sum();

    let mut output = serde_json::json!({"continue": true});
    if match_count > 0 {
        output["additionalContext"] = Value::String(
            "\n[Comment Quality] Detected AI-generated comment pattern(s) in your edit. \
             Avoid narrating what code does (\"This function handles...\", \"Updated to...\"). \
             Comments should explain WHY, not WHAT. Remove or rewrite slop comments.\n"
                .to_string(),
        );
    }

    crate::hooks::write_output(&output);
    Ok(())
}

/// Extract the written code content from tool_input based on tool type.
fn extract_written_content(tool_name: &str, tool_input: Option<&Value>) -> String {
    let input = match tool_input {
        Some(v) => v,
        None => return String::new(),
    };
    match tool_name {
        "Edit" => input
            .get("new_string")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string(),
        "Write" => input
            .get("content")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string(),
        "MultiEdit" => {
            if let Some(edits) = input.get("edits").and_then(|v| v.as_array()) {
                edits
                    .iter()
                    .filter_map(|e| e.get("new_string")?.as_str())
                    .collect::<Vec<_>>()
                    .join("\n")
            } else {
                String::new()
            }
        }
        _ => String::new(),
    }
}
