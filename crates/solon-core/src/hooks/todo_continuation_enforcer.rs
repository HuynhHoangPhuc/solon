use crate::hooks::{get_env, is_hook_enabled, write_context};
/// UserPromptSubmit hook: remind about incomplete plan todos (throttled, 10min cooldown).
use anyhow::Result;

const COOLDOWN_MS: i64 = 10 * 60 * 1000;
const COOLDOWN_FILE: &str = "sl-todo-enforcer-cooldown.json";

pub fn run() -> Result<()> {
    if !is_hook_enabled("todo-continuation-enforcer") {
        std::process::exit(0);
    }

    let _input = match crate::hooks::read_hook_input() {
        Ok(v) => v,
        Err(_) => std::process::exit(0),
    };

    let plan_path = get_env("SL_ACTIVE_PLAN").unwrap_or_default();
    if plan_path.is_empty() {
        std::process::exit(0);
    }

    if !cooldown_expired() {
        std::process::exit(0);
    }

    let pattern = format!("{}/phase-*.md", plan_path);
    let phase_files: Vec<_> = glob::glob(&pattern)
        .map(|p| p.flatten().collect())
        .unwrap_or_default();
    if phase_files.is_empty() {
        std::process::exit(0);
    }

    let mut total_incomplete = 0usize;
    let mut files_with_todos = 0usize;

    for f in &phase_files {
        let count = count_incomplete_todos(f);
        if count > 0 {
            total_incomplete += count;
            files_with_todos += 1;
        }
    }

    if total_incomplete == 0 {
        std::process::exit(0);
    }

    write_cooldown();
    write_context(&format!(
        "\n[Plan Progress] {} incomplete todo(s) across {} phase file(s). \
         Continue working through the plan before considering the task done.\n",
        total_incomplete, files_with_todos
    ));

    Ok(())
}

fn count_incomplete_todos(path: &std::path::Path) -> usize {
    let data = match std::fs::read_to_string(path) {
        Ok(d) => d,
        Err(_) => return 0,
    };
    data.lines()
        .filter(|l| l.trim().starts_with("- [ ]"))
        .count()
}

fn cooldown_path() -> std::path::PathBuf {
    std::env::temp_dir().join(COOLDOWN_FILE)
}

fn cooldown_expired() -> bool {
    let data = match std::fs::read_to_string(cooldown_path()) {
        Ok(d) => d,
        Err(_) => return true,
    };
    let v: serde_json::Value = match serde_json::from_str(&data) {
        Ok(v) => v,
        Err(_) => return true,
    };
    let ts = v.get("timestamp").and_then(|t| t.as_i64()).unwrap_or(0);
    let now = chrono::Utc::now().timestamp_millis();
    now - ts > COOLDOWN_MS
}

fn write_cooldown() {
    let entry = serde_json::json!({ "timestamp": chrono::Utc::now().timestamp_millis() });
    let _ = std::fs::write(
        cooldown_path(),
        serde_json::to_string(&entry).unwrap_or_default(),
    );
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_count_incomplete_todos() {
        use std::io::Write;
        let mut f = tempfile::NamedTempFile::new().unwrap();
        writeln!(f, "- [ ] task one\n- [x] done\n- [ ] task two").unwrap();
        assert_eq!(count_incomplete_todos(f.path()), 2);
    }
}
