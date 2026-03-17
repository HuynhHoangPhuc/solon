use crate::hooks::is_hook_enabled;
/// PreToolUse(Write): inject file naming guidance as allow response.
use anyhow::Result;

const FILE_NAMING_GUIDANCE: &str = "\
## File naming guidance:
- Skip this guidance if you are creating markdown or plain text files
- Prefer kebab-case for JS/TS/Python/shell (.js, .ts, .py, .sh) with descriptive names
- Respect language conventions: C#/Java/Kotlin/Swift use PascalCase (.cs, .java, .kt, .swift), Go/Rust use snake_case (.go, .rs)
- Other languages: follow their ecosystem's standard naming convention
- Goal: self-documenting names for LLM tools (Grep, Glob, Search)";

pub fn run() -> Result<()> {
    if !is_hook_enabled("descriptive-name") {
        std::process::exit(0);
    }

    let output = serde_json::json!({
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "allow",
            "additionalContext": FILE_NAMING_GUIDANCE
        }
    });
    println!("{}", serde_json::to_string(&output)?);
    Ok(())
}
