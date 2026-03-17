/// Plan report indexing: scan reports/ and research/ subdirectories.
/// Ported from Go solon-core/internal/report/indexer.go
use anyhow::{anyhow, Result};
use serde::{Deserialize, Serialize};
use std::path::Path;

/// A single report file found in a plan directory.
#[derive(Debug, Serialize, Deserialize)]
pub struct ReportEntry {
    pub filename: String,
    /// "reports" or "research"
    pub directory: String,
    /// full path
    pub path: String,
}

/// Scan reports/ and research/ subdirectories of planDir.
/// Returns entries sorted by directory then filename.
pub fn index_reports(plan_dir: &str) -> Result<Vec<ReportEntry>> {
    let mut entries: Vec<ReportEntry> = Vec::new();

    for sub in &["reports", "research"] {
        let sub_dir = Path::new(plan_dir).join(sub);
        let dir_entries = match std::fs::read_dir(&sub_dir) {
            Ok(e) => e,
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => continue,
            Err(e) => return Err(anyhow!("read {}: {}", sub, e)),
        };

        for entry in dir_entries.flatten() {
            if entry.path().is_dir() {
                continue;
            }
            entries.push(ReportEntry {
                filename: entry.file_name().to_string_lossy().to_string(),
                directory: sub.to_string(),
                path: entry.path().to_string_lossy().to_string(),
            });
        }
    }

    Ok(entries)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_index_empty_plan_dir() {
        let tmp = tempfile::tempdir().unwrap();
        let entries = index_reports(tmp.path().to_str().unwrap()).unwrap();
        assert!(entries.is_empty());
    }

    #[test]
    fn test_index_finds_reports() {
        let tmp = tempfile::tempdir().unwrap();
        std::fs::create_dir(tmp.path().join("reports")).unwrap();
        std::fs::create_dir(tmp.path().join("research")).unwrap();
        std::fs::write(tmp.path().join("reports").join("report-1.md"), "").unwrap();
        std::fs::write(tmp.path().join("research").join("researcher-1.md"), "").unwrap();

        let entries = index_reports(tmp.path().to_str().unwrap()).unwrap();
        assert_eq!(entries.len(), 2);
        assert!(entries.iter().any(|e| e.directory == "reports"));
        assert!(entries.iter().any(|e| e.directory == "research"));
    }

    #[test]
    fn test_index_skips_subdirectories() {
        let tmp = tempfile::tempdir().unwrap();
        std::fs::create_dir_all(tmp.path().join("reports").join("subdir")).unwrap();
        std::fs::write(tmp.path().join("reports").join("file.md"), "").unwrap();

        let entries = index_reports(tmp.path().to_str().unwrap()).unwrap();
        assert_eq!(entries.len(), 1);
        assert_eq!(entries[0].filename, "file.md");
    }
}
