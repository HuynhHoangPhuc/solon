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
    diags.iter().map(|d| format_diagnostic_value(d, file_path)).collect()
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
    locs.iter().map(|l| format!("{}\n", format_location(l))).collect()
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
        return arr.iter().filter_map(|v| {
            v.as_str()
                .map(str::to_string)
                .or_else(|| v.get("value").and_then(Value::as_str).map(str::to_string))
        }).collect::<Vec<_>>().join("\n");
    }
    "(no hover content)".to_string()
}
