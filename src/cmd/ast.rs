use anyhow::Result;
use clap::{Args, Subcommand};
use std::path::PathBuf;
use std::time::Duration;

use crate::ast::format::{format_replace_preview, format_search_results};
use crate::ast::sg::require_sg;

const DEFAULT_TIMEOUT_SECS: u64 = 30;
const DEFAULT_MAX_RESULTS: usize = 50;

#[derive(Args, Debug)]
pub struct AstArgs {
    #[command(subcommand)]
    pub command: AstCommands,
}

#[derive(Subcommand, Debug)]
pub enum AstCommands {
    /// Semantic search using ast-grep patterns
    Search(SearchArgs),
    /// Semantic replace using ast-grep patterns
    Replace(ReplaceArgs),
}

#[derive(Args, Debug)]
pub struct SearchArgs {
    /// ast-grep pattern (e.g. "fn $NAME($$$ARGS)")
    pub pattern: String,

    /// Language to search (e.g. rust, typescript, python)
    #[arg(long, short)]
    pub lang: Option<String>,

    /// Directory or file to search (default: current dir)
    #[arg(long, short, default_value = ".")]
    pub path: PathBuf,

    /// Output raw JSON from sg
    #[arg(long)]
    pub json: bool,

    /// Maximum results to display
    #[arg(long, default_value_t = DEFAULT_MAX_RESULTS)]
    pub max_results: usize,

    /// Search timeout in seconds
    #[arg(long, default_value_t = DEFAULT_TIMEOUT_SECS)]
    pub timeout: u64,
}

#[derive(Args, Debug)]
pub struct ReplaceArgs {
    /// ast-grep pattern to match
    pub pattern: String,

    /// Replacement pattern
    pub replacement: String,

    /// Language (e.g. rust, typescript, python)
    #[arg(long, short)]
    pub lang: Option<String>,

    /// Directory or file to process (default: current dir)
    #[arg(long, short, default_value = ".")]
    pub path: PathBuf,

    /// Apply changes without confirmation
    #[arg(long)]
    pub update_all: bool,

    /// Search timeout in seconds
    #[arg(long, default_value_t = DEFAULT_TIMEOUT_SECS)]
    pub timeout: u64,
}

pub fn run(args: AstArgs) -> Result<()> {
    match args.command {
        AstCommands::Search(a) => run_search(a),
        AstCommands::Replace(a) => run_replace(a),
    }
}

fn run_search(args: SearchArgs) -> Result<()> {
    let sg = require_sg()?;
    let timeout = Duration::from_secs(args.timeout);

    let mut sg_args: Vec<&str> = vec!["run", "--pattern", &args.pattern, "--json"];
    let lang_str;
    if let Some(ref l) = args.lang {
        sg_args.push("--lang");
        lang_str = l.clone();
        sg_args.push(&lang_str);
    }
    let path_str = args.path.to_string_lossy().into_owned();
    sg_args.push(&path_str);

    let output = crate::ast::sg::run_sg(&sg, &sg_args, timeout)?;

    if args.json {
        print!("{output}");
    } else {
        print!("{}", format_search_results(&output, args.max_results));
    }
    Ok(())
}

fn run_replace(args: ReplaceArgs) -> Result<()> {
    let sg = require_sg()?;
    let timeout = Duration::from_secs(args.timeout);

    let mut sg_args: Vec<&str> = vec!["run", "--pattern", &args.pattern, "--rewrite", &args.replacement];
    let lang_str;
    if let Some(ref l) = args.lang {
        sg_args.push("--lang");
        lang_str = l.clone();
        sg_args.push(&lang_str);
    }
    if args.update_all {
        sg_args.push("--update-all");
    }
    let path_str = args.path.to_string_lossy().into_owned();
    sg_args.push(&path_str);

    let output = crate::ast::sg::run_sg(&sg, &sg_args, timeout)?;
    print!("{}", format_replace_preview(&output));
    Ok(())
}
