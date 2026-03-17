/// Plan workflow status: phase completion tracking and progress calculation.
/// Ported from Go solon-core/internal/workflow/status.go
use anyhow::{anyhow, Result};
use serde::{Deserialize, Serialize};
use std::path::Path;

/// Status of a single phase.
#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PhaseStatus {
    pub phase: usize,
    pub file: String,
    /// "completed", "in-progress", or "pending"
    pub state: String,
    pub todo_total: usize,
    pub todo_done: usize,
}

/// Aggregate phase counts by state.
#[derive(Debug, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PhaseCounts {
    pub total: usize,
    pub completed: usize,
    pub in_progress: usize,
    pub pending: usize,
}

/// JSON output of GetStatus.
#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WorkflowStatus {
    pub plan_dir: String,
    /// "completed", "in-progress", or "pending"
    pub status: String,
    /// 0-100
    pub progress: usize,
    pub phases: PhaseCounts,
    pub reports: usize,
    pub detail: Vec<PhaseStatus>,
}

/// Return workflow status for the given plan directory.
pub fn get_status(plan_dir: &str) -> Result<WorkflowStatus> {
    let dir = Path::new(plan_dir);
    let entries = std::fs::read_dir(dir).map_err(|e| anyhow!("read plan dir: {}", e))?;

    let mut result = WorkflowStatus {
        plan_dir: plan_dir.to_string(),
        status: String::new(),
        progress: 0,
        phases: PhaseCounts::default(),
        reports: 0,
        detail: Vec::new(),
    };

    let mut files: Vec<String> = entries
        .flatten()
        .filter(|e| {
            let name = e.file_name().to_string_lossy().to_string();
            !e.path().is_dir() && name.starts_with("phase-") && name.ends_with(".md")
        })
        .map(|e| e.file_name().to_string_lossy().to_string())
        .collect();
    files.sort();

    for fname in files {
        let ps = categorize_phase(plan_dir, &fname)
            .map_err(|e| anyhow!("categorize {}: {}", fname, e))?;
        result.phases.total += 1;
        match ps.state.as_str() {
            "completed" => result.phases.completed += 1,
            "in-progress" => result.phases.in_progress += 1,
            _ => result.phases.pending += 1,
        }
        result.detail.push(ps);
    }

    // Calculate progress percentage
    if result.phases.total > 0 {
        result.progress = (result.phases.completed * 100) / result.phases.total;
    }

    // Overall status
    result.status = if result.phases.total == 0 {
        "pending".to_string()
    } else if result.phases.completed == result.phases.total {
        "completed".to_string()
    } else if result.phases.completed > 0 || result.phases.in_progress > 0 {
        "in-progress".to_string()
    } else {
        "pending".to_string()
    };

    result.reports = count_report_files(plan_dir);
    Ok(result)
}

fn categorize_phase(plan_dir: &str, fname: &str) -> Result<PhaseStatus> {
    let content = std::fs::read_to_string(Path::new(plan_dir).join(fname))?;

    let mut ps = PhaseStatus {
        phase: extract_phase_num(fname),
        file: fname.to_string(),
        state: String::new(),
        todo_total: 0,
        todo_done: 0,
    };

    for line in content.lines() {
        let t = line.trim();
        if t.starts_with("- [x]") || t.starts_with("- [X]") {
            ps.todo_done += 1;
            ps.todo_total += 1;
        } else if t.starts_with("- [ ]") {
            ps.todo_total += 1;
        }
    }

    ps.state = if ps.todo_total == 0 {
        "pending".to_string()
    } else if ps.todo_done == ps.todo_total {
        "completed".to_string()
    } else if ps.todo_done > 0 {
        "in-progress".to_string()
    } else {
        "pending".to_string()
    };

    Ok(ps)
}

fn extract_phase_num(fname: &str) -> usize {
    let stripped = fname.strip_prefix("phase-").unwrap_or(fname);
    stripped
        .chars()
        .take_while(|c| c.is_ascii_digit())
        .collect::<String>()
        .parse()
        .unwrap_or(0)
}

fn count_report_files(plan_dir: &str) -> usize {
    let mut count = 0;
    for sub in &["reports", "research"] {
        if let Ok(entries) = std::fs::read_dir(Path::new(plan_dir).join(sub)) {
            count += entries.flatten().filter(|e| !e.path().is_dir()).count();
        }
    }
    count
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_extract_phase_num() {
        assert_eq!(extract_phase_num("phase-03-research.md"), 3);
        assert_eq!(extract_phase_num("phase-10-final.md"), 10);
    }

    #[test]
    fn test_get_status_all_pending() {
        let tmp = tempfile::tempdir().unwrap();
        let dir = tmp.path().to_str().unwrap();
        std::fs::write(tmp.path().join("phase-01-a.md"), "## TODO\n\n- [ ] Task\n").unwrap();
        std::fs::write(tmp.path().join("phase-02-b.md"), "## TODO\n\n- [ ] Task\n").unwrap();

        let s = get_status(dir).unwrap();
        assert_eq!(s.status, "pending");
        assert_eq!(s.progress, 0);
        assert_eq!(s.phases.total, 2);
        assert_eq!(s.phases.pending, 2);
    }

    #[test]
    fn test_get_status_completed() {
        let tmp = tempfile::tempdir().unwrap();
        let dir = tmp.path().to_str().unwrap();
        std::fs::write(tmp.path().join("phase-01-a.md"), "- [x] Done\n").unwrap();

        let s = get_status(dir).unwrap();
        assert_eq!(s.status, "completed");
        assert_eq!(s.progress, 100);
    }

    #[test]
    fn test_get_status_in_progress() {
        let tmp = tempfile::tempdir().unwrap();
        let dir = tmp.path().to_str().unwrap();
        std::fs::write(tmp.path().join("phase-01-a.md"), "- [x] Done\n").unwrap();
        std::fs::write(tmp.path().join("phase-02-b.md"), "- [ ] Pending\n").unwrap();

        let s = get_status(dir).unwrap();
        assert_eq!(s.status, "in-progress");
        assert_eq!(s.progress, 50);
    }

    #[test]
    fn test_count_report_files() {
        let tmp = tempfile::tempdir().unwrap();
        let dir = tmp.path().to_str().unwrap();
        std::fs::create_dir(tmp.path().join("reports")).unwrap();
        std::fs::write(tmp.path().join("reports").join("report-1.md"), "").unwrap();
        std::fs::write(tmp.path().join("reports").join("report-2.md"), "").unwrap();
        assert_eq!(count_report_files(dir), 2);
    }
}
