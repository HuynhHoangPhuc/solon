/// Plan directory scaffolding: creates plan.md and phase template files.
/// Ported from Go solon-core/internal/plan/scaffolder.go
use crate::plan::naming::build_plan_dir_name;
use crate::template::{render_phase, render_plan};
use anyhow::{anyhow, Result};
use serde::{Deserialize, Serialize};
use solon_common::SLConfig;
use std::path::Path;

/// Scaffold modes controlling which phase templates are created.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ScaffoldMode {
    Fast,
    Hard,
    Parallel,
    Two,
}

impl ScaffoldMode {
    pub fn as_str(&self) -> &'static str {
        match self {
            ScaffoldMode::Fast => "fast",
            ScaffoldMode::Hard => "hard",
            ScaffoldMode::Parallel => "parallel",
            ScaffoldMode::Two => "two",
        }
    }
}

impl std::str::FromStr for ScaffoldMode {
    type Err = ();
    fn from_str(s: &str) -> std::result::Result<Self, Self::Err> {
        match s {
            "fast" => Ok(ScaffoldMode::Fast),
            "hard" => Ok(ScaffoldMode::Hard),
            "parallel" => Ok(ScaffoldMode::Parallel),
            "two" => Ok(ScaffoldMode::Two),
            _ => Err(()),
        }
    }
}

/// JSON output of plan scaffolding.
#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ScaffoldResult {
    pub plan_dir: String,
    pub mode: String,
    pub files_created: Vec<String>,
}

fn default_phases(mode: &ScaffoldMode) -> Vec<&'static str> {
    match mode {
        ScaffoldMode::Fast => vec![
            "phase-01-research.md",
            "phase-02-implementation.md",
            "phase-03-testing.md",
        ],
        ScaffoldMode::Hard => vec![
            "phase-01-research.md",
            "phase-02-design.md",
            "phase-03-implementation.md",
            "phase-04-testing.md",
            "phase-05-review.md",
        ],
        ScaffoldMode::Parallel => vec![
            "phase-01-research.md",
            "phase-02-design.md",
            "phase-03-implementation-a.md",
            "phase-04-implementation-b.md",
            "phase-05-integration.md",
            "phase-06-testing.md",
            "phase-07-review.md",
        ],
        ScaffoldMode::Two => vec!["phase-01-first-pass.md", "phase-02-second-pass.md"],
    }
}

/// Convert a phase filename to a human-readable title.
/// e.g. "phase-01-research.md" → "Research"
fn phase_title(filename: &str) -> String {
    let name = filename.trim_end_matches(".md");
    let parts: Vec<&str> = name.splitn(3, '-').collect();
    let base = if parts.len() >= 3 { parts[2] } else { name };
    base.split('-')
        .map(|w| {
            let mut chars = w.chars();
            match chars.next() {
                None => String::new(),
                Some(c) => c.to_uppercase().to_string() + chars.as_str(),
            }
        })
        .collect::<Vec<_>>()
        .join(" ")
}

/// Create a plan directory with plan.md and phase template files.
pub fn scaffold_plan(
    slug: &str,
    mode: &ScaffoldMode,
    num_phases: usize,
    cfg: &SLConfig,
) -> Result<ScaffoldResult> {
    if slug.is_empty() {
        return Err(anyhow!("slug is required"));
    }

    let plans_dir = if cfg.paths.plans.is_empty() {
        "plans"
    } else {
        &cfg.paths.plans
    };

    let dir_name = build_plan_dir_name(&cfg.plan, "", slug);
    let plan_dir = Path::new(plans_dir).join(&dir_name);

    std::fs::create_dir_all(plan_dir.join("research"))
        .map_err(|e| anyhow!("create plan dir: {}", e))?;
    std::fs::create_dir_all(plan_dir.join("reports"))
        .map_err(|e| anyhow!("create reports dir: {}", e))?;

    let mut files_created = Vec::new();

    // Write plan.md
    let plan_content = render_plan(slug, mode.as_str(), &dir_name);
    std::fs::write(plan_dir.join("plan.md"), &plan_content)
        .map_err(|e| anyhow!("write plan.md: {}", e))?;
    files_created.push("plan.md".to_string());

    // Determine phase file names
    let phase_names: Vec<String> = if num_phases > 0 {
        (1..=num_phases)
            .map(|i| format!("phase-{:02}-step-{}.md", i, i))
            .collect()
    } else {
        default_phases(mode).iter().map(|s| s.to_string()).collect()
    };

    let total = phase_names.len();
    for phase_name in &phase_names {
        let title = phase_title(phase_name);
        let content = render_phase(&title, total);
        std::fs::write(plan_dir.join(phase_name), &content)
            .map_err(|e| anyhow!("write {}: {}", phase_name, e))?;
        files_created.push(phase_name.clone());
    }

    Ok(ScaffoldResult {
        plan_dir: plan_dir.to_string_lossy().to_string(),
        mode: mode.as_str().to_string(),
        files_created,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_phase_title() {
        assert_eq!(phase_title("phase-01-research.md"), "Research");
        assert_eq!(
            phase_title("phase-03-implementation-a.md"),
            "Implementation A"
        );
    }

    #[test]
    fn test_scaffold_empty_slug_errors() {
        let cfg = SLConfig::default();
        let result = scaffold_plan("", &ScaffoldMode::Fast, 0, &cfg);
        assert!(result.is_err());
    }

    #[test]
    fn test_scaffold_creates_files() {
        let tmp = tempfile::tempdir().unwrap();
        let mut cfg = SLConfig::default();
        cfg.paths.plans = tmp.path().to_string_lossy().to_string();
        cfg.plan.naming_format = "{date}-{slug}".to_string();
        cfg.plan.date_format = "YYMMDD".to_string();

        let result = scaffold_plan("my-feature", &ScaffoldMode::Fast, 0, &cfg).unwrap();
        assert_eq!(result.mode, "fast");
        assert!(result.files_created.contains(&"plan.md".to_string()));
        assert!(result
            .files_created
            .contains(&"phase-01-research.md".to_string()));
        assert!(std::path::Path::new(&result.plan_dir)
            .join("plan.md")
            .exists());
    }

    #[test]
    fn test_scaffold_custom_num_phases() {
        let tmp = tempfile::tempdir().unwrap();
        let mut cfg = SLConfig::default();
        cfg.paths.plans = tmp.path().to_string_lossy().to_string();
        cfg.plan.naming_format = "{date}-{slug}".to_string();
        cfg.plan.date_format = "YYMMDD".to_string();

        let result = scaffold_plan("test", &ScaffoldMode::Hard, 2, &cfg).unwrap();
        assert_eq!(result.files_created.len(), 3); // plan.md + 2 phases
    }
}
