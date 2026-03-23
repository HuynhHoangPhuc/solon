use anyhow::Result;
use clap::{Parser, Subcommand};

#[derive(Parser)]
#[command(name = "sl", version, about = "Solon CLI for Claude Code")]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Read a file with hashline annotations
    Read(solon_cli::cmd::read::ReadArgs),
    /// Edit a file using hashline references
    Edit(solon_cli::cmd::edit::EditArgs),
    /// AST-grep semantic search and replace
    Ast(solon_cli::cmd::ast::AstArgs),
    /// LSP-based code intelligence
    Lsp(solon_cli::cmd::lsp::LspArgs),
    /// Plan management (scaffold, resolve, validate, red-team, archive)
    Plan(solon_core::cmd::plan::PlanArgs),
    /// Task management (hydrate, sync)
    Task(solon_core::cmd::task::TaskArgs),
    /// Workflow status tracking
    Workflow(solon_core::cmd::workflow::WorkflowArgs),
    /// Report indexing
    Report(solon_core::cmd::report::ReportArgs),
    /// Skill management (create, validate, catalog)
    Skill(solon_core::cmd::skill::SkillArgs),
    /// Hook event handlers
    #[command(subcommand)]
    Hook(solon_core::cmd::hook::HookCommand),
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();
    match cli.command {
        Commands::Read(args) => solon_cli::cmd::read::run(args),
        Commands::Edit(args) => solon_cli::cmd::edit::run(args),
        Commands::Ast(args) => solon_cli::cmd::ast::run(args),
        Commands::Lsp(args) => solon_cli::cmd::lsp::run(args).await,
        Commands::Plan(args) => solon_core::cmd::plan::run(args),
        Commands::Task(args) => solon_core::cmd::task::run(args),
        Commands::Workflow(args) => solon_core::cmd::workflow::run(args),
        Commands::Report(args) => solon_core::cmd::report::run(args),
        Commands::Skill(args) => solon_core::cmd::skill::run(args),
        Commands::Hook(args) => solon_core::cmd::hook::run(args),
    }
}
