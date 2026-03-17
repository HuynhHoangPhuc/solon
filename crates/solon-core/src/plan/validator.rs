/// Plan directory completeness validation.
/// Ported from Go solon-core/internal/plan/validator.go
use crate::plan::resolver::extract_frontmatter_field;
use serde::{Deserialize, Serialize};
use std::path::Path;

/// JSON output of plan validation.
#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ValidationResult {
    pub valid: bool,
    pub plan_dir: String,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub errors: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub warnings: Vec<String>,
    pub stats: ValidationStats,
}

/// Counts about the plan.
#[derive(Debug, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ValidationStats {
    pub phase_count: usize,
    pub todo_total: usize,
    pub todo_completed: usize,
}

/// Check a plan directory for completeness.
pub fn validate_plan(plan_dir: &str) -> ValidationResult {
    let mut result = ValidationResult {
        valid: true,
        plan_dir: plan_dir.to_string(),
        errors: Vec::new(),
        warnings: Vec::new(),
        stats: ValidationStats::default(),
    };

    let dir = Path::new(plan_dir);

    // Check plan.md exists
    let plan_file = dir.join("plan.md");
    if !plan_file.exists() {
        result.valid = false;
        result.errors.push("plan.md not found".to_string());
    } else {
        let status = extract_frontmatter_field(&plan_file, "status");
        if status.is_empty() {
            result
                .warnings
                .push("plan.md missing status in frontmatter".to_string());
        }
    }

    // Read directory entries
    let entries = match std::fs::read_dir(dir) {
        Ok(e) => e,
        Err(_) => {
            result.valid = false;
            result.errors.push("cannot read plan directory".to_string());
            return result;
        }
    };

    for entry in entries.flatten() {
        let name = entry.file_name().to_string_lossy().to_string();
        if entry.path().is_dir() || !name.starts_with("phase-") || !name.ends_with(".md") {
            continue;
        }
        result.stats.phase_count += 1;

        let content = match std::fs::read_to_string(entry.path()) {
            Ok(c) => c,
            Err(_) => continue,
        };

        let has_todo = content.contains("## TODO") || content.contains("## Todo");
        if !has_todo {
            result
                .warnings
                .push(format!("{} missing TODO section", name));
        }

        for line in content.lines() {
            let trimmed = line.trim();
            if trimmed.starts_with("- [ ]") {
                result.stats.todo_total += 1;
            } else if trimmed.starts_with("- [x]") || trimmed.starts_with("- [X]") {
                result.stats.todo_total += 1;
                result.stats.todo_completed += 1;
            }
        }
    }

    if result.stats.phase_count == 0 {
        result.valid = false;
        result.errors.push("no phase files found".to_string());
    }

    result
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_validate_nonexistent_dir() {
        let r = validate_plan("/nonexistent/plan/dir");
        assert!(!r.valid);
        assert!(r
            .errors
            .iter()
            .any(|e| e.contains("plan.md not found") || e.contains("cannot read")));
    }

    #[test]
    fn test_validate_valid_plan() {
        let tmp = tempfile::tempdir().unwrap();
        let dir = tmp.path();

        // Write plan.md with frontmatter
        std::fs::write(dir.join("plan.md"), "---\nstatus: pending\n---\n# Plan\n").unwrap();

        // Write a phase file with TODO
        std::fs::write(
            dir.join("phase-01-research.md"),
            "# Phase: Research\n\n## TODO\n\n- [ ] Task 1\n- [x] Task 2\n",
        )
        .unwrap();

        let r = validate_plan(dir.to_str().unwrap());
        assert!(r.valid);
        assert_eq!(r.stats.phase_count, 1);
        assert_eq!(r.stats.todo_total, 2);
        assert_eq!(r.stats.todo_completed, 1);
    }

    #[test]
    fn test_validate_missing_todo_section() {
        let tmp = tempfile::tempdir().unwrap();
        let dir = tmp.path();
        std::fs::write(dir.join("plan.md"), "---\nstatus: pending\n---\n").unwrap();
        std::fs::write(
            dir.join("phase-01-step.md"),
            "# Phase: Step\n\nNo todo here.\n",
        )
        .unwrap();

        let r = validate_plan(dir.to_str().unwrap());
        assert!(r
            .warnings
            .iter()
            .any(|w| w.contains("missing TODO section")));
    }
}
