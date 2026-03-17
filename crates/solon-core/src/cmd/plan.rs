/// Plan management CLI commands: scaffold, resolve, validate, red-team, archive.
/// Ported from Go solon-core/cmd/plan*.go
use crate::config::load_config;
use crate::plan::{
    enrich_resolve_result, resolve_plan_path, scaffold_plan, validate_plan, ScaffoldMode,
};
use crate::template::render_red_team;
use anyhow::{anyhow, Result};
use clap::{Args, Subcommand};
use std::path::{Path, PathBuf};

#[derive(Args)]
pub struct PlanArgs {
    #[command(subcommand)]
    pub command: PlanCommand,
}

#[derive(Subcommand)]
pub enum PlanCommand {
    /// Create plan directory with template files
    Scaffold(ScaffoldArgs),
    /// Resolve active plan path
    Resolve(FormatArgs),
    /// Check plan completeness
    Validate(ValidateArgs),
    /// Generate red-team review prompt
    #[command(name = "red-team")]
    RedTeam(PlanDirArgs),
    /// Mark plan as archived
    Archive(PlanDirArgs),
}

#[derive(Args)]
pub struct ScaffoldArgs {
    /// Plan slug (required)
    #[arg(long)]
    pub slug: String,
    /// Scaffold mode: fast, hard, parallel, two
    #[arg(long, default_value = "hard")]
    pub mode: String,
    /// Custom number of phases (overrides mode)
    #[arg(long, default_value_t = 0)]
    pub phases: usize,
    /// Output format: json or text
    #[arg(long, default_value = "json")]
    pub format: String,
}

#[derive(Args)]
pub struct FormatArgs {
    /// Output format: json or text
    #[arg(long, default_value = "json")]
    pub format: String,
}

#[derive(Args)]
pub struct ValidateArgs {
    /// Plan directory path
    pub plan_dir: String,
    /// Output format: json or text
    #[arg(long, default_value = "json")]
    pub format: String,
}

#[derive(Args)]
pub struct PlanDirArgs {
    /// Plan directory path
    pub plan_dir: String,
}

pub fn run(args: PlanArgs) -> Result<()> {
    match args.command {
        PlanCommand::Scaffold(a) => run_scaffold(a),
        PlanCommand::Resolve(a) => run_resolve(a),
        PlanCommand::Validate(a) => run_validate(a),
        PlanCommand::RedTeam(a) => run_red_team(a),
        PlanCommand::Archive(a) => run_archive(a),
    }
}

fn run_scaffold(args: ScaffoldArgs) -> Result<()> {
    if args.slug.is_empty() {
        return Err(anyhow!("--slug is required"));
    }
    let mode = match args.mode.as_str() {
        "fast" => ScaffoldMode::Fast,
        "parallel" => ScaffoldMode::Parallel,
        "two" => ScaffoldMode::Two,
        _ => ScaffoldMode::Hard,
    };
    let cfg = load_config();
    let result = scaffold_plan(&args.slug, &mode, args.phases, &cfg)?;

    if args.format == "text" {
        println!("Created plan: {} (mode: {})", result.plan_dir, result.mode);
        for f in &result.files_created {
            println!("  + {}", f);
        }
        return Ok(());
    }
    println!("{}", serde_json::to_string_pretty(&result)?);
    Ok(())
}

fn run_resolve(args: FormatArgs) -> Result<()> {
    let session_id = std::env::var("SL_SESSION_ID").unwrap_or_default();
    let cfg = load_config();
    let resolved = resolve_plan_path(&session_id, &cfg);
    let result = enrich_resolve_result(resolved);

    if args.format == "text" {
        if result.path.is_empty() {
            println!("No active plan resolved");
            return Ok(());
        }
        println!("Plan: {}", result.path);
        println!("Resolved by: {}", result.resolved_by);
        if !result.status.is_empty() {
            println!("Status: {}", result.status);
        }
        if result.phases > 0 {
            println!("Phases: {}", result.phases);
        }
        return Ok(());
    }
    println!("{}", serde_json::to_string_pretty(&result)?);
    Ok(())
}

fn run_validate(args: ValidateArgs) -> Result<()> {
    let result = validate_plan(&args.plan_dir);
    if args.format == "text" {
        if result.valid {
            println!(
                "Plan valid: {} ({} phases, {}/{} TODOs)",
                args.plan_dir,
                result.stats.phase_count,
                result.stats.todo_completed,
                result.stats.todo_total
            );
        } else {
            println!("Plan invalid: {}", args.plan_dir);
            for e in &result.errors {
                println!("  ERROR: {}", e);
            }
        }
        for w in &result.warnings {
            println!("  WARN: {}", w);
        }
        return Ok(());
    }
    println!("{}", serde_json::to_string_pretty(&result)?);
    Ok(())
}

fn run_red_team(args: PlanDirArgs) -> Result<()> {
    let dir = Path::new(&args.plan_dir);
    let entries = std::fs::read_dir(dir).map_err(|e| anyhow!("read plan dir: {}", e))?;
    let mut phases: Vec<String> = entries
        .flatten()
        .filter(|e| {
            let name = e.file_name().to_string_lossy().to_string();
            !e.path().is_dir() && name.starts_with("phase-") && name.ends_with(".md")
        })
        .map(|e| {
            PathBuf::from(&args.plan_dir)
                .join(e.file_name())
                .to_string_lossy()
                .to_string()
        })
        .collect();
    phases.sort();
    let prompt = render_red_team(&args.plan_dir, &phases);
    print!("{}", prompt);
    Ok(())
}

fn run_archive(args: PlanDirArgs) -> Result<()> {
    let plan_file = Path::new(&args.plan_dir).join("plan.md");
    let content =
        std::fs::read_to_string(&plan_file).map_err(|e| anyhow!("read plan.md: {}", e))?;
    let updated = update_frontmatter_status(&content, "archived");
    if updated == content {
        println!("No status field found in frontmatter; no changes made");
        return Ok(());
    }
    std::fs::write(&plan_file, &updated).map_err(|e| anyhow!("write plan.md: {}", e))?;
    println!("Archived: {}", plan_file.display());
    Ok(())
}

/// Replace `status: <value>` in YAML frontmatter.
fn update_frontmatter_status(content: &str, new_status: &str) -> String {
    let lines: Vec<&str> = content.lines().collect();
    let mut in_frontmatter = false;
    for (i, line) in lines.iter().enumerate() {
        let trimmed = line.trim();
        if trimmed == "---" {
            if in_frontmatter {
                break;
            }
            in_frontmatter = true;
            continue;
        }
        if in_frontmatter && trimmed.starts_with("status:") {
            // Rebuild lines with replacement
            let mut result: Vec<String> = lines[..i].iter().map(|s| s.to_string()).collect();
            result.push(format!("status: {}", new_status));
            result.extend(lines[i + 1..].iter().map(|s| s.to_string()));
            return result.join("\n");
        }
    }
    content.to_string()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_update_frontmatter_status() {
        let content = "---\nstatus: pending\npriority: P1\n---\n# Plan\n";
        let updated = update_frontmatter_status(content, "archived");
        assert!(updated.contains("status: archived"));
        assert!(!updated.contains("status: pending"));
    }

    #[test]
    fn test_update_frontmatter_no_status_unchanged() {
        let content = "---\npriority: P1\n---\n# Plan\n";
        let updated = update_frontmatter_status(content, "archived");
        assert_eq!(updated, content);
    }
}
