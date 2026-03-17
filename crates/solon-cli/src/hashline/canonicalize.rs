/// Line ending style detected in a file
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum LineEnding {
    Lf,
    CrLf,
    Mixed,
}

/// Strip UTF-8 BOM (EF BB BF) if present
pub fn strip_bom(content: &[u8]) -> &[u8] {
    if content.starts_with(&[0xEF, 0xBB, 0xBF]) {
        &content[3..]
    } else {
        content
    }
}

/// Detect the dominant line ending style
pub fn detect_line_ending(content: &str) -> LineEnding {
    let crlf_count = content.matches("\r\n").count();
    let lf_count = content.matches('\n').count() - crlf_count;
    match (crlf_count, lf_count) {
        (0, _) => LineEnding::Lf,
        (_, 0) => LineEnding::CrLf,
        _ => LineEnding::Mixed,
    }
}

/// Normalize CRLF → LF for internal processing
pub fn normalize_line_endings(content: &str) -> String {
    content.replace("\r\n", "\n")
}

/// Restore original line endings before writing back
pub fn restore_line_endings(content: &str, style: LineEnding) -> String {
    match style {
        LineEnding::CrLf => content.replace('\n', "\r\n"),
        LineEnding::Lf | LineEnding::Mixed => content.to_owned(),
    }
}

/// Check if the byte slice looks like binary (contains null bytes or high ratio of non-UTF8)
pub fn is_binary(data: &[u8]) -> bool {
    // Null byte is a reliable binary indicator
    if data.iter().take(8192).any(|&b| b == 0) {
        return true;
    }
    // Try UTF-8 decode; if it fails, treat as binary
    std::str::from_utf8(data).is_err()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn strips_bom() {
        let bom: &[u8] = &[0xEF, 0xBB, 0xBF, b'h', b'i'];
        assert_eq!(strip_bom(bom), b"hi");
    }

    #[test]
    fn no_bom_unchanged() {
        let data = b"hello";
        assert_eq!(strip_bom(data), b"hello");
    }

    #[test]
    fn normalizes_crlf() {
        let s = "line1\r\nline2\r\n";
        assert_eq!(normalize_line_endings(s), "line1\nline2\n");
    }

    #[test]
    fn detects_crlf() {
        assert_eq!(detect_line_ending("a\r\nb\r\n"), LineEnding::CrLf);
        assert_eq!(detect_line_ending("a\nb\n"), LineEnding::Lf);
    }

    #[test]
    fn restores_crlf() {
        let normalized = "a\nb\n";
        assert_eq!(
            restore_line_endings(normalized, LineEnding::CrLf),
            "a\r\nb\r\n"
        );
        assert_eq!(restore_line_endings(normalized, LineEnding::Lf), "a\nb\n");
    }

    #[test]
    fn binary_detection() {
        assert!(is_binary(b"hello\x00world"));
        assert!(!is_binary(b"hello world\n"));
        assert!(is_binary(b"\xff\xfe")); // invalid UTF-8
    }
}
