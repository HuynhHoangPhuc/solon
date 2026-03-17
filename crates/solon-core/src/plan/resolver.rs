/// Plan path resolution: session → branch cascade strategy.
/// Ported from Go solon-core/internal/plan/resolver.go
use crate::plan::naming::sanitize_slug;
use crate::session::read_session_state;
use regex::Regex;
use serde::{Deserialize, Serialize};
use solon_common::{PlanConfig, PlanResolution, SLConfig};
use std::path::{Path, PathBuf};

/// JSON output of plan resolution.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(rename_all = "camelCase")]
pub struct ResolveResult {
    pub path: String,
    pub resolved_by: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub absolute: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub plan_file: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub status: String,
    #[serde(skip_serializing_if = "is_zero")]
    pub phases: usize,
}

fn is_zero(v: &usize) -> bool {
    *v == 0
}

/// Extract a sanitized feature slug from a git branch name.
pub fn extract_slug_from_branch(branch: &str, pattern: &str) -> String {
    if branch.is_empty() {
        return String::new();
    }
    let default_pat = r"(?:feat|fix|chore|refactor|docs)/(?:[^/]+/)?(.+)";
    let re = if pattern.is_empty() {
        Regex::new(default_pat).unwrap()
    } else {
        Regex::new(pattern).unwrap_or_else(|_| Regex::new(default_pat).unwrap())
    };
    re.captures(branch)
        .and_then(|c| c.get(1))
        .map(|m| sanitize_slug(m.as_str()))
        .unwrap_or_default()
}

/// Return the most recently created plan directory (lexicographically last timestamped dir).
pub fn find_most_recent_plan(plans_dir: &str) -> String {
    let timestamp_re = Regex::new(r"^\d{6}").unwrap();
    let entries = match std::fs::read_dir(plans_dir) {
        Ok(e) => e,
        Err(_) => return String::new(),
    };
    let mut latest = String::new();
    for entry in entries.flatten() {
        let name = entry.file_name().to_string_lossy().to_string();
        if entry.path().is_dir() && timestamp_re.is_match(&name) && name > latest {
            latest = name;
        }
    }
    if latest.is_empty() {
        return String::new();
    }
    Path::new(plans_dir)
        .join(&latest)
        .to_string_lossy()
        .to_string()
}

/// Run a whitelisted read-only git command, return trimmed stdout or empty string.
fn git_safe(cmd: &str) -> String {
    let allowed = [
        "git branch --show-current",
        "git rev-parse --abbrev-ref HEAD",
    ];
    if !allowed.contains(&cmd) {
        return String::new();
    }
    std::process::Command::new("sh")
        .args(["-c", cmd])
        .output()
        .ok()
        .filter(|o| o.status.success())
        .map(|o| String::from_utf8_lossy(&o.stdout).trim().to_string())
        .unwrap_or_default()
}

/// Resolve the active plan using the configured resolution order.
pub fn resolve_plan_path(session_id: &str, cfg: &SLConfig) -> PlanResolution {
    let plans_dir = if cfg.paths.plans.is_empty() {
        "plans"
    } else {
        &cfg.paths.plans
    };
    let order = if cfg.plan.resolution.order.is_empty() {
        vec!["session".to_string(), "branch".to_string()]
    } else {
        cfg.plan.resolution.order.clone()
    };

    for method in &order {
        match method.as_str() {
            "session" => {
                if let Some(state) = read_session_state(session_id) {
                    if let Some(active) = &state.active_plan {
                        if !active.is_empty() {
                            let resolved = if !Path::new(active).is_absolute()
                                && !state.session_origin.is_empty()
                            {
                                Path::new(&state.session_origin)
                                    .join(active)
                                    .to_string_lossy()
                                    .to_string()
                            } else {
                                active.clone()
                            };
                            return PlanResolution {
                                path: resolved,
                                resolved_by: "session".to_string(),
                            };
                        }
                    }
                }
            }
            "branch" => {
                let branch = git_safe("git branch --show-current");
                let slug = extract_slug_from_branch(&branch, &cfg.plan.resolution.branch_pattern);
                if slug.is_empty() {
                    continue;
                }
                let entries = match std::fs::read_dir(plans_dir) {
                    Ok(e) => e,
                    Err(_) => continue,
                };
                let mut matched = String::new();
                for entry in entries.flatten() {
                    let name = entry.file_name().to_string_lossy().to_string();
                    if entry.path().is_dir() && name.contains(&slug) {
                        matched = name;
                    }
                }
                if !matched.is_empty() {
                    return PlanResolution {
                        path: Path::new(plans_dir)
                            .join(&matched)
                            .to_string_lossy()
                            .to_string(),
                        resolved_by: "branch".to_string(),
                    };
                }
            }
            _ => {}
        }
    }
    PlanResolution::default()
}

/// Read a simple YAML frontmatter field value from a file.
pub fn extract_frontmatter_field(file_path: &Path, field: &str) -> String {
    let data = match std::fs::read_to_string(file_path) {
        Ok(d) => d,
        Err(_) => return String::new(),
    };
    let prefix = format!("{}:", field);
    let mut in_frontmatter = false;
    for line in data.lines() {
        let trimmed = line.trim();
        if trimmed == "---" {
            if in_frontmatter {
                break;
            }
            in_frontmatter = true;
            continue;
        }
        if in_frontmatter && trimmed.starts_with(&prefix) {
            return trimmed[prefix.len()..].trim().to_string();
        }
    }
    String::new()
}

/// Count phase-*.md files in a directory.
pub fn count_phase_files(dir: &Path) -> usize {
    std::fs::read_dir(dir)
        .map(|entries| {
            entries
                .flatten()
                .filter(|e| {
                    let name = e.file_name().to_string_lossy().to_string();
                    !e.path().is_dir() && name.starts_with("phase-") && name.ends_with(".md")
                })
                .count()
        })
        .unwrap_or(0)
}

/// Add absolute path, plan file info, status, and phase count to a resolution.
pub fn enrich_resolve_result(res: PlanResolution) -> ResolveResult {
    let mut result = ResolveResult {
        path: res.path.clone(),
        resolved_by: res.resolved_by.clone(),
        ..Default::default()
    };
    if res.path.is_empty() {
        return result;
    }

    // Compute absolute path
    let abs = if Path::new(&res.path).is_absolute() {
        PathBuf::from(&res.path)
    } else {
        std::env::current_dir().unwrap_or_default().join(&res.path)
    };
    result.absolute = abs.to_string_lossy().to_string();

    // Check for plan.md
    let plan_file = abs.join("plan.md");
    if plan_file.exists() {
        result.plan_file = "plan.md".to_string();
        result.status = extract_frontmatter_field(&plan_file, "status");
    }

    result.phases = count_phase_files(&abs);
    result
}

/// Return the reports path based on plan resolution.
pub fn get_reports_path(
    plan_path: &str,
    resolved_by: &str,
    plan_cfg: &PlanConfig,
    plans_dir: &str,
    base_dir: &str,
) -> String {
    let reports_dir = {
        let r = plan_cfg.reports_dir.trim().trim_end_matches('/');
        if r.is_empty() {
            "reports"
        } else {
            r
        }
    };
    let plans = {
        let p = plans_dir.trim().trim_end_matches('/');
        if p.is_empty() {
            "plans"
        } else {
            p
        }
    };

    let report_path = if !plan_path.is_empty() && resolved_by == "session" {
        let normalized = plan_path.trim().trim_end_matches('/');
        if !normalized.is_empty() {
            format!("{}/{}", normalized, reports_dir)
        } else {
            format!("{}/{}", plans, reports_dir)
        }
    } else {
        format!("{}/{}", plans, reports_dir)
    };

    if !base_dir.is_empty() {
        Path::new(base_dir)
            .join(&report_path)
            .to_string_lossy()
            .to_string()
    } else {
        format!("{}/", report_path)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_extract_slug_from_branch() {
        assert_eq!(
            extract_slug_from_branch("feat/my-feature", ""),
            "my-feature"
        );
        assert_eq!(
            extract_slug_from_branch("fix/issue-42-bug-fix", ""),
            "issue-42-bug-fix"
        );
        assert_eq!(extract_slug_from_branch("main", ""), "");
    }

    #[test]
    fn test_extract_frontmatter_field() {
        use std::io::Write;
        let mut f = tempfile::NamedTempFile::new().unwrap();
        write!(f, "---\nstatus: pending\npriority: P1\n---\n").unwrap();
        let val = extract_frontmatter_field(f.path(), "status");
        assert_eq!(val, "pending");
    }

    #[test]
    fn test_enrich_empty_path() {
        let res = PlanResolution::default();
        let enriched = enrich_resolve_result(res);
        assert!(enriched.path.is_empty());
        assert!(enriched.absolute.is_empty());
    }
}
