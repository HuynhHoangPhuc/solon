use anyhow::{bail, Context, Result};
use clap::Args;
use std::fs;
use std::io::{self, Read};
use std::path::PathBuf;

use crate::hashline::canonicalize::{
    detect_line_ending, is_binary, normalize_line_endings, restore_line_endings, strip_bom,
};
use crate::hashline::edit::{apply_edit, generate_diff, parse_content_escapes, EditOp};
use crate::hashline::validate::{parse_line_ref, validate_hash};

#[derive(Args, Debug)]
pub struct EditArgs {
    /// File to edit
    pub file: PathBuf,

    /// Start line reference (e.g. 5#HH). For --after/--before/--delete, this is the target.
    pub start_ref: String,

    /// Second positional arg: END_REF (e.g. 10#QQ) for range ops, or CONTENT for single-line
    pub second: Option<String>,

    /// Third positional arg: CONTENT (when second is END_REF)
    pub content: Option<String>,

    /// Insert content AFTER the referenced line
    #[arg(long, conflicts_with_all = ["before", "delete"])]
    pub after: bool,

    /// Insert content BEFORE the referenced line
    #[arg(long, conflicts_with_all = ["after", "delete"])]
    pub before: bool,

    /// Delete the referenced line(s) instead of replacing
    #[arg(long, conflicts_with_all = ["after", "before"])]
    pub delete: bool,

    /// Read content from stdin instead of positional argument
    #[arg(long)]
    pub stdin: bool,

    /// Skip creating a .bak backup file
    #[arg(long)]
    pub no_backup: bool,
}

/// Returns true if the string looks like a line reference: digits followed by #XX
fn is_line_ref(s: &str) -> bool {
    if let Some(pos) = s.find('#') {
        let before = &s[..pos];
        let after = &s[pos + 1..];
        before.parse::<usize>().is_ok()
            && after.len() == 2
            && after.chars().all(|c| c.is_ascii_uppercase())
    } else {
        false
    }
}

pub fn run(args: EditArgs) -> Result<()> {
    let path = &args.file;

    if !path.exists() {
        bail!("File not found: {}", path.display());
    }

    // Disambiguate `second` as either END_REF or CONTENT based on pattern
    let (end_ref_str, content_arg): (Option<String>, Option<String>) = match args.second {
        None => (None, None),
        Some(ref s) if is_line_ref(s) => (Some(s.clone()), args.content.clone()),
        Some(ref s) => (None, Some(s.clone())),
    };

    // Read file
    let raw = fs::read(path).with_context(|| format!("Failed to read {}", path.display()))?;
    if is_binary(&raw) {
        bail!("{} appears to be a binary file", path.display());
    }
    let stripped = strip_bom(&raw);
    let file_str = std::str::from_utf8(stripped)
        .map_err(|_| anyhow::anyhow!("{} is not valid UTF-8", path.display()))?;

    let line_ending = detect_line_ending(file_str);
    let normalized = normalize_line_endings(file_str);
    let original: Vec<String> = normalized.lines().map(str::to_owned).collect();
    let mut modified = original.clone();

    // Parse and validate start ref
    let (start_line, start_cid) = parse_line_ref(&args.start_ref)?;
    validate_hash(&original, start_line, &start_cid)?;

    // Resolve end line
    let end_line = if let Some(ref end_ref) = end_ref_str {
        let (en, ec) = parse_line_ref(end_ref)?;
        validate_hash(&original, en, &ec)?;
        en
    } else {
        start_line
    };

    // Resolve new content
    let new_content: Vec<String> = if args.delete {
        vec![]
    } else {
        let raw_content = if args.stdin {
            let mut buf = String::new();
            io::stdin().read_to_string(&mut buf)?;
            buf
        } else {
            content_arg.unwrap_or_default()
        };
        parse_content_escapes(&raw_content)
    };

    // Build and apply edit operation
    let op = if args.delete {
        EditOp::Delete {
            start: start_line,
            end: end_line,
        }
    } else if args.after {
        EditOp::Append {
            after: start_line,
            content: new_content,
        }
    } else if args.before {
        EditOp::Prepend {
            before: start_line,
            content: new_content,
        }
    } else {
        EditOp::Replace {
            start: start_line,
            end: end_line,
            content: new_content,
        }
    };

    apply_edit(&mut modified, op)?;

    let diff = generate_diff(&original, &modified, &path.display().to_string());

    // Optional backup
    if !args.no_backup {
        let bak_path = path.with_extension(format!(
            "{}.bak",
            path.extension().and_then(|e| e.to_str()).unwrap_or("")
        ));
        fs::copy(path, &bak_path)
            .with_context(|| format!("Failed to create backup at {}", bak_path.display()))?;
    }

    // Atomic write: temp file → rename
    let mut new_content_str = modified.join("\n");
    if file_str.ends_with('\n') {
        new_content_str.push('\n');
    }
    let output = restore_line_endings(&new_content_str, line_ending);

    let tmp_path = path.with_extension("tmp");
    fs::write(&tmp_path, output.as_bytes())
        .with_context(|| format!("Failed to write temp file {}", tmp_path.display()))?;
    fs::rename(&tmp_path, path)
        .with_context(|| format!("Failed to rename temp file to {}", path.display()))?;

    print!("{diff}");
    Ok(())
}
