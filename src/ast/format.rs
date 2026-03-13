use serde::Deserialize;

const DEFAULT_MAX_RESULTS: usize = 50;

/// Minimal representation of a sg JSON match result
#[derive(Deserialize, Debug)]
struct SgMatch {
    #[serde(default)]
    file: String,
    range: SgRange,
    text: String,
}

#[derive(Deserialize, Debug)]
struct SgRange {
    start: SgPos,
}

#[derive(Deserialize, Debug)]
struct SgPos {
    line: usize,
    column: usize,
}

/// Format sg JSON output as concise `file:line:col: matched_code` lines
pub fn format_search_results(json_output: &str, max_results: usize) -> String {
    let max = if max_results == 0 { DEFAULT_MAX_RESULTS } else { max_results };

    if json_output.trim().is_empty() {
        return "0 matches found.\n".to_string();
    }

    // sg outputs one JSON object per line (NDJSON) or a JSON array
    let matches: Vec<SgMatch> = if json_output.trim_start().starts_with('[') {
        serde_json::from_str(json_output).unwrap_or_default()
    } else {
        json_output
            .lines()
            .filter(|l| !l.trim().is_empty())
            .filter_map(|l| serde_json::from_str(l).ok())
            .collect()
    };

    if matches.is_empty() {
        return "0 matches found.\n".to_string();
    }

    let total = matches.len();
    let displayed = matches.len().min(max);
    let mut out = String::new();

    for m in &matches[..displayed] {
        let line = m.range.start.line + 1; // sg is 0-based
        let col = m.range.start.column + 1;
        let snippet = m.text.lines().next().unwrap_or("").trim_end();
        out.push_str(&format!("{}:{}:{}: {}\n", m.file, line, col, snippet));
    }

    if total > max {
        out.push_str(&format!("... ({} more matches, use --max-results to see more)\n", total - max));
    } else {
        out.push_str(&format!("{total} match(es) found.\n"));
    }

    out
}

/// Format sg replace preview output
pub fn format_replace_preview(json_output: &str) -> String {
    if json_output.trim().is_empty() {
        return "No replacements.\n".to_string();
    }
    json_output.to_owned()
}
