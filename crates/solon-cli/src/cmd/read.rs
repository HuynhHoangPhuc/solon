use anyhow::{bail, Context, Result};
use clap::Args;
use std::fs;
use std::path::PathBuf;

use crate::hashline::canonicalize::{is_binary, normalize_line_endings, strip_bom};
use crate::hashline::format::annotate_line;

const DEFAULT_CHUNK_SIZE: usize = 200;

#[derive(Args, Debug)]
pub struct ReadArgs {
    /// File to read
    pub file: PathBuf,

    /// Line range to display, e.g. 5:20 or 5: for line 5 to EOF
    #[arg(long, value_name = "N:M")]
    pub lines: Option<String>,

    /// Maximum lines per output chunk (default: 200)
    #[arg(long, default_value_t = DEFAULT_CHUNK_SIZE)]
    pub chunk_size: usize,
}

/// Parse a line range like "5:20", "5:", ":20", ":" into (start, end)
/// Returns 1-based inclusive start and optional inclusive end.
fn parse_range(s: &str) -> Result<(usize, Option<usize>)> {
    let parts: Vec<&str> = s.splitn(2, ':').collect();
    if parts.len() != 2 {
        bail!("Invalid range '{s}': expected format N:M or N:");
    }
    let start = if parts[0].is_empty() {
        1
    } else {
        parts[0]
            .parse::<usize>()
            .map_err(|_| anyhow::anyhow!("Invalid start line '{}'", parts[0]))?
    };
    let end = if parts[1].is_empty() {
        None
    } else {
        Some(
            parts[1]
                .parse::<usize>()
                .map_err(|_| anyhow::anyhow!("Invalid end line '{}'", parts[1]))?,
        )
    };
    Ok((start, end))
}

pub fn run(args: ReadArgs) -> Result<()> {
    let path = &args.file;

    if !path.exists() {
        bail!("File not found: {}", path.display());
    }
    if path.is_dir() {
        bail!("{} is a directory, not a file", path.display());
    }

    let raw = fs::read(path).with_context(|| format!("Failed to read {}", path.display()))?;

    if is_binary(&raw) {
        bail!("{} appears to be a binary file", path.display());
    }

    let stripped = strip_bom(&raw);
    let content = std::str::from_utf8(stripped)
        .map_err(|_| anyhow::anyhow!("{} is not valid UTF-8", path.display()))?;
    let normalized = normalize_line_endings(content);

    let all_lines: Vec<&str> = normalized.lines().collect();
    let total = all_lines.len();

    let (start, end_opt) = if let Some(ref range_str) = args.lines {
        parse_range(range_str)?
    } else {
        (1, None)
    };

    let end = end_opt.unwrap_or(total).min(total);

    if start > total {
        bail!("Start line {start} exceeds file length {total}");
    }
    if start > end {
        bail!("Start line {start} is greater than end line {end}");
    }

    let selected_lines = &all_lines[start - 1..end];

    if selected_lines.len() > args.chunk_size {
        eprintln!(
            "Warning: output truncated to {} lines (file has {} lines total in range). \
             Use --lines to read specific ranges.",
            args.chunk_size,
            selected_lines.len()
        );
    }

    let display_lines = &selected_lines[..selected_lines.len().min(args.chunk_size)];

    for (offset, &line) in display_lines.iter().enumerate() {
        let line_num = start + offset;
        println!("{}", annotate_line(line_num, line));
    }

    Ok(())
}
