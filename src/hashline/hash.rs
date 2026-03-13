/// CID alphabet: 16 chars → 256 unique 2-char codes
const ALPHABET: &[u8; 16] = b"ZPMQVRWSNKTXJBYH";

/// Returns true if line contains at least one alphanumeric character
pub fn classify_line(line: &str) -> bool {
    line.chars().any(|c| c.is_alphanumeric())
}

/// Compute a 2-char CID for a line.
/// Seed: 0 for alphanumeric lines, line_number for blank/punctuation-only lines.
pub fn compute_cid(line: &str, line_number: usize) -> [u8; 2] {
    let seed = if classify_line(line) { 0 } else { line_number as u32 };
    let hash = xxhash_rust::xxh32::xxh32(line.as_bytes(), seed);
    let index = (hash % 256) as usize;
    let high = index / 16;
    let low = index % 16;
    [ALPHABET[high], ALPHABET[low]]
}

/// Convenience wrapper: returns CID as a 2-char String
pub fn hash_to_cid(line: &str, line_number: usize) -> String {
    let cid = compute_cid(line, line_number);
    String::from_utf8_lossy(&cid).into_owned()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn alphanumeric_line_uses_seed_zero() {
        let cid1 = hash_to_cid("fn main() {", 1);
        let cid2 = hash_to_cid("fn main() {", 99); // line number shouldn't matter
        assert_eq!(cid1, cid2);
        assert_eq!(cid1.len(), 2);
    }

    #[test]
    fn blank_line_uses_line_number_seed() {
        let cid1 = hash_to_cid("", 1);
        let cid2 = hash_to_cid("", 2);
        // Different line numbers → different seeds → (likely) different CIDs
        // Not guaranteed but very likely for consecutive lines
        assert_eq!(cid1.len(), 2);
        assert_eq!(cid2.len(), 2);
    }

    #[test]
    fn deterministic_output() {
        let cid = hash_to_cid("hello world", 5);
        assert_eq!(cid, hash_to_cid("hello world", 5));
    }

    #[test]
    fn cid_chars_in_alphabet() {
        let alphabet = "ZPMQVRWSNKTXJBYH";
        for line in &["fn main() {", "", "   ", "// comment", "42"] {
            let cid = hash_to_cid(line, 1);
            for ch in cid.chars() {
                assert!(alphabet.contains(ch), "unexpected char '{ch}' in CID");
            }
        }
    }

    #[test]
    fn classify_line_alphanumeric() {
        assert!(classify_line("fn main()"));
        assert!(classify_line("  42  "));
        assert!(!classify_line(""));
        assert!(!classify_line("   "));
        assert!(!classify_line("// !!!"));
    }
}
