/// Task management CLI commands: hydrate, sync.
/// Ported from Go solon-core/cmd/task*.go
use crate::task::{hydrate_plan, sync_completions};
use anyhow::{anyhow, Result};
use clap::{Args, Subcommand};

#[derive(Args)]
pub struct TaskArgs {
    #[command(subcommand)]
    pub command: TaskCommand,
}

#[derive(Subcommand)]
pub enum TaskCommand {
    /// Extract tasks from phase files in a plan directory
    Hydrate(HydrateArgs),
    /// Mark completed phase TODO items as done
    Sync(SyncArgs),
}

#[derive(Args)]
pub struct HydrateArgs {
    /// Plan directory path
    pub plan_dir: String,
    /// Output format: json or text
    #[arg(long, default_value = "json")]
    pub format: String,
}

#[derive(Args)]
pub struct SyncArgs {
    /// Plan directory path
    pub plan_dir: String,
    /// Comma-separated list of completed phase numbers (e.g. 1,2,3)
    #[arg(long, default_value = "")]
    pub completed: String,
    /// Output format: json or text
    #[arg(long, default_value = "json")]
    pub format: String,
}

pub fn run(args: TaskArgs) -> Result<()> {
    match args.command {
        TaskCommand::Hydrate(a) => run_hydrate(a),
        TaskCommand::Sync(a) => run_sync(a),
    }
}

fn run_hydrate(args: HydrateArgs) -> Result<()> {
    let result = hydrate_plan(&args.plan_dir)?;

    if args.format == "text" {
        if result.skipped {
            println!("Skipped: {}", result.skip_reason);
            return Ok(());
        }
        println!("Plan: {}\nTasks: {}\n", result.plan_dir, result.task_count);
        for t in &result.tasks {
            println!(
                "  [{}] {} (priority: {}, effort: {}, todos: {})",
                t.phase, t.title, t.priority, t.effort, t.todo_count
            );
        }
        return Ok(());
    }
    println!("{}", serde_json::to_string_pretty(&result)?);
    Ok(())
}

fn run_sync(args: SyncArgs) -> Result<()> {
    let completed_phases: Vec<usize> = args
        .completed
        .split(',')
        .map(|s| s.trim())
        .filter(|s| !s.is_empty())
        .map(|s| {
            s.parse::<usize>()
                .map_err(|_| anyhow!("invalid phase number {:?}", s))
        })
        .collect::<Result<Vec<_>>>()?;

    let result = sync_completions(&args.plan_dir, &completed_phases)?;

    if args.format == "text" {
        println!(
            "Plan: {}\nFiles modified: {}\nCheckboxes updated: {}",
            result.plan_dir,
            result.files_modified.len(),
            result.checkboxes_updated
        );
        for d in &result.details {
            println!("  {}: {} updated", d.file, d.updated);
        }
        return Ok(());
    }
    println!("{}", serde_json::to_string_pretty(&result)?);
    Ok(())
}
