use anyhow::{bail, Result};

/// Supported edit operations on a line buffer
#[derive(Debug)]
pub enum EditOp {
    /// Replace lines start..=end (1-based) with new content lines
    Replace { start: usize, end: usize, content: Vec<String> },
    /// Insert content after `after` line (1-based)
    Append { after: usize, content: Vec<String> },
    /// Insert content before `before` line (1-based)
    Prepend { before: usize, content: Vec<String> },
    /// Delete lines start..=end (1-based)
    Delete { start: usize, end: usize },
}

/// Apply an edit operation to a mutable line buffer
pub fn apply_edit(lines: &mut Vec<String>, op: EditOp) -> Result<()> {
    let len = lines.len();
    match op {
        EditOp::Replace { start, end, content } => {
            if start == 0 || end == 0 || start > end {
                bail!("Invalid range {start}:{end} for replace");
            }
            if end > len {
                bail!("End line {end} exceeds file length {len}");
            }
            lines.splice((start - 1)..end, content);
        }
        EditOp::Append { after, content } => {
            if after == 0 || after > len {
                bail!("Append after line {after} is out of range (file has {len} lines)");
            }
            let insert_pos = after; // insert_pos is 0-based index = after (1-based)
            for (i, line) in content.into_iter().enumerate() {
                lines.insert(insert_pos + i, line);
            }
        }
        EditOp::Prepend { before, content } => {
            if before == 0 || before > len {
                bail!("Prepend before line {before} is out of range (file has {len} lines)");
            }
            let insert_pos = before - 1;
            for (i, line) in content.into_iter().enumerate() {
                lines.insert(insert_pos + i, line);
            }
        }
        EditOp::Delete { start, end } => {
            if start == 0 || end == 0 || start > end {
                bail!("Invalid range {start}:{end} for delete");
            }
            if end > len {
                bail!("End line {end} exceeds file length {len}");
            }
            lines.drain((start - 1)..end);
        }
    }
    Ok(())
}

/// Generate a simple unified diff between original and modified line buffers
pub fn generate_diff(original: &[String], modified: &[String], filename: &str) -> String {
    let mut output = format!("--- {filename}\n+++ {filename}\n");
    let context = 3usize;

    // Find changed regions
    let orig_len = original.len();
    let mod_len = modified.len();

    // Simple LCS-based diff using Myers diff (inline minimal version)
    let hunks = compute_hunks(original, modified, context);

    if hunks.is_empty() {
        return "(no changes)\n".to_string();
    }

    for hunk in &hunks {
        let orig_start = hunk.orig_start + 1;
        let mod_start = hunk.mod_start + 1;
        let orig_count = hunk.orig_count;
        let mod_count = hunk.mod_count;

        output.push_str(&format!(
            "@@ -{orig_start},{orig_count} +{mod_start},{mod_count} @@\n"
        ));

        for change in &hunk.changes {
            match change {
                Change::Context(line) => output.push_str(&format!(" {line}\n")),
                Change::Removed(line) => output.push_str(&format!("-{line}\n")),
                Change::Added(line) => output.push_str(&format!("+{line}\n")),
            }
        }
    }

    let _ = (orig_len, mod_len); // suppress unused warnings
    output
}

#[derive(Debug)]
enum Change {
    Context(String),
    Removed(String),
    Added(String),
}

#[derive(Debug)]
struct Hunk {
    orig_start: usize,
    mod_start: usize,
    orig_count: usize,
    mod_count: usize,
    changes: Vec<Change>,
}

/// Minimal diff: find changed lines and group into hunks with context
fn compute_hunks(original: &[String], modified: &[String], context: usize) -> Vec<Hunk> {
    // Build edit script: align lines by equality
    let n = original.len();
    let m = modified.len();

    // Simple O(n*m) LCS edit distance (sufficient for typical code files)
    // dp[i][j] = LCS length of original[..i] and modified[..j]
    let mut dp = vec![vec![0usize; m + 1]; n + 1];
    for i in 1..=n {
        for j in 1..=m {
            if original[i - 1] == modified[j - 1] {
                dp[i][j] = dp[i - 1][j - 1] + 1;
            } else {
                dp[i][j] = dp[i - 1][j].max(dp[i][j - 1]);
            }
        }
    }

    // Backtrack to get edit ops: (orig_idx or -1, mod_idx or -1, is_common)
    let mut ops: Vec<(Option<usize>, Option<usize>)> = Vec::new();
    let (mut i, mut j) = (n, m);
    while i > 0 || j > 0 {
        if i > 0 && j > 0 && original[i - 1] == modified[j - 1] {
            ops.push((Some(i - 1), Some(j - 1)));
            i -= 1;
            j -= 1;
        } else if j > 0 && (i == 0 || dp[i][j - 1] >= dp[i - 1][j]) {
            ops.push((None, Some(j - 1)));
            j -= 1;
        } else {
            ops.push((Some(i - 1), None));
            i -= 1;
        }
    }
    ops.reverse();

    // Group ops into change regions with context
    // Identify which original/modified indices are changed
    let mut changed_orig = vec![false; n];
    let mut changed_mod = vec![false; m];
    for (oi, mi) in &ops {
        if oi.is_some() && mi.is_none() {
            changed_orig[oi.unwrap()] = true;
        }
        if mi.is_some() && oi.is_none() {
            changed_mod[mi.unwrap()] = true;
        }
    }

    // Find changed ranges in original and build hunks
    let mut hunks = Vec::new();
    let mut op_idx = 0;
    let total_ops = ops.len();

    while op_idx < total_ops {
        // Skip context-only runs
        let start_op = op_idx;
        // Check if this op is a change
        let (oi, mi) = ops[op_idx];
        let is_change = oi.map(|i| changed_orig[i]).unwrap_or(false)
            || mi.map(|j| changed_mod[j]).unwrap_or(false);

        if !is_change {
            op_idx += 1;
            continue;
        }

        // Find extent of change block
        let mut block_end = op_idx;
        while block_end < total_ops {
            let (oi2, mi2) = ops[block_end];
            let chg = oi2.map(|i| changed_orig[i]).unwrap_or(false)
                || mi2.map(|j| changed_mod[j]).unwrap_or(false);
            if chg {
                block_end += 1;
            } else {
                break;
            }
        }

        // Add context before: go back up to `context` ops from start_op
        let ctx_before_start = start_op.saturating_sub(context);
        let ctx_after_end = (block_end + context).min(total_ops);

        let hunk_ops = &ops[ctx_before_start..ctx_after_end];
        let mut changes = Vec::new();
        let mut orig_start = usize::MAX;
        let mut mod_start = usize::MAX;
        let mut orig_count = 0;
        let mut mod_count = 0;

        for (oi, mi) in hunk_ops {
            match (oi, mi) {
                (Some(i), Some(_)) => {
                    // Common line
                    if orig_start == usize::MAX { orig_start = *i; }
                    if mod_start == usize::MAX { mod_start = mi.unwrap(); }
                    changes.push(Change::Context(original[*i].clone()));
                    orig_count += 1;
                    mod_count += 1;
                }
                (Some(i), None) => {
                    if orig_start == usize::MAX { orig_start = *i; }
                    changes.push(Change::Removed(original[*i].clone()));
                    orig_count += 1;
                }
                (None, Some(j)) => {
                    if mod_start == usize::MAX { mod_start = *j; }
                    changes.push(Change::Added(modified[*j].clone()));
                    mod_count += 1;
                }
                (None, None) => {}
            }
        }

        if orig_start == usize::MAX { orig_start = 0; }
        if mod_start == usize::MAX { mod_start = 0; }

        hunks.push(Hunk { orig_start, mod_start, orig_count, mod_count, changes });
        op_idx = block_end;
    }

    hunks
}

/// Parse content string: interpret `\n`, `\t`, `\\` escape sequences
pub fn parse_content_escapes(s: &str) -> Vec<String> {
    let mut result = String::new();
    let mut chars = s.chars().peekable();
    while let Some(c) = chars.next() {
        if c == '\\' {
            match chars.peek() {
                Some('n') => { chars.next(); result.push('\n'); }
                Some('t') => { chars.next(); result.push('\t'); }
                Some('\\') => { chars.next(); result.push('\\'); }
                _ => result.push('\\'),
            }
        } else {
            result.push(c);
        }
    }
    result.lines().map(str::to_owned).collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    fn lines(v: &[&str]) -> Vec<String> {
        v.iter().map(|s| s.to_string()).collect()
    }

    #[test]
    fn replace_single_line() {
        let mut buf = lines(&["a", "b", "c"]);
        apply_edit(&mut buf, EditOp::Replace {
            start: 2, end: 2, content: lines(&["B"])
        }).unwrap();
        assert_eq!(buf, lines(&["a", "B", "c"]));
    }

    #[test]
    fn replace_range() {
        let mut buf = lines(&["a", "b", "c", "d"]);
        apply_edit(&mut buf, EditOp::Replace {
            start: 2, end: 3, content: lines(&["X", "Y", "Z"])
        }).unwrap();
        assert_eq!(buf, lines(&["a", "X", "Y", "Z", "d"]));
    }

    #[test]
    fn append_after_line() {
        let mut buf = lines(&["a", "b", "c"]);
        apply_edit(&mut buf, EditOp::Append {
            after: 1, content: lines(&["new"])
        }).unwrap();
        assert_eq!(buf, lines(&["a", "new", "b", "c"]));
    }

    #[test]
    fn prepend_before_line() {
        let mut buf = lines(&["a", "b", "c"]);
        apply_edit(&mut buf, EditOp::Prepend {
            before: 2, content: lines(&["new"])
        }).unwrap();
        assert_eq!(buf, lines(&["a", "new", "b", "c"]));
    }

    #[test]
    fn delete_range() {
        let mut buf = lines(&["a", "b", "c", "d"]);
        apply_edit(&mut buf, EditOp::Delete { start: 2, end: 3 }).unwrap();
        assert_eq!(buf, lines(&["a", "d"]));
    }

    #[test]
    fn parse_escape_sequences() {
        let v = parse_content_escapes("line1\\nline2\\nline3");
        assert_eq!(v, vec!["line1", "line2", "line3"]);
    }

    #[test]
    fn replace_out_of_range_errors() {
        let mut buf = lines(&["a"]);
        assert!(apply_edit(&mut buf, EditOp::Replace {
            start: 1, end: 5, content: lines(&["x"])
        }).is_err());
    }
}
