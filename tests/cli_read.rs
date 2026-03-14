/// Integration tests for `sl read` using assert_cmd.
use assert_cmd::Command;
use predicates::prelude::*;
use std::io::Write;
use tempfile::NamedTempFile;

fn sl() -> Command {
    Command::cargo_bin("sl").expect("sl binary not found")
}

/// Write content to a named temp file and return the file handle (keep alive for test duration).
fn make_temp_file(content: &[u8]) -> NamedTempFile {
    let mut f = NamedTempFile::new().unwrap();
    f.write_all(content).unwrap();
    f.flush().unwrap();
    f
}

// ---------------------------------------------------------------------------
// Test 1: simple file produces LINE#HASH|CONTENT format on every line
// ---------------------------------------------------------------------------
#[test]
fn read_simple_file_produces_hashline_format() {
    let f = make_temp_file(b"hello\nworld\nfoo\n");
    sl().args(["read", f.path().to_str().unwrap()])
        .assert()
        .success()
        .stdout(predicate::str::contains("#").and(predicate::str::contains("|")));
}

// ---------------------------------------------------------------------------
// Test 2: --lines 2:4 returns only lines 2-4
// ---------------------------------------------------------------------------
#[test]
fn read_line_range_returns_correct_slice() {
    let f = make_temp_file(b"line1\nline2\nline3\nline4\nline5\n");
    let output = sl()
        .args(["read", f.path().to_str().unwrap(), "--lines", "2:4"])
        .assert()
        .success()
        .get_output()
        .stdout
        .clone();
    let text = String::from_utf8_lossy(&output);
    let lines: Vec<&str> = text.lines().collect();
    assert_eq!(
        lines.len(),
        3,
        "expected 3 lines, got {}: {text}",
        lines.len()
    );
    assert!(
        lines[0].starts_with("2#"),
        "first line should start with '2#', got: {}",
        lines[0]
    );
    assert!(
        lines[2].starts_with("4#"),
        "last line should start with '4#', got: {}",
        lines[2]
    );
}

// ---------------------------------------------------------------------------
// Test 3: --chunk-size 2 truncates output to 2 lines (with warning on stderr)
// ---------------------------------------------------------------------------
#[test]
fn read_chunk_size_truncates_output() {
    // 5 lines, request chunk-size 2 → only 2 lines emitted
    let f = make_temp_file(b"a\nb\nc\nd\ne\n");
    let output = sl()
        .args(["read", f.path().to_str().unwrap(), "--chunk-size", "2"])
        .assert()
        .success()
        .get_output()
        .stdout
        .clone();
    let text = String::from_utf8_lossy(&output);
    let count = text.lines().count();
    assert_eq!(count, 2, "expected 2 lines due to chunk-size, got {count}");
}

// ---------------------------------------------------------------------------
// Test 4: nonexistent file exits non-zero with an error message
// ---------------------------------------------------------------------------
#[test]
fn read_nonexistent_file_exits_nonzero() {
    sl().args(["read", "/tmp/definitely_does_not_exist_xyz_abc.rs"])
        .assert()
        .failure()
        .stderr(predicate::str::contains("not found").or(predicate::str::contains("No such")));
}

// ---------------------------------------------------------------------------
// Test 5: empty file exits non-zero with a meaningful error message
// (sl read reports "Start line 1 exceeds file length 0" for empty files)
// ---------------------------------------------------------------------------
#[test]
fn read_empty_file_exits_with_error() {
    let f = make_temp_file(b"");
    sl().args(["read", f.path().to_str().unwrap()])
        .assert()
        .failure()
        .stderr(predicate::str::contains("exceeds").or(predicate::str::contains("empty")));
}

// ---------------------------------------------------------------------------
// Test 6: file with UTF-8 BOM — BOM stripped, output is correct hashline
// ---------------------------------------------------------------------------
#[test]
fn read_bom_file_strips_bom_and_outputs_correct_content() {
    // UTF-8 BOM = 0xEF 0xBB 0xBF
    let mut content = vec![0xEF_u8, 0xBB, 0xBF];
    content.extend_from_slice(b"first line\nsecond line\n");
    let f = make_temp_file(&content);
    let output = sl()
        .args(["read", f.path().to_str().unwrap(), "--lines", "1:1"])
        .assert()
        .success()
        .get_output()
        .stdout
        .clone();
    let text = String::from_utf8_lossy(&output);
    // Should not contain the raw BOM bytes in content part
    assert!(
        text.contains("first line"),
        "expected 'first line' in output, got: {text}"
    );
    // BOM bytes must not appear in the output
    assert!(
        !text.contains('\u{FEFF}'),
        "BOM character should be stripped"
    );
}
