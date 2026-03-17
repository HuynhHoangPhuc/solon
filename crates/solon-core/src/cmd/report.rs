/// Report management CLI commands: index.
/// Ported from Go solon-core/cmd/report*.go
use crate::report::index_reports;
use anyhow::Result;
use clap::{Args, Subcommand};

#[derive(Args)]
pub struct ReportArgs {
    #[command(subcommand)]
    pub command: ReportCommand,
}

#[derive(Subcommand)]
pub enum ReportCommand {
    /// List all report files in a plan directory
    Index(IndexArgs),
}

#[derive(Args)]
pub struct IndexArgs {
    /// Plan directory path
    pub plan_dir: String,
    /// Output format: json or text
    #[arg(long, default_value = "json")]
    pub format: String,
}

pub fn run(args: ReportArgs) -> Result<()> {
    match args.command {
        ReportCommand::Index(a) => run_index(a),
    }
}

fn run_index(args: IndexArgs) -> Result<()> {
    let entries = index_reports(&args.plan_dir)?;

    if args.format == "text" {
        if entries.is_empty() {
            println!("No reports found.");
            return Ok(());
        }
        for e in &entries {
            println!("[{}] {}", e.directory, e.filename);
        }
        return Ok(());
    }
    println!("{}", serde_json::to_string_pretty(&entries)?);
    Ok(())
}
