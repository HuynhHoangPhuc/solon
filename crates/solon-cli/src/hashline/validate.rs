use super::hash::hash_to_cid;
use anyhow::{bail, Result};

/// Parse a line reference like "5#HH" into (line_number, cid)
pub fn parse_line_ref(s: &str) -> Result<(usize, String)> {
    let Some(pos) = s.find('#') else {
        bail!("Invalid line reference '{s}': expected format LINE#HASH (e.g. 5#HH)");
    };
    let line_num: usize = s[..pos]
        .parse()
        .map_err(|_| anyhow::anyhow!("Invalid line number in reference '{s}'"))?;
    if line_num == 0 {
        bail!("Line numbers are 1-based; got 0 in reference '{s}'");
    }
    let cid = s[pos + 1..].to_owned();
    if cid.len() != 2 {
        bail!("Invalid CID in reference '{s}': expected exactly 2 characters");
    }
    Ok((line_num, cid))
}

/// Validate that line at `line_num` (1-based) in `file_lines` matches `expected_cid`.
pub fn validate_hash(file_lines: &[String], line_num: usize, expected_cid: &str) -> Result<()> {
    if line_num == 0 || line_num > file_lines.len() {
        bail!(
            "Line {line_num} is out of range (file has {} lines)",
            file_lines.len()
        );
    }
    let content = &file_lines[line_num - 1];
    let actual_cid = hash_to_cid(content, line_num);
    if actual_cid != expected_cid {
        bail!(
            "Hash mismatch at line {line_num}: expected {expected_cid}, got {actual_cid}. \
             File may have changed since last read."
        );
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::hashline::hash::hash_to_cid;

    #[test]
    fn parse_valid_ref() {
        let (n, cid) = parse_line_ref("5#HH").unwrap();
        assert_eq!(n, 5);
        assert_eq!(cid, "HH");
    }

    #[test]
    fn parse_invalid_refs() {
        assert!(parse_line_ref("noHash").is_err());
        assert!(parse_line_ref("0#ZP").is_err());
        assert!(parse_line_ref("abc#ZP").is_err());
        assert!(parse_line_ref("5#Z").is_err()); // CID too short
    }

    #[test]
    fn validate_correct_hash() {
        let lines = vec!["fn main() {".to_string()];
        let cid = hash_to_cid("fn main() {", 1);
        assert!(validate_hash(&lines, 1, &cid).is_ok());
    }

    #[test]
    fn validate_wrong_hash() {
        let lines = vec!["fn main() {".to_string()];
        let err = validate_hash(&lines, 1, "ZZ").unwrap_err();
        assert!(err.to_string().contains("Hash mismatch"));
    }

    #[test]
    fn validate_out_of_range() {
        let lines = vec!["line1".to_string()];
        assert!(validate_hash(&lines, 2, "ZP").is_err());
        assert!(validate_hash(&lines, 0, "ZP").is_err());
    }
}
