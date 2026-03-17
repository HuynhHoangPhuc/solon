use crate::hooks::{classify_intent, get_str, is_hook_enabled, write_context};
/// UserPromptSubmit hook: classify intent and inject compact strategy guidance.
use anyhow::Result;

pub fn run() -> Result<()> {
    if !is_hook_enabled("intent-gate") {
        std::process::exit(0);
    }

    let input = match crate::hooks::read_hook_input() {
        Ok(v) => v,
        Err(_) => std::process::exit(0),
    };

    let prompt = get_str(&input, "prompt");
    if prompt.is_empty() {
        std::process::exit(0);
    }

    if let Some(cat) = classify_intent(prompt) {
        write_context(&format!("[Intent: {}] {}\n", cat.name, cat.strategy));
    }

    Ok(())
}
