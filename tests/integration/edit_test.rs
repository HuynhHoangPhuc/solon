use std::fs;
use std::path::PathBuf;
use std::process::Command;
use tempfile::TempDir;

fn sl_bin() -> PathBuf {
    let mut path = std::env::current_exe().unwrap();
    path.pop();
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

/// Copy sample.rs to a temp dir and return (temp_dir, file_path)
fn setup_temp_file(filename: &str) -> (TempDir, PathBuf) {
    let tmp = TempDir::new().unwrap();
    let src = fixtures_dir().join(filename);
    let dest = tmp.path().join(filename);
    fs::copy(&src, &dest).unwrap();
    (tmp, dest)
}

/// Get the CID for a specific line by reading via sl
fn get_line_ref(file: &str, line: usize) -> String {
    let (_, stdout, _) = run_sl(&["read", file, "--lines", &format!("{line}:{line}")]);
    let first = stdout.lines().next().expect("no output from sl read");
    // Format: "LINE#HASH|CONTENT" → return "LINE#HASH"
    let pipe_pos = first.find('|').expect("no pipe in hashline");
    first[..pipe_pos].to_string()
}

#[test]
fn replace_single_line_changes_content() {
    let (_tmp, dest) = setup_temp_file("sample.rs");
    let path = dest.to_str().unwrap();

    let line_ref = get_line_ref(path, 1);
    let (code, _, _) = run_sl(&["edit", path, &line_ref, "// replaced first line", "--no-backup"]);
    assert_eq!(code, 0, "edit should succeed");

    let content = fs::read_to_string(&dest).unwrap();
    assert!(content.contains("// replaced first line"), "file should contain new content");
    assert!(!content.contains("use std::fmt;"), "old content should be gone");
}

#[test]
fn replace_roundtrip_verifiable_with_read() {
    let (_tmp, dest) = setup_temp_file("sample.rs");
    let path = dest.to_str().unwrap();

    let line_ref = get_line_ref(path, 2);
    let (code, _, _) = run_sl(&["edit", path, &line_ref, "// line 2 replaced", "--no-backup"]);
    assert_eq!(code, 0);

    // Re-read line 2 — should show new content with updated hash
    let (code2, stdout, _) = run_sl(&["read", path, "--lines", "2:2"]);
    assert_eq!(code2, 0);
    let line = stdout.lines().next().unwrap();
    assert!(line.contains("// line 2 replaced"), "read should show new content: {line}");
}

#[test]
fn hash_mismatch_rejected() {
    let (_tmp, dest) = setup_temp_file("sample.rs");
    let path = dest.to_str().unwrap();

    // Use a wrong hash for line 1
    let (code, _, stderr) = run_sl(&["edit", path, "1#ZZ", "new content", "--no-backup"]);
    assert_ne!(code, 0, "edit with wrong hash should fail");
    assert!(stderr.contains("Hash mismatch") || stderr.contains("mismatch"),
        "expected hash mismatch error, got: {stderr}");
}

#[test]
fn append_after_line_inserts_content() {
    let (_tmp, dest) = setup_temp_file("sample.rs");
    let path = dest.to_str().unwrap();

    let line_ref = get_line_ref(path, 1);
    let (code, _, _) = run_sl(&[
        "edit", path, "--after", &line_ref, "// inserted after line 1", "--no-backup"
    ]);
    assert_eq!(code, 0);

    let content = fs::read_to_string(&dest).unwrap();
    let lines: Vec<&str> = content.lines().collect();
    assert_eq!(lines[0], "use std::fmt;", "line 1 unchanged");
    assert_eq!(lines[1], "// inserted after line 1", "line 2 should be new");
}

#[test]
fn delete_removes_lines() {
    let (_tmp, dest) = setup_temp_file("sample.rs");
    let path = dest.to_str().unwrap();

    let line_ref = get_line_ref(path, 3);
    let original_line_count = fs::read_to_string(&dest).unwrap().lines().count();

    let (code, _, _) = run_sl(&["edit", path, "--delete", &line_ref, "--no-backup"]);
    assert_eq!(code, 0);

    let new_line_count = fs::read_to_string(&dest).unwrap().lines().count();
    assert_eq!(new_line_count, original_line_count - 1, "one line should be deleted");
}

#[test]
fn edit_diff_output_shows_changes() {
    let (_tmp, dest) = setup_temp_file("sample.rs");
    let path = dest.to_str().unwrap();

    let line_ref = get_line_ref(path, 1);
    let (code, stdout, _) = run_sl(&["edit", path, &line_ref, "// changed", "--no-backup"]);
    assert_eq!(code, 0);
    // Diff output should contain + and - markers
    assert!(stdout.contains('+') || stdout.contains('-') || stdout.contains("@@"),
        "expected diff output, got: {stdout}");
}
