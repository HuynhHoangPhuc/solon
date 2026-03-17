use crate::hooks::{get_str, is_hook_enabled};
/// TaskCompleted hook: log task completion and inject progress context.
use anyhow::Result;

pub fn run() -> Result<()> {
    if !is_hook_enabled("task-completed-handler") {
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

    let task_id = get_str(&input, "task_id").to_string();
    let task_subject = get_str(&input, "task_subject").to_string();
    let teammate_name = get_str(&input, "teammate_name").to_string();

    log_task_completion(&team_name, &task_id, &task_subject, &teammate_name);

    let counts = count_team_tasks(&team_name);
    let mut lines = vec![
        "## Task Completed".to_string(),
        format!(
            "Task #{} \"{}\" completed by {}.",
            task_id, task_subject, teammate_name
        ),
    ];

    if let Some(c) = counts {
        let remaining = c.pending + c.in_progress;
        lines.push(format!(
            "Progress: {}/{} done. {} pending, {} in progress.",
            c.completed, c.total, c.pending, c.in_progress
        ));
        if remaining == 0 {
            lines.push(String::new());
            lines.push("**All tasks completed.** Consider shutting down teammates and synthesizing results.".to_string());
        }
    }

    crate::hooks::write_output(&serde_json::json!({
        "hookSpecificOutput": {
            "hookEventName": "TaskCompleted",
            "additionalContext": lines.join("\n"),
        }
    }));
    Ok(())
}

struct TaskCounts {
    pending: usize,
    in_progress: usize,
    completed: usize,
    total: usize,
}

/// Read ~/.claude/tasks/{team}/*.json and tally status counts.
fn count_team_tasks(team_name: &str) -> Option<TaskCounts> {
    let home = dirs::home_dir()?;
    let task_dir = home.join(".claude").join("tasks").join(team_name);
    let entries = std::fs::read_dir(&task_dir).ok()?;

    let mut counts = TaskCounts {
        pending: 0,
        in_progress: 0,
        completed: 0,
        total: 0,
    };
    for entry in entries.flatten() {
        let path = entry.path();
        if path.extension().and_then(|e| e.to_str()) != Some("json") {
            continue;
        }
        let data = match std::fs::read_to_string(&path) {
            Ok(d) => d,
            Err(_) => continue,
        };
        let v: serde_json::Value = match serde_json::from_str(&data) {
            Ok(v) => v,
            Err(_) => continue,
        };
        match v.get("status").and_then(|s| s.as_str()).unwrap_or("") {
            "pending" => counts.pending += 1,
            "in_progress" => counts.in_progress += 1,
            "completed" => counts.completed += 1,
            _ => continue,
        }
    }
    counts.total = counts.pending + counts.in_progress + counts.completed;
    Some(counts)
}

/// Append a completion line to the team reports log file.
fn log_task_completion(team_name: &str, task_id: &str, task_subject: &str, teammate_name: &str) {
    let reports_path = match std::env::var("SL_REPORTS_PATH") {
        Ok(p) if !p.is_empty() => p,
        _ => return,
    };
    let log_file =
        std::path::PathBuf::from(&reports_path).join(format!("team-{}-completions.md", team_name));
    if let Some(parent) = log_file.parent() {
        let _ = std::fs::create_dir_all(parent);
    }
    let timestamp = chrono::Local::now().format("%Y-%m-%d %H:%M:%S").to_string();
    let line = format!(
        "- [{}] Task #{} \"{}\" completed by {}\n",
        timestamp, task_id, task_subject, teammate_name
    );
    use std::io::Write;
    if let Ok(mut f) = std::fs::OpenOptions::new()
        .append(true)
        .create(true)
        .open(&log_file)
    {
        let _ = f.write_all(line.as_bytes());
    }
}
