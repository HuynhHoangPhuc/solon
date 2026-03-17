/// Sync completion state: mark all TODO items as done in specified phase files.
/// Ported from Go solon-core/internal/task/syncer.go
use anyhow::{anyhow, Result};
use regex::Regex;
use serde::{Deserialize, Serialize};
use std::collections::HashSet;
use std::path::Path;

/// Output of SyncCompletions.
#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SyncResult {
    pub plan_dir: String,
    pub files_modified: Vec<String>,
    pub checkboxes_updated: usize,
    pub details: Vec<SyncDetail>,
}

/// Per-file sync stats.
#[derive(Debug, Serialize, Deserialize)]
pub struct SyncDetail {
    pub file: String,
    pub updated: usize,
}

/// Mark all TODO items as done in the specified phase files.
/// Uses atomic write (temp file + rename).
pub fn sync_completions(plan_dir: &str, completed_phases: &[usize]) -> Result<SyncResult> {
    if completed_phases.is_empty() {
        return Ok(SyncResult {
            plan_dir: plan_dir.to_string(),
            files_modified: vec![],
            checkboxes_updated: 0,
            details: vec![],
        });
    }

    let phase_set: HashSet<usize> = completed_phases.iter().copied().collect();
    let re_phase_num = Regex::new(r"^phase-(\d+)-").unwrap();

    let entries = std::fs::read_dir(plan_dir).map_err(|e| anyhow!("read plan dir: {}", e))?;

    let mut result = SyncResult {
        plan_dir: plan_dir.to_string(),
        files_modified: vec![],
        checkboxes_updated: 0,
        details: vec![],
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
        let phase_num: usize = re_phase_num
            .captures(&fname)
            .and_then(|c| c[1].parse().ok())
            .unwrap_or(0);

        if !phase_set.contains(&phase_num) {
            continue;
        }

        let updated =
            sync_phase_file(plan_dir, &fname).map_err(|e| anyhow!("sync {}: {}", fname, e))?;

        if updated > 0 {
            result.files_modified.push(fname.clone());
            result.checkboxes_updated += updated;
            result.details.push(SyncDetail {
                file: fname,
                updated,
            });
        }
    }

    Ok(result)
}

/// Replace all unchecked "- [ ]" with "- [x]" in a phase file atomically.
/// Returns count of replacements made.
fn sync_phase_file(plan_dir: &str, fname: &str) -> Result<usize> {
    let fpath = Path::new(plan_dir).join(fname);
    let content = std::fs::read_to_string(&fpath)?;
    let count = content.matches("- [ ]").count();
    if count == 0 {
        return Ok(0);
    }

    let updated = content.replace("- [ ]", "- [x]");
    let tmp_path = format!("{}.tmp", fpath.display());
    std::fs::write(&tmp_path, &updated).map_err(|e| anyhow!("write temp file: {}", e))?;
    std::fs::rename(&tmp_path, &fpath).map_err(|e| {
        let _ = std::fs::remove_file(&tmp_path);
        anyhow!("rename temp file: {}", e)
    })?;

    Ok(count)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sync_empty_phases_returns_empty() {
        let r = sync_completions("/tmp", &[]).unwrap();
        assert_eq!(r.checkboxes_updated, 0);
        assert!(r.files_modified.is_empty());
    }

    #[test]
    fn test_sync_marks_todos_done() {
        let tmp = tempfile::tempdir().unwrap();
        let dir = tmp.path().to_str().unwrap();
        std::fs::write(
            tmp.path().join("phase-01-step.md"),
            "## TODO\n\n- [ ] Task A\n- [ ] Task B\n- [x] Already done\n",
        )
        .unwrap();

        let r = sync_completions(dir, &[1]).unwrap();
        assert_eq!(r.checkboxes_updated, 2);
        assert_eq!(r.files_modified, vec!["phase-01-step.md"]);

        let content = std::fs::read_to_string(tmp.path().join("phase-01-step.md")).unwrap();
        assert!(!content.contains("- [ ]"));
        assert_eq!(content.matches("- [x]").count(), 3);
    }

    #[test]
    fn test_sync_skips_unspecified_phases() {
        let tmp = tempfile::tempdir().unwrap();
        let dir = tmp.path().to_str().unwrap();
        std::fs::write(tmp.path().join("phase-02-step.md"), "- [ ] Task\n").unwrap();

        let r = sync_completions(dir, &[1]).unwrap(); // phase 1 not present
        assert_eq!(r.checkboxes_updated, 0);
    }
}
