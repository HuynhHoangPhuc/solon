use anyhow::Result;
use clap::{Parser, Subcommand};

mod ast;
mod cmd;
mod hashline;
mod lsp;

#[derive(Parser)]
#[command(
    name = "sl",
    version,
    about = "Hashline read/edit, AST-grep, LSP tools for Claude Code"
)]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Read a file with hashline annotations
    Read(cmd::read::ReadArgs),
    /// Edit a file using hashline references
    Edit(cmd::edit::EditArgs),
    /// AST-grep semantic search and replace
    Ast(cmd::ast::AstArgs),
    /// LSP-based code intelligence
    Lsp(cmd::lsp::LspArgs),
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();

    match cli.command {
        Commands::Read(args) => cmd::read::run(args),
        Commands::Edit(args) => cmd::edit::run(args),
        Commands::Ast(args) => cmd::ast::run(args),
        Commands::Lsp(args) => cmd::lsp::run(args).await,
    }
}
