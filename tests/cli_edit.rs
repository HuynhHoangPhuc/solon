/// Integration tests for `sl edit` using assert_cmd.
use assert_cmd::Command;
use predicates::prelude::*;
use std::fs;
use std::io::Write;
use tempfile::NamedTempFile;

fn sl() -> Command {
    Command::cargo_bin("sl").expect("sl binary not found")
}

fn make_temp_file(content: &[u8]) -> NamedTempFile {
    let mut f = NamedTempFile::new().unwrap();
    f.write_all(content).unwrap();
    f.flush().unwrap();
    f
}

/// Read the hashline reference for a specific line (e.g. "3#AB").
fn get_line_ref(path: &str, line: usize) -> String {
    let output = sl()
        .args(["read", path, "--lines", &format!("{line}:{line}")])
        .output()
        .unwrap();
    let stdout = String::from_utf8_lossy(&output.stdout).into_owned();
    let first = stdout.lines().next().expect("sl read returned no output");
    // "LINE#HASH|CONTENT" → "LINE#HASH"
    let pipe = first.find('|').expect("no pipe in hashline");
    first[..pipe].to_string()
}

// ---------------------------------------------------------------------------
// Test 1: replace single line — file content updated
// ---------------------------------------------------------------------------
#[test]
fn edit_replace_single_line_changes_content() {
    let f = make_temp_file(b"alpha\nbeta\ngamma\n");
    let path = f.path().to_str().unwrap();

    let line_ref = get_line_ref(path, 1);
    sl().args(["edit", path, &line_ref, "// replaced", "--no-backup"])
        .assert()
        .success();

    let content = fs::read_to_string(f.path()).unwrap();
    assert!(
        content.contains("// replaced"),
        "expected new content in file"
    );
    assert!(!content.contains("alpha"), "old content should be gone");
}

// ---------------------------------------------------------------------------
// Test 2: replace a range of lines
// ---------------------------------------------------------------------------
#[test]
fn edit_replace_range_updates_correct_lines() {
    let f = make_temp_file(b"line1\nline2\nline3\nline4\n");
    let path = f.path().to_str().unwrap();

    let start_ref = get_line_ref(path, 2);
    let end_ref = get_line_ref(path, 3);
    sl().args([
        "edit",
        path,
        &start_ref,
        &end_ref,
        "// merged",
        "--no-backup",
    ])
    .assert()
    .success();

    let content = fs::read_to_string(f.path()).unwrap();
    // lines 2 and 3 replaced by one line "// merged"
    assert!(content.contains("// merged"), "merged content missing");
    assert!(!content.contains("line2"), "line2 should be replaced");
    assert!(!content.contains("line3"), "line3 should be replaced");
    assert!(content.contains("line1"), "line1 should be untouched");
    assert!(content.contains("line4"), "line4 should be untouched");
}

// ---------------------------------------------------------------------------
// Test 3: --after inserts content after the target line
// ---------------------------------------------------------------------------
#[test]
fn edit_after_inserts_content_after_target() {
    let f = make_temp_file(b"first\nsecond\nthird\n");
    let path = f.path().to_str().unwrap();

    let line_ref = get_line_ref(path, 1);
    sl().args([
        "edit",
        path,
        "--after",
        &line_ref,
        "// inserted",
        "--no-backup",
    ])
    .assert()
    .success();

    let content = fs::read_to_string(f.path()).unwrap();
    let lines: Vec<&str> = content.lines().collect();
    assert_eq!(lines[0], "first", "line 1 should be unchanged");
    assert_eq!(
        lines[1], "// inserted",
        "line 2 should be new inserted line"
    );
    assert_eq!(lines[2], "second", "line 3 should be former line 2");
}

// ---------------------------------------------------------------------------
// Test 4: --before inserts content before the target line
// ---------------------------------------------------------------------------
#[test]
fn edit_before_inserts_content_before_target() {
    let f = make_temp_file(b"first\nsecond\nthird\n");
    let path = f.path().to_str().unwrap();

    let line_ref = get_line_ref(path, 2);
    sl().args([
        "edit",
        path,
        "--before",
        &line_ref,
        "// before second",
        "--no-backup",
    ])
    .assert()
    .success();

    let content = fs::read_to_string(f.path()).unwrap();
    let lines: Vec<&str> = content.lines().collect();
    assert_eq!(lines[0], "first");
    assert_eq!(
        lines[1], "// before second",
        "inserted line should appear before 'second'"
    );
    assert_eq!(lines[2], "second");
}

// ---------------------------------------------------------------------------
// Test 5: --delete removes the target line
// ---------------------------------------------------------------------------
#[test]
fn edit_delete_removes_target_line() {
    let f = make_temp_file(b"keep\nremove_me\nkeep_too\n");
    let path = f.path().to_str().unwrap();

    let original_count = fs::read_to_string(f.path()).unwrap().lines().count();
    let line_ref = get_line_ref(path, 2);

    sl().args(["edit", path, "--delete", &line_ref, "--no-backup"])
        .assert()
        .success();

    let content = fs::read_to_string(f.path()).unwrap();
    let new_count = content.lines().count();
    assert_eq!(
        new_count,
        original_count - 1,
        "line count should decrease by 1"
    );
    assert!(
        !content.contains("remove_me"),
        "deleted line should be gone"
    );
}

// ---------------------------------------------------------------------------
// Test 6: wrong hash rejected with "hash mismatch" error
// ---------------------------------------------------------------------------
#[test]
fn edit_wrong_hash_rejected_with_mismatch_error() {
    let f = make_temp_file(b"one\ntwo\nthree\n");
    let path = f.path().to_str().unwrap();

    sl().args(["edit", path, "1#ZZ", "new content", "--no-backup"])
        .assert()
        .failure()
        .stderr(
            predicate::str::contains("mismatch")
                .or(predicate::str::contains("Mismatch"))
                .or(predicate::str::contains("hash")),
        );
}

// ---------------------------------------------------------------------------
// Test 7: --no-backup skips creation of .bak file
// ---------------------------------------------------------------------------
#[test]
fn edit_no_backup_creates_no_bak_file() {
    let f = make_temp_file(b"only\none\nline\n");
    let path = f.path().to_str().unwrap();
    let bak_path = format!("{}.bak", path);

    let line_ref = get_line_ref(path, 1);
    sl().args(["edit", path, &line_ref, "changed", "--no-backup"])
        .assert()
        .success();

    assert!(
        !std::path::Path::new(&bak_path).exists(),
        ".bak file should not be created with --no-backup"
    );
}

// ---------------------------------------------------------------------------
// Test 8: --stdin reads replacement content from stdin pipe
// ---------------------------------------------------------------------------
#[test]
fn edit_stdin_reads_content_from_stdin() {
    let f = make_temp_file(b"original line\nsecond line\n");
    let path = f.path().to_str().unwrap();

    let line_ref = get_line_ref(path, 1);
    sl().args(["edit", path, &line_ref, "--stdin", "--no-backup"])
        .write_stdin("from stdin\n")
        .assert()
        .success();

    let content = fs::read_to_string(f.path()).unwrap();
    assert!(
        content.contains("from stdin"),
        "stdin content should be in file"
    );
    assert!(
        !content.contains("original line"),
        "original content should be replaced"
    );
}

// ---------------------------------------------------------------------------
// Test 9: diff output contains +/- markers or @@ hunks
// ---------------------------------------------------------------------------
#[test]
fn edit_diff_output_contains_diff_markers() {
    let f = make_temp_file(b"first line\nsecond line\nthird line\n");
    let path = f.path().to_str().unwrap();

    let line_ref = get_line_ref(path, 2);
    sl().args(["edit", path, &line_ref, "// changed", "--no-backup"])
        .assert()
        .success()
        .stdout(
            predicate::str::contains('+')
                .or(predicate::str::contains('-'))
                .or(predicate::str::contains("@@")),
        );
}
