/// SubagentStart hook: inject team peer list and task summary for team agents.
/// Agent ID format: "name@team-name" — extracts team name from that.
use anyhow::Result;
use serde_json::Value;

use crate::hooks::{get_str, is_hook_enabled};

pub fn run() -> Result<()> {
    if !is_hook_enabled("team-context-inject") {
        std::process::exit(0);
    }

    let input: Value = match crate::hooks::read_hook_input() {
        Ok(v) => v,
        Err(_) => std::process::exit(0),
    };

    let agent_id = get_str(&input, "agent_id").to_string();
    let team_name = match extract_team_name(&agent_id) {
        Some(t) => t,
        None => std::process::exit(0),
    };

    let home = dirs::home_dir().unwrap_or_default();
    let config_path = home
        .join(".claude")
        .join("teams")
        .join(&team_name)
        .join("config.json");

    let team_cfg = match read_team_config(&config_path) {
        Ok(c) => c,
        Err(_) => std::process::exit(0),
    };

    let peer_list = build_peer_list(&team_cfg, &agent_id);
    let tasks = summarize_team_tasks(&home, &team_name);

    let mut lines: Vec<String> = Vec::new();
    lines.push("## Team Context".to_string());
    let display_name = team_cfg
        .get("name")
        .and_then(|v| v.as_str())
        .unwrap_or(&team_name);
    lines.push(format!("Team: {}", display_name));
    lines.push(format!("Your peers: {}", peer_list));
    if let Some(t) = tasks {
        lines.push(format!(
            "Task summary: {} pending, {} in progress, {} completed",
            t.pending, t.in_progress, t.completed
        ));
    }

    let sl_ctx = build_sl_context();
    if sl_ctx.len() > 1 {
        lines.push(String::new());
        lines.push("## CK Context".to_string());
        lines.extend(sl_ctx);
    }

    lines.push(String::new());
    lines.push("Remember: Check TaskList, claim tasks, respect file ownership, use SendMessage to communicate.".to_string());

    let output = serde_json::json!({
        "hookSpecificOutput": {
            "hookEventName": "SubagentStart",
            "additionalContext": lines.join("\n")
        }
    });
    println!("{}", serde_json::to_string(&output)?);
    Ok(())
}

/// Extract team name from "name@team-name" format. Rejects path traversal.
fn extract_team_name(agent_id: &str) -> Option<String> {
    let idx = agent_id.find('@')?;
    if idx == 0 {
        return None;
    }
    let name = &agent_id[idx + 1..];
    if name.contains("..") || name.contains('/') || name.contains('\\') {
        return None;
    }
    if name.is_empty() {
        return None;
    }
    Some(name.to_string())
}

fn read_team_config(path: &std::path::Path) -> anyhow::Result<serde_json::Map<String, Value>> {
    let data = std::fs::read_to_string(path)?;
    let v: Value = serde_json::from_str(&data)?;
    v.as_object()
        .cloned()
        .ok_or_else(|| anyhow::anyhow!("not an object"))
}

fn build_peer_list(cfg: &serde_json::Map<String, Value>, current_agent_id: &str) -> String {
    let members = match cfg.get("members").and_then(|v| v.as_array()) {
        Some(m) => m,
        None => return "none".to_string(),
    };
    let peers: Vec<String> = members
        .iter()
        .filter_map(|m| m.as_object())
        .filter(|m| m.get("agentId").and_then(|v| v.as_str()) != Some(current_agent_id))
        .map(|m| {
            let name = m.get("name").and_then(|v| v.as_str()).unwrap_or("unknown");
            let agent_type = m
                .get("agentType")
                .and_then(|v| v.as_str())
                .unwrap_or("unknown");
            format!("{} ({})", name, agent_type)
        })
        .collect();
    if peers.is_empty() {
        "none".to_string()
    } else {
        peers.join(", ")
    }
}

struct TaskSummary {
    pending: usize,
    in_progress: usize,
    completed: usize,
}

fn summarize_team_tasks(home: &std::path::Path, team_name: &str) -> Option<TaskSummary> {
    let task_dir = home.join(".claude").join("tasks").join(team_name);
    let entries = std::fs::read_dir(&task_dir).ok()?;
    let mut s = TaskSummary {
        pending: 0,
        in_progress: 0,
        completed: 0,
    };
    for entry in entries.flatten() {
        if entry.path().is_dir() {
            continue;
        }
        if !entry.file_name().to_string_lossy().ends_with(".json") {
            continue;
        }
        let data = std::fs::read_to_string(entry.path()).ok()?;
        let v: Value = serde_json::from_str(&data).ok()?;
        match v.get("status").and_then(|s| s.as_str()) {
            Some("pending") => s.pending += 1,
            Some("in_progress") => s.in_progress += 1,
            Some("completed") => s.completed += 1,
            _ => {}
        }
    }
    Some(s)
}

fn build_sl_context() -> Vec<String> {
    let mut ctx = Vec::new();
    if let Ok(v) = std::env::var("SL_REPORTS_PATH") {
        if !v.is_empty() {
            ctx.push(format!("Reports: {}", v));
        }
    }
    if let Ok(v) = std::env::var("SL_PLANS_PATH") {
        if !v.is_empty() {
            ctx.push(format!("Plans: {}", v));
        }
    }
    if let Ok(v) = std::env::var("SL_PROJECT_ROOT") {
        if !v.is_empty() {
            ctx.push(format!("Project: {}", v));
        }
    }
    if let Ok(v) = std::env::var("SL_NAME_PATTERN") {
        if !v.is_empty() {
            ctx.push(format!("Naming: {}", v));
        }
    }
    if let Ok(v) = std::env::var("SL_GIT_BRANCH") {
        if !v.is_empty() {
            ctx.push(format!("Branch: {}", v));
        }
    }
    if let Ok(v) = std::env::var("SL_ACTIVE_PLAN") {
        if !v.is_empty() {
            ctx.push(format!("Active plan: {}", v));
        }
    }
    ctx.push("Commits: conventional (feat:, fix:, docs:, refactor:, test:, chore:)".to_string());
    ctx
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_extract_team_name() {
        assert_eq!(
            extract_team_name("alice@my-team"),
            Some("my-team".to_string())
        );
        assert_eq!(extract_team_name("alice"), None);
        assert_eq!(extract_team_name("@team"), None);
        assert_eq!(extract_team_name("alice@../evil"), None);
    }
}
