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
    let max = if max_results == 0 {
        DEFAULT_MAX_RESULTS
    } else {
        max_results
    };

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
        out.push_str(&format!(
            "... ({} more matches, use --max-results to see more)\n",
            total - max
        ));
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

#[cfg(test)]
mod tests {
    use super::*;

    // --- format_search_results ---

    #[test]
    fn search_results_empty_input() {
        assert_eq!(format_search_results("", 0), "0 matches found.\n");
        assert_eq!(format_search_results("   ", 0), "0 matches found.\n");
    }

    #[test]
    fn search_results_single_ndjson_match() {
        let ndjson = r#"{"file":"src/main.rs","range":{"start":{"line":4,"column":2}},"text":"let x = 1;"}"#;
        let result = format_search_results(ndjson, 0);
        assert!(result.contains("src/main.rs:5:3: let x = 1;"));
        assert!(result.contains("1 match(es) found."));
    }

    #[test]
    fn search_results_json_array_format() {
        let json = r#"[{"file":"foo.rs","range":{"start":{"line":0,"column":0}},"text":"bar"}]"#;
        let result = format_search_results(json, 0);
        assert!(result.contains("foo.rs:1:1: bar"));
        assert!(result.contains("1 match(es) found."));
    }

    #[test]
    fn search_results_max_results_truncation() {
        // Build 3 NDJSON lines; cap at max=2
        let line = r#"{"file":"a.rs","range":{"start":{"line":0,"column":0}},"text":"x"}"#;
        let input = format!("{}\n{}\n{}", line, line, line);
        let result = format_search_results(&input, 2);
        assert!(result.contains("... (1 more matches"));
    }

    #[test]
    fn search_results_invalid_json() {
        assert_eq!(format_search_results("not json at all", 0), "0 matches found.\n");
    }

    #[test]
    fn search_results_multiline_text_shows_first_line_only() {
        let ndjson = "{\"file\":\"b.rs\",\"range\":{\"start\":{\"line\":0,\"column\":0}},\"text\":\"first line\\nsecond line\"}";
        let result = format_search_results(ndjson, 0);
        assert!(result.contains("first line"));
        assert!(!result.contains("second line"));
    }

    // --- format_replace_preview ---

    #[test]
    fn replace_preview_empty_input() {
        assert_eq!(format_replace_preview(""), "No replacements.\n");
        assert_eq!(format_replace_preview("  "), "No replacements.\n");
    }

    #[test]
    fn replace_preview_non_empty_returned_as_is() {
        let content = "some preview content\n";
        assert_eq!(format_replace_preview(content), content);
    }
}
