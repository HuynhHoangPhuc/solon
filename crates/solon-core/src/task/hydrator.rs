/// Plan task extraction from phase-*.md files.
/// Ported from Go solon-core/internal/task/hydrator.go
use anyhow::{anyhow, Result};
use regex::Regex;
use serde::{Deserialize, Serialize};
use std::path::Path;

/// A single plan phase as a task definition.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskDef {
    pub phase: usize,
    pub title: String,
    pub priority: String,
    pub effort: String,
    pub description: String,
    pub phase_file: String,
    pub blocked_by: Vec<usize>,
    pub todo_count: usize,
    pub done_count: usize,
}

/// JSON output of HydratePlan.
#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct HydrateResult {
    pub plan_dir: String,
    pub task_count: usize,
    pub tasks: Vec<TaskDef>,
    pub skipped: bool,
    pub skip_reason: String,
}

/// Scan planDir for phase-*.md files and return structured TaskDef list.
pub fn hydrate_plan(plan_dir: &str) -> Result<HydrateResult> {
    let dir = Path::new(plan_dir);
    let entries = std::fs::read_dir(dir).map_err(|e| anyhow!("read plan dir: {}", e))?;

    let mut phase_files: Vec<String> = entries
        .flatten()
        .filter(|e| {
            let name = e.file_name().to_string_lossy().to_string();
            !e.path().is_dir() && name.starts_with("phase-") && name.ends_with(".md")
        })
        .map(|e| e.file_name().to_string_lossy().to_string())
        .collect();
    phase_files.sort();

    let mut result = HydrateResult {
        plan_dir: plan_dir.to_string(),
        task_count: 0,
        tasks: Vec::new(),
        skipped: false,
        skip_reason: String::new(),
    };

    if phase_files.len() < 3 {
        result.skipped = true;
        result.skip_reason = format!(
            "only {} phase file(s) found, minimum 3 required",
            phase_files.len()
        );
        return Ok(result);
    }

    for fname in &phase_files {
        let task =
            parse_phase_file(plan_dir, fname).map_err(|e| anyhow!("parse {}: {}", fname, e))?;
        result.tasks.push(task);
    }

    // Sequential blocking: phase N blocked by N-1
    for i in 0..result.tasks.len() {
        result.tasks[i].blocked_by = if i == 0 {
            vec![]
        } else {
            vec![result.tasks[i - 1].phase]
        };
    }

    result.task_count = result.tasks.len();
    Ok(result)
}

fn parse_phase_file(plan_dir: &str, fname: &str) -> Result<TaskDef> {
    let phase_num = extract_phase_num(fname);
    let content = std::fs::read_to_string(Path::new(plan_dir).join(fname))?;

    let re_title_alt = Regex::new(r"(?i)^#\s+Phase\s+\d+:\s*(.+)").unwrap();
    let re_title_lbl = Regex::new(r"(?i)^#\s+Phase:\s*(.+)").unwrap();
    let re_priority = Regex::new(r"(?i)\*\*Priority:\*\*\s*(\S+)").unwrap();
    let re_effort = Regex::new(r"(?i)\*\*Effort:\*\*\s*(\S+)").unwrap();
    let re_description = Regex::new(r"(?i)\*\*Description:\*\*\s*(.+)").unwrap();

    let mut task = TaskDef {
        phase: phase_num,
        title: String::new(),
        priority: String::new(),
        effort: String::new(),
        description: String::new(),
        phase_file: fname.to_string(),
        blocked_by: Vec::new(),
        todo_count: 0,
        done_count: 0,
    };

    for line in content.lines() {
        let trimmed = line.trim();

        if task.title.is_empty() {
            if let Some(cap) = re_title_alt.captures(trimmed) {
                task.title = strip_backticks(cap[1].trim());
                continue;
            }
            if let Some(cap) = re_title_lbl.captures(trimmed) {
                task.title = strip_backticks(cap[1].trim());
                continue;
            }
        }
        if task.priority.is_empty() {
            if let Some(cap) = re_priority.captures(trimmed) {
                task.priority = cap[1].to_string();
            }
        }
        if task.effort.is_empty() {
            if let Some(cap) = re_effort.captures(trimmed) {
                task.effort = cap[1].to_string();
            }
        }
        if task.description.is_empty() {
            if let Some(cap) = re_description.captures(trimmed) {
                task.description = cap[1].trim().to_string();
            }
        }

        if trimmed.starts_with("- [ ]") {
            task.todo_count += 1;
        } else if trimmed.starts_with("- [x]") || trimmed.starts_with("- [X]") {
            task.done_count += 1;
            task.todo_count += 1;
        }
    }

    Ok(task)
}

fn extract_phase_num(fname: &str) -> usize {
    let re = Regex::new(r"^phase-(\d+)-").unwrap();
    re.captures(fname)
        .and_then(|c| c[1].parse().ok())
        .unwrap_or(0)
}

fn strip_backticks(s: &str) -> String {
    s.replace('`', "")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_extract_phase_num() {
        assert_eq!(extract_phase_num("phase-03-research.md"), 3);
        assert_eq!(extract_phase_num("phase-10-final.md"), 10);
        assert_eq!(extract_phase_num("other.md"), 0);
    }

    #[test]
    fn test_hydrate_too_few_phases() {
        let tmp = tempfile::tempdir().unwrap();
        let dir = tmp.path().to_str().unwrap();
        std::fs::write(tmp.path().join("phase-01-a.md"), "# Phase: A\n").unwrap();
        std::fs::write(tmp.path().join("phase-02-b.md"), "# Phase: B\n").unwrap();
        let r = hydrate_plan(dir).unwrap();
        assert!(r.skipped);
        assert!(r.skip_reason.contains("minimum 3"));
    }

    #[test]
    fn test_hydrate_parses_tasks() {
        let tmp = tempfile::tempdir().unwrap();
        let dir = tmp.path().to_str().unwrap();
        for i in 1..=3usize {
            std::fs::write(
                tmp.path().join(format!("phase-0{}-step.md", i)),
                format!(
                    "# Phase: Step {}\n\n- **Priority:** P1\n- **Effort:** S\n- **Description:** Desc {}\n\n## TODO\n\n- [ ] Task\n- [x] Done\n",
                    i, i
                ),
            )
            .unwrap();
        }
        let r = hydrate_plan(dir).unwrap();
        assert!(!r.skipped);
        assert_eq!(r.task_count, 3);
        assert_eq!(r.tasks[0].blocked_by, Vec::<usize>::new());
        assert_eq!(r.tasks[1].blocked_by, vec![1]);
        assert_eq!(r.tasks[0].todo_count, 2);
        assert_eq!(r.tasks[0].done_count, 1);
    }
}
