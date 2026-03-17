/// Workflow management CLI commands: status.
/// Ported from Go solon-core/cmd/workflow*.go
use crate::workflow::get_status;
use anyhow::Result;
use clap::{Args, Subcommand};

#[derive(Args)]
pub struct WorkflowArgs {
    #[command(subcommand)]
    pub command: WorkflowCommand,
}

#[derive(Subcommand)]
pub enum WorkflowCommand {
    /// Show workflow completion status for a plan
    Status(StatusArgs),
}

#[derive(Args)]
pub struct StatusArgs {
    /// Plan directory path
    pub plan_dir: String,
    /// Output format: json or text
    #[arg(long, default_value = "json")]
    pub format: String,
}

pub fn run(args: WorkflowArgs) -> Result<()> {
    match args.command {
        WorkflowCommand::Status(a) => run_status(a),
    }
}

fn run_status(args: StatusArgs) -> Result<()> {
    let result = get_status(&args.plan_dir)?;

    if args.format == "text" {
        println!(
            "Plan: {}\nStatus: {}\nProgress: {}%",
            result.plan_dir, result.status, result.progress
        );
        println!(
            "Phases: {} total, {} completed, {} in-progress, {} pending",
            result.phases.total,
            result.phases.completed,
            result.phases.in_progress,
            result.phases.pending
        );
        println!("Reports: {}", result.reports);
        return Ok(());
    }
    println!("{}", serde_json::to_string_pretty(&result)?);
    Ok(())
}
