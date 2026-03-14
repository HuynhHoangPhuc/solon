use serde_json::Value;

/// Format a diagnostic value as `file:line:col: severity: message`
pub fn format_diagnostic_value(diag: &Value, file_path: &str) -> String {
    let line = diag["range"]["start"]["line"].as_u64().unwrap_or(0) + 1;
    let col = diag["range"]["start"]["character"].as_u64().unwrap_or(0) + 1;
    let severity = match diag["severity"].as_u64() {
        Some(1) => "error",
        Some(2) => "warning",
        Some(3) => "info",
        Some(4) => "hint",
        _ => "unknown",
    };
    let message = diag["message"].as_str().unwrap_or("(no message)");
    format!("{file_path}:{line}:{col}: {severity}: {message}\n")
}

/// Format a list of diagnostics from raw JSON array
pub fn format_diagnostics(diags: &[Value], file_path: &str) -> String {
    if diags.is_empty() {
        return format!("{file_path}: No diagnostics.\n");
    }
    diags
        .iter()
        .map(|d| format_diagnostic_value(d, file_path))
        .collect()
}

/// Format a location value as `file:line:col`
pub fn format_location(loc: &Value) -> String {
    let uri = loc["uri"].as_str().unwrap_or("");
    // Strip "file://" prefix for display
    let path = uri.strip_prefix("file://").unwrap_or(uri);
    let line = loc["range"]["start"]["line"].as_u64().unwrap_or(0) + 1;
    let col = loc["range"]["start"]["character"].as_u64().unwrap_or(0) + 1;
    format!("{path}:{line}:{col}")
}

/// Format a list of location values
pub fn format_locations(locs: &[Value], label: &str) -> String {
    if locs.is_empty() {
        return format!("No {label} found.\n");
    }
    locs.iter()
        .map(|l| format!("{}\n", format_location(l)))
        .collect()
}

/// Format hover value, truncating long content
pub fn format_hover(hover: &Value) -> String {
    const MAX_LEN: usize = 500;
    let text = extract_hover_text(hover);
    if text.len() > MAX_LEN {
        format!("{}...\n", &text[..MAX_LEN])
    } else {
        format!("{text}\n")
    }
}

fn extract_hover_text(hover: &Value) -> String {
    let contents = &hover["contents"];
    if let Some(s) = contents.as_str() {
        return s.to_string();
    }
    if let Some(obj) = contents.as_object() {
        if let Some(value) = obj.get("value").and_then(Value::as_str) {
            return value.to_string();
        }
    }
    if let Some(arr) = contents.as_array() {
        return arr
            .iter()
            .filter_map(|v| {
                v.as_str()
                    .map(str::to_string)
                    .or_else(|| v.get("value").and_then(Value::as_str).map(str::to_string))
            })
            .collect::<Vec<_>>()
            .join("\n");
    }
    "(no hover content)".to_string()
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    // --- format_diagnostic_value ---

    #[test]
    fn diagnostic_value_error_severity() {
        let diag =
            json!({"range":{"start":{"line":2,"character":4}},"severity":1,"message":"type error"});
        let result = format_diagnostic_value(&diag, "src/main.rs");
        assert_eq!(result, "src/main.rs:3:5: error: type error\n");
    }

    #[test]
    fn diagnostic_value_warning_severity() {
        let diag =
            json!({"range":{"start":{"line":0,"character":0}},"severity":2,"message":"unused var"});
        let result = format_diagnostic_value(&diag, "lib.rs");
        assert!(result.contains("warning"));
    }

    #[test]
    fn diagnostic_value_info_and_hint() {
        let info_diag =
            json!({"range":{"start":{"line":0,"character":0}},"severity":3,"message":"info"});
        assert!(format_diagnostic_value(&info_diag, "f.rs").contains("info"));

        let hint_diag =
            json!({"range":{"start":{"line":0,"character":0}},"severity":4,"message":"hint"});
        assert!(format_diagnostic_value(&hint_diag, "f.rs").contains("hint"));
    }

    #[test]
    fn diagnostic_value_unknown_severity() {
        let diag = json!({"range":{"start":{"line":0,"character":0}},"severity":99,"message":"x"});
        assert!(format_diagnostic_value(&diag, "f.rs").contains("unknown"));
    }

    #[test]
    fn diagnostic_value_missing_fields_use_defaults() {
        // Missing range and message — defaults: line 1, col 1, "(no message)"
        let diag = json!({});
        let result = format_diagnostic_value(&diag, "f.rs");
        assert!(result.contains("f.rs:1:1:"));
        assert!(result.contains("(no message)"));
    }

    // --- format_diagnostics ---

    #[test]
    fn diagnostics_empty_array() {
        let result = format_diagnostics(&[], "main.rs");
        assert_eq!(result, "main.rs: No diagnostics.\n");
    }

    #[test]
    fn diagnostics_multiple_entries_concatenated() {
        let diags = vec![
            json!({"range":{"start":{"line":0,"character":0}},"severity":1,"message":"err1"}),
            json!({"range":{"start":{"line":1,"character":0}},"severity":2,"message":"warn1"}),
        ];
        let result = format_diagnostics(&diags, "f.rs");
        assert!(result.contains("error: err1"));
        assert!(result.contains("warning: warn1"));
    }

    // --- format_location ---

    #[test]
    fn location_strips_file_prefix() {
        let loc = json!({"uri":"file:///home/user/proj/src/lib.rs","range":{"start":{"line":9,"character":3}}});
        let result = format_location(&loc);
        assert_eq!(result, "/home/user/proj/src/lib.rs:10:4");
    }

    #[test]
    fn location_without_file_prefix_preserved() {
        let loc = json!({"uri":"/abs/path/foo.rs","range":{"start":{"line":0,"character":0}}});
        let result = format_location(&loc);
        assert_eq!(result, "/abs/path/foo.rs:1:1");
    }

    #[test]
    fn location_zero_based_to_one_based_conversion() {
        let loc = json!({"uri":"file:///f.rs","range":{"start":{"line":4,"character":7}}});
        let result = format_location(&loc);
        assert_eq!(result, "/f.rs:5:8");
    }

    // --- format_locations ---

    #[test]
    fn locations_empty_returns_label_message() {
        let result = format_locations(&[], "definitions");
        assert_eq!(result, "No definitions found.\n");
    }

    #[test]
    fn locations_multiple_entries_newline_separated() {
        let locs = vec![
            json!({"uri":"file:///a.rs","range":{"start":{"line":0,"character":0}}}),
            json!({"uri":"file:///b.rs","range":{"start":{"line":1,"character":2}}}),
        ];
        let result = format_locations(&locs, "references");
        assert!(result.contains("/a.rs:1:1\n"));
        assert!(result.contains("/b.rs:2:3\n"));
    }

    // --- format_hover ---

    #[test]
    fn hover_string_contents() {
        let hover = json!({"contents": "let x: i32"});
        assert_eq!(format_hover(&hover), "let x: i32\n");
    }

    #[test]
    fn hover_object_with_value_field() {
        let hover = json!({"contents": {"kind": "markdown", "value": "## Docs\nsome text"}});
        let result = format_hover(&hover);
        assert!(result.contains("## Docs"));
    }

    #[test]
    fn hover_array_of_strings_and_objects_joined() {
        let hover = json!({"contents": ["plain string", {"value": "obj value"}]});
        let result = format_hover(&hover);
        assert!(result.contains("plain string"));
        assert!(result.contains("obj value"));
    }

    #[test]
    fn hover_empty_or_unknown_returns_placeholder() {
        let hover = json!({});
        assert!(format_hover(&hover).contains("(no hover content)"));
    }

    #[test]
    fn hover_long_content_truncated() {
        let long = "x".repeat(600);
        let hover = json!({"contents": long});
        let result = format_hover(&hover);
        assert!(result.ends_with("...\n"));
        // Result should be 500 chars + "...\n" = 505 chars
        assert!(result.len() < 600);
    }
}
