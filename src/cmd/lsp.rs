use anyhow::{Context, Result};
use clap::{Args, Subcommand};
use serde_json::{json, Value};
use std::path::{Path, PathBuf};

use crate::lsp::client::LspClient;
use crate::lsp::detect::{detect_server, find_project_root};
use crate::lsp::format::{format_diagnostics, format_hover, format_locations};

#[derive(Args, Debug)]
pub struct LspArgs {
    #[command(subcommand)]
    pub command: LspCommands,
}

#[derive(Subcommand, Debug)]
pub enum LspCommands {
    /// Show diagnostics (errors/warnings) for a file
    Diagnostics(DiagnosticsArgs),
    /// Go to definition at a position
    GotoDef(GotoDefArgs),
    /// Find all references at a position
    References(ReferencesArgs),
    /// Show hover info at a position
    Hover(HoverArgs),
}

#[derive(Args, Debug)]
pub struct DiagnosticsArgs {
    pub file: PathBuf,
}

#[derive(Args, Debug)]
pub struct GotoDefArgs {
    pub file: PathBuf,
    /// Line number (1-based)
    pub line: u32,
    /// Column number (1-based)
    pub col: u32,
}

#[derive(Args, Debug)]
pub struct ReferencesArgs {
    pub file: PathBuf,
    /// Line number (1-based)
    pub line: u32,
    /// Column number (1-based)
    pub col: u32,
}

#[derive(Args, Debug)]
pub struct HoverArgs {
    pub file: PathBuf,
    /// Line number (1-based)
    pub line: u32,
    /// Column number (1-based)
    pub col: u32,
}

pub async fn run(args: LspArgs) -> Result<()> {
    match args.command {
        LspCommands::Diagnostics(a) => run_diagnostics(a),
        LspCommands::GotoDef(a) => run_goto_def(a),
        LspCommands::References(a) => run_references(a),
        LspCommands::Hover(a) => run_hover(a),
    }
}

fn connect(file: &Path) -> Result<(LspClient, PathBuf)> {
    let abs_path = file
        .canonicalize()
        .with_context(|| format!("File not found: {}", file.display()))?;

    let config = detect_server(&abs_path).ok_or_else(|| {
        anyhow::anyhow!(
            "No language server configured for '{}'",
            abs_path
                .extension()
                .and_then(|e| e.to_str())
                .unwrap_or("unknown")
        )
    })?;

    let root = find_project_root(&abs_path, &config.root_markers);
    let client = LspClient::connect(&config, &root)?;
    Ok((client, abs_path))
}

/// Build a LSP position object (0-based) from 1-based line/col
fn lsp_position(line: u32, col: u32) -> Value {
    json!({ "line": line - 1, "character": col - 1 })
}

/// Build a LSP textDocumentIdentifier from an absolute path
fn text_document_id(path: &Path) -> Value {
    let uri = format!("file://{}", path.display());
    json!({ "uri": uri })
}

fn run_diagnostics(args: DiagnosticsArgs) -> Result<()> {
    let (mut client, abs_path) = connect(&args.file)?;
    client.open_document(&abs_path)?;
    let diags = client.get_diagnostics(&abs_path)?;
    let display = abs_path.display().to_string();
    print!("{}", format_diagnostics(&diags, &display));
    Ok(())
}

fn run_goto_def(args: GotoDefArgs) -> Result<()> {
    let (mut client, abs_path) = connect(&args.file)?;
    client.open_document(&abs_path)?;

    let result = client.request(
        "textDocument/definition",
        json!({
            "textDocument": text_document_id(&abs_path),
            "position": lsp_position(args.line, args.col)
        }),
    )?;

    let locations = json_to_location_list(result);
    print!("{}", format_locations(&locations, "definitions"));
    Ok(())
}

fn run_references(args: ReferencesArgs) -> Result<()> {
    let (mut client, abs_path) = connect(&args.file)?;
    client.open_document(&abs_path)?;

    let result = client.request(
        "textDocument/references",
        json!({
            "textDocument": text_document_id(&abs_path),
            "position": lsp_position(args.line, args.col),
            "context": { "includeDeclaration": true }
        }),
    )?;

    let locations = json_to_location_list(result);
    print!("{}", format_locations(&locations, "references"));
    Ok(())
}

fn run_hover(args: HoverArgs) -> Result<()> {
    let (mut client, abs_path) = connect(&args.file)?;
    client.open_document(&abs_path)?;

    let result = client.request(
        "textDocument/hover",
        json!({
            "textDocument": text_document_id(&abs_path),
            "position": lsp_position(args.line, args.col)
        }),
    )?;

    if result.is_null() {
        println!("No hover information available.");
        return Ok(());
    }

    print!("{}", format_hover(&result));
    Ok(())
}

/// Normalize LSP response: single Location or array of Locations → Vec<Value>
fn json_to_location_list(v: Value) -> Vec<Value> {
    match v {
        Value::Array(arr) => arr,
        Value::Object(_) => vec![v],
        _ => vec![],
    }
}
