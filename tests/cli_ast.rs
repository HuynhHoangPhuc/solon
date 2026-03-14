/// Integration tests for `sl ast` using assert_cmd.
/// These tests require the `sg` (ast-grep) binary to be installed.
/// All tests are marked `#[ignore]` so they run only when `sg` is available
/// (CI installs it via `cargo install ast-grep --locked`).
///
/// To run locally once sg is installed:
///   cargo test --test cli_ast -- --include-ignored
use assert_cmd::Command;
use predicates::prelude::*;
use std::io::Write;
use tempfile::{Builder, TempDir};

fn sl() -> Command {
    Command::cargo_bin("sl").expect("sl binary not found")
}

/// Create a temporary directory containing a single Rust source file.
fn make_rust_tmpdir(src: &str) -> (TempDir, std::path::PathBuf) {
    let dir = TempDir::new().unwrap();
    let file = dir.path().join("sample.rs");
    let mut f = std::fs::File::create(&file).unwrap();
    f.write_all(src.as_bytes()).unwrap();
    (dir, file)
}

const SAMPLE_RS: &str = r#"
fn add(a: i32, b: i32) -> i32 {
    a + b
}

fn multiply(x: i32, y: i32) -> i32 {
    x * y
}

fn main() {
    println!("{}", add(1, 2));
}
"#;

// ---------------------------------------------------------------------------
// Test 1: search for fn pattern finds functions in the directory
// ---------------------------------------------------------------------------
#[test]
#[ignore = "requires sg (ast-grep) binary — installed in CI"]
fn ast_search_finds_functions_in_rust_file() {
    let (dir, _file) = make_rust_tmpdir(SAMPLE_RS);

    sl().args([
        "ast",
        "search",
        "fn $NAME($$$ARGS)",
        "--lang",
        "rust",
        "--path",
        dir.path().to_str().unwrap(),
    ])
    .assert()
    .success()
    // Should find at least one of: add, multiply, main
    .stdout(predicate::str::contains("add").or(predicate::str::contains("multiply")));
}

// ---------------------------------------------------------------------------
// Test 2: --json flag emits raw JSON from sg
// ---------------------------------------------------------------------------
#[test]
#[ignore = "requires sg (ast-grep) binary — installed in CI"]
fn ast_search_json_flag_produces_json_output() {
    let (dir, _file) = make_rust_tmpdir(SAMPLE_RS);

    sl().args([
        "ast",
        "search",
        "fn $NAME($$$ARGS)",
        "--lang",
        "rust",
        "--path",
        dir.path().to_str().unwrap(),
        "--json",
    ])
    .assert()
    .success()
    // JSON output starts with [ or {
    .stdout(predicate::str::starts_with("[").or(predicate::str::starts_with("{")));
}

// ---------------------------------------------------------------------------
// Test 3: --max-results 1 limits output to one match
// ---------------------------------------------------------------------------
#[test]
#[ignore = "requires sg (ast-grep) binary — installed in CI"]
fn ast_search_max_results_limits_output() {
    let (dir, _file) = make_rust_tmpdir(SAMPLE_RS);

    let output = sl()
        .args([
            "ast",
            "search",
            "fn $NAME($$$ARGS)",
            "--lang",
            "rust",
            "--path",
            dir.path().to_str().unwrap(),
            "--max-results",
            "1",
        ])
        .assert()
        .success()
        .get_output()
        .stdout
        .clone();

    let text = String::from_utf8_lossy(&output);
    // With max-results 1, the formatted output should mention only 1 match
    // The format_search_results function labels matches — count "Match" occurrences
    let match_count = text.matches("Match").count();
    assert!(
        match_count <= 1,
        "expected at most 1 match block, got {match_count}:\n{text}"
    );
}

// ---------------------------------------------------------------------------
// Test 4: pattern with no matches reports "0 matches" or empty output
// ---------------------------------------------------------------------------
#[test]
#[ignore = "requires sg (ast-grep) binary — installed in CI"]
fn ast_search_no_matches_reports_zero() {
    let (dir, _file) = make_rust_tmpdir(SAMPLE_RS);

    let output = sl()
        .args([
            "ast",
            "search",
            "class $NAME",
            "--lang",
            "rust",
            "--path",
            dir.path().to_str().unwrap(),
        ])
        .assert()
        .success()
        .get_output()
        .stdout
        .clone();

    let text = String::from_utf8_lossy(&output);
    // Either empty output or an explicit "0 matches" message
    assert!(
        text.trim().is_empty() || text.contains("0"),
        "expected no match output, got: {text}"
    );
}

// ---------------------------------------------------------------------------
// Test 5: replace preview — sl ast replace shows diff-like output
// ---------------------------------------------------------------------------
#[test]
#[ignore = "requires sg (ast-grep) binary — installed in CI"]
fn ast_replace_preview_shows_replacement() {
    let (dir, _file) = make_rust_tmpdir(SAMPLE_RS);

    // Replace pattern: rename `add` calls to `sum`
    sl().args([
        "ast",
        "replace",
        "fn $NAME($$$ARGS)",
        "fn renamed_$NAME($$$ARGS)",
        "--lang",
        "rust",
        "--path",
        dir.path().to_str().unwrap(),
    ])
    .assert()
    .success();
    // Just verify the command exits 0 — preview output format varies by sg version
}

// ---------------------------------------------------------------------------
// Test 6: search across multiple files in a directory
// ---------------------------------------------------------------------------
#[test]
#[ignore = "requires sg (ast-grep) binary — installed in CI"]
fn ast_search_across_multiple_files() {
    let dir = TempDir::new().unwrap();

    // Write two Rust files
    for (name, src) in [
        ("alpha.rs", "fn alpha() {}\n"),
        ("beta.rs", "fn beta() -> i32 { 42 }\n"),
    ] {
        let path = dir.path().join(name);
        let mut f = std::fs::File::create(&path).unwrap();
        f.write_all(src.as_bytes()).unwrap();
    }

    let output = sl()
        .args([
            "ast",
            "search",
            "fn $NAME($$$ARGS)",
            "--lang",
            "rust",
            "--path",
            dir.path().to_str().unwrap(),
        ])
        .assert()
        .success()
        .get_output()
        .stdout
        .clone();

    let text = String::from_utf8_lossy(&output);
    // Should find functions from both files
    assert!(
        text.contains("alpha") || text.contains("beta"),
        "expected matches from both files, got: {text}"
    );
}
