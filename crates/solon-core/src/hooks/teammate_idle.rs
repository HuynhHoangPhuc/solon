use crate::hooks::{get_str, is_hook_enabled};
/// TeammateIdle hook: inject available task context when a teammate goes idle.
use anyhow::Result;

pub fn run() -> Result<()> {
    if !is_hook_enabled("teammate-idle-handler") {
        std::process::exit(0);
    }

    let input = match crate::hooks::read_hook_input() {
        Ok(v) => v,
        Err(_) => std::process::exit(0),
    };

    let team_name = get_str(&input, "team_name").to_string();
    if team_name.is_empty() {
        std::process::exit(0);
    }

    let teammate_name = get_str(&input, "teammate_name").to_string();
    let info = get_available_tasks(&team_name);

    let mut lines = vec![
        "## Teammate Idle".to_string(),
        format!("{} is idle.", teammate_name),
    ];

    if let Some(info) = &info {
        let remaining = info.pending + info.in_progress;
        lines.push(format!(
            "Tasks: {}/{} done. {} remaining.",
            info.completed, info.total, remaining
        ));

        if !info.unblocked.is_empty() {
            let parts: Vec<String> = info
                .unblocked
                .iter()
                .map(|(id, subject)| format!("#{} \"{}\"", id, subject))
                .collect();
            lines.push(format!("Unblocked & unassigned: {}", parts.join(", ")));
            lines.push(format!(
                "Consider assigning work to {} or waking them with a message.",
                teammate_name
            ));
        } else if remaining == 0 {
            lines.push(format!(
                "No remaining tasks. Consider shutting down {}.",
                teammate_name
            ));
        } else {
            lines.push(format!(
                "All remaining tasks are blocked or assigned. {} may be waiting for dependencies.",
                teammate_name
            ));
        }
    }

    crate::hooks::write_output(&serde_json::json!({
        "hookSpecificOutput": {
            "hookEventName": "TeammateIdle",
            "additionalContext": lines.join("\n"),
        }
    }));
    Ok(())
}

struct TeamInfo {
    pending: usize,
    in_progress: usize,
    completed: usize,
    total: usize,
    unblocked: Vec<(String, String)>, // (id, subject)
}

#[derive(serde::Deserialize)]
struct TaskFile {
    #[serde(default)]
    id: String,
    #[serde(default)]
    status: String,
    #[serde(default)]
    subject: String,
    #[serde(rename = "blockedBy", default)]
    blocked_by: Vec<String>,
    #[serde(default)]
    owner: String,
}

/// Read ~/.claude/tasks/{team}/*.json, compute pending/in_progress/completed and unblocked tasks.
fn get_available_tasks(team_name: &str) -> Option<TeamInfo> {
    let home = dirs::home_dir()?;
    let task_dir = home.join(".claude").join("tasks").join(team_name);
    let entries = std::fs::read_dir(&task_dir).ok()?;

    let mut tasks: Vec<TaskFile> = Vec::new();
    for entry in entries.flatten() {
        let path = entry.path();
        if path.extension().and_then(|e| e.to_str()) != Some("json") {
            continue;
        }
        let data = match std::fs::read_to_string(&path) {
            Ok(d) => d,
            Err(_) => continue,
        };
        let t: TaskFile = match serde_json::from_str(&data) {
            Ok(t) => t,
            Err(_) => continue,
        };
        if !t.status.is_empty() {
            tasks.push(t);
        }
    }

    // Build completed ID set for dependency resolution
    let completed_ids: std::collections::HashSet<&str> = tasks
        .iter()
        .filter(|t| t.status == "completed")
        .map(|t| t.id.as_str())
        .collect();

    let mut info = TeamInfo {
        pending: 0,
        in_progress: 0,
        completed: 0,
        total: 0,
        unblocked: Vec::new(),
    };
    for t in &tasks {
        match t.status.as_str() {
            "completed" => info.completed += 1,
            "in_progress" => info.in_progress += 1,
            "pending" => {
                info.pending += 1;
                // Unblocked = all deps complete AND no owner assigned
                let all_deps_done = t
                    .blocked_by
                    .iter()
                    .all(|dep| completed_ids.contains(dep.as_str()));
                if all_deps_done && t.owner.is_empty() {
                    info.unblocked.push((t.id.clone(), t.subject.clone()));
                }
            }
            _ => continue,
        }
    }
    info.total = info.pending + info.in_progress + info.completed;
    Some(info)
}
