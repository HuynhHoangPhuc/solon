use std::path::PathBuf;
use std::process::Command;

fn sl_bin() -> PathBuf {
    // Use the debug binary built by cargo test
    let mut path = std::env::current_exe().unwrap();
    path.pop(); // strip test binary name
    if path.ends_with("deps") { path.pop(); }
    path.push(if cfg!(windows) { "sl.exe" } else { "sl" });
    path
}

fn fixtures_dir() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("tests/fixtures")
}

fn run_sl(args: &[&str]) -> (i32, String, String) {
    let output = Command::new(sl_bin())
        .args(args)
        .output()
        .expect("failed to run sl");
    let code = output.status.code().unwrap_or(-1);
    let stdout = String::from_utf8_lossy(&output.stdout).into_owned();
    let stderr = String::from_utf8_lossy(&output.stderr).into_owned();
    (code, stdout, stderr)
}

#[test]
fn read_sample_rs_produces_hashline_format() {
    let sample = fixtures_dir().join("sample.rs");
    let (code, stdout, _) = run_sl(&["read", sample.to_str().unwrap()]);
    assert_eq!(code, 0, "sl read should exit 0");
    // Every line should match LINE#HASH|CONTENT
    for line in stdout.lines() {
        assert!(
            line.contains('#') && line.contains('|'),
            "expected hashline format, got: {line}"
        );
    }
}

#[test]
fn read_line_range_returns_correct_lines() {
    let sample = fixtures_dir().join("sample.rs");
    let (code, stdout, _) = run_sl(&["read", sample.to_str().unwrap(), "--lines", "1:3"]);
    assert_eq!(code, 0);
    let lines: Vec<&str> = stdout.lines().collect();
    assert_eq!(lines.len(), 3, "expected 3 lines, got {}", lines.len());
    // First line should be line 1
    assert!(lines[0].starts_with("1#"), "expected line 1, got: {}", lines[0]);
    assert!(lines[2].starts_with("3#"), "expected line 3, got: {}", lines[2]);
}

#[test]
fn read_open_ended_range_reads_to_eof() {
    let sample = fixtures_dir().join("sample.rs");
    // Read from line 30 to EOF — sample.rs has ~40 lines
    let (code, stdout, _) = run_sl(&["read", sample.to_str().unwrap(), "--lines", "30:"]);
    assert_eq!(code, 0);
    // Should have at least 1 line
    assert!(!stdout.trim().is_empty(), "expected some output for open range");
    // First line number should be 30
    let first = stdout.lines().next().unwrap();
    assert!(first.starts_with("30#"), "expected first line to be 30#, got: {first}");
}

#[test]
fn read_nonexistent_file_exits_nonzero() {
    let (code, _, stderr) = run_sl(&["read", "nonexistent_file_xyz.rs"]);
    assert_ne!(code, 0, "expected nonzero exit for missing file");
    assert!(stderr.contains("not found") || stderr.contains("No such"),
        "expected error message, got: {stderr}");
}

#[test]
fn read_hash_is_deterministic() {
    let sample = fixtures_dir().join("sample.rs");
    let (_, out1, _) = run_sl(&["read", sample.to_str().unwrap(), "--lines", "1:5"]);
    let (_, out2, _) = run_sl(&["read", sample.to_str().unwrap(), "--lines", "1:5"]);
    assert_eq!(out1, out2, "hash output should be deterministic");
}
