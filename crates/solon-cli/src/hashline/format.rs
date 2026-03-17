use super::hash::hash_to_cid;

/// Format a line with hashline annotation: `LINE#HASH|CONTENT`
pub fn format_line(line_number: usize, cid: &str, content: &str) -> String {
    format!("{line_number}#{cid}|{content}")
}

/// Format a line by computing its CID automatically
pub fn annotate_line(line_number: usize, content: &str) -> String {
    let cid = hash_to_cid(content, line_number);
    format_line(line_number, &cid, content)
}

/// Parse a hashline annotation back into (line_number, cid, content).
/// Returns None if the string is not a valid hashline.
#[allow(dead_code)]
pub fn parse_hashline(annotated: &str) -> Option<(usize, String, String)> {
    // Format: "LINE#HASH|CONTENT"
    let hash_pos = annotated.find('#')?;
    let pipe_pos = annotated.find('|')?;
    if pipe_pos < hash_pos {
        return None;
    }
    let line_number: usize = annotated[..hash_pos].parse().ok()?;
    let cid = annotated[hash_pos + 1..pipe_pos].to_owned();
    let content = annotated[pipe_pos + 1..].to_owned();
    Some((line_number, cid, content))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn format_roundtrip() {
        let formatted = format_line(1, "ZP", "fn main() {");
        assert_eq!(formatted, "1#ZP|fn main() {");
        let (n, cid, content) = parse_hashline(&formatted).unwrap();
        assert_eq!(n, 1);
        assert_eq!(cid, "ZP");
        assert_eq!(content, "fn main() {");
    }

    #[test]
    fn parse_invalid() {
        assert!(parse_hashline("not a hashline").is_none());
        assert!(parse_hashline("abc#ZP|content").is_none()); // non-numeric line
    }

    #[test]
    fn annotate_line_produces_correct_format() {
        let result = annotate_line(5, "hello");
        // Should be "5#XX|hello" where XX is deterministic CID
        let (n, _cid, content) = parse_hashline(&result).unwrap();
        assert_eq!(n, 5);
        assert_eq!(content, "hello");
    }

    #[test]
    fn content_with_pipe_preserved() {
        // Content containing | should still parse correctly (only first | is delimiter)
        let formatted = format_line(2, "AB", "a | b | c");
        let (_, _, content) = parse_hashline(&formatted).unwrap();
        assert_eq!(content, "a | b | c");
    }
}
