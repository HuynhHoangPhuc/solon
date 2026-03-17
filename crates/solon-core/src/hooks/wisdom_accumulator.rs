use crate::hooks::{get_env, get_str, is_hook_enabled};
/// SubagentStop hook: extract learnings from transcript tail and append to wisdom file.
use anyhow::Result;

const MAX_WISDOM_LINES: usize = 50;

pub fn run() -> Result<()> {
    if !is_hook_enabled("wisdom-accumulation") {
        std::process::exit(0);
    }

    let input = match crate::hooks::read_hook_input() {
        Ok(v) => v,
        Err(_) => serde_json::Value::Null,
    };

    let plan_path = get_env("SL_ACTIVE_PLAN").unwrap_or_default();
    let session_id = {
        let s = get_str(&input, "session_id").to_string();
        if s.is_empty() {
            get_env("SL_SESSION_ID").unwrap_or_default()
        } else {
            s
        }
    };

    let wisdom_path = if !plan_path.is_empty() {
        std::path::PathBuf::from(&plan_path).join(".wisdom.md")
    } else if !session_id.is_empty() {
        std::env::temp_dir().join(format!("sl-wisdom-{}.md", session_id))
    } else {
        std::process::exit(0);
    };

    let transcript_path = get_str(&input, "transcript_path").to_string();
    let agent_type = get_str(&input, "agent_type").to_string();

    let learnings = extract_learnings(&transcript_path, &agent_type);
    if learnings.is_empty() {
        std::process::exit(0);
    }

    append_wisdom(&wisdom_path, &learnings);
    prune_wisdom(&wisdom_path, MAX_WISDOM_LINES);

    Ok(())
}

fn extract_learnings(transcript_path: &str, agent_type: &str) -> String {
    if transcript_path.is_empty() {
        return String::new();
    }

    let data = match std::fs::read_to_string(transcript_path) {
        Ok(d) => d,
        Err(_) => return String::new(),
    };

    let all_lines: Vec<&str> = data.lines().collect();
    let start = if all_lines.len() > 30 {
        all_lines.len() - 30
    } else {
        0
    };
    let lines = &all_lines[start..];

    let keywords = [
        "learned",
        "gotcha",
        "important",
        "note:",
        "warning:",
        "convention",
        "pattern",
        "decision",
        "discovered",
        "must",
        "should",
        "avoid",
        "don't",
        "careful",
        "works",
        "doesn't work",
        "failed",
        "succeeded",
    ];

    let mut learnings: Vec<String> = lines
        .iter()
        .filter_map(|line| {
            let lower = line.to_lowercase();
            if keywords.iter().any(|kw| lower.contains(kw)) {
                let trimmed = line.trim().to_string();
                if trimmed.len() > 10 && trimmed.len() < 200 {
                    return Some(trimmed);
                }
            }
            None
        })
        .collect();

    if learnings.is_empty() {
        return String::new();
    }
    if learnings.len() > 5 {
        let start = learnings.len() - 5;
        learnings = learnings[start..].to_vec();
    }

    let now = chrono::Local::now();
    let header = format!("### {} ({})", agent_type, now.format("%H:%M"));
    format!("{}\n{}", header, learnings.join("\n"))
}

fn append_wisdom(path: &std::path::Path, content: &str) {
    use std::io::Write;
    if let Ok(mut f) = std::fs::OpenOptions::new()
        .append(true)
        .create(true)
        .open(path)
    {
        let _ = writeln!(f, "\n{}", content);
    }
}

fn prune_wisdom(path: &std::path::Path, max_lines: usize) {
    let data = match std::fs::read_to_string(path) {
        Ok(d) => d,
        Err(_) => return,
    };
    let lines: Vec<&str> = data.split('\n').collect();
    if lines.len() <= max_lines {
        return;
    }
    let pruned = lines[lines.len() - max_lines..].join("\n");
    let _ = std::fs::write(path, pruned);
}
