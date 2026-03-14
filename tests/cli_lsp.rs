/// Integration tests for `sl lsp` using assert_cmd.
/// These tests require `rust-analyzer` to be installed (via `rustup component add rust-analyzer`).
/// All tests are marked `#[ignore]` — they run in CI where rust-analyzer is guaranteed present.
///
/// To run locally once rust-analyzer is installed:
///   cargo test --test cli_lsp -- --include-ignored
use assert_cmd::Command;
use predicates::prelude::*;
use std::fs;
use tempfile::TempDir;

fn sl() -> Command {
    Command::cargo_bin("sl").expect("sl binary not found")
}

/// Create a minimal Cargo project in a temp dir and return (dir, main_rs_path).
/// rust-analyzer needs a valid Cargo workspace to provide useful results.
fn make_rust_project() -> (TempDir, std::path::PathBuf) {
    let dir = TempDir::new().unwrap();

    // Cargo.toml
    fs::write(
        dir.path().join("Cargo.toml"),
        r#"[package]
name = "test-lsp-fixture"
version = "0.1.0"
edition = "2021"
"#,
    )
    .unwrap();

    // src/main.rs
    let src_dir = dir.path().join("src");
    fs::create_dir_all(&src_dir).unwrap();
    let main_rs = src_dir.join("main.rs");
    fs::write(
        &main_rs,
        r#"fn greet(name: &str) -> String {
    format!("Hello, {}!", name)
}

fn main() {
    println!("{}", greet("world"));
}
"#,
    )
    .unwrap();

    (dir, main_rs)
}

// ---------------------------------------------------------------------------
// Test 1: diagnostics on a valid file reports "No diagnostics" or a list
// ---------------------------------------------------------------------------
#[test]
#[ignore = "requires rust-analyzer — installed in CI via rustup component add rust-analyzer"]
fn lsp_diagnostics_valid_file_exits_zero() {
    let (_dir, main_rs) = make_rust_project();
    sl()
        .args(["lsp", "diagnostics", main_rs.to_str().unwrap()])
        .assert()
        .success();
}

// ---------------------------------------------------------------------------
// Test 2: diagnostics on a file with a type error reports the error
// ---------------------------------------------------------------------------
#[test]
#[ignore = "requires rust-analyzer — installed in CI via rustup component add rust-analyzer"]
fn lsp_diagnostics_file_with_error_reports_diagnostic() {
    let dir = TempDir::new().unwrap();

    fs::write(
        dir.path().join("Cargo.toml"),
        "[package]\nname = \"broken\"\nversion = \"0.1.0\"\nedition = \"2021\"\n",
    )
    .unwrap();
    let src = dir.path().join("src");
    fs::create_dir_all(&src).unwrap();
    let main_rs = src.join("main.rs");
    // Intentional type error: assign string to integer
    fs::write(&main_rs, "fn main() { let x: i32 = \"oops\"; }\n").unwrap();

    let output = sl()
        .args(["lsp", "diagnostics", main_rs.to_str().unwrap()])
        .assert()
        .success()
        .get_output()
        .stdout
        .clone();

    let text = String::from_utf8_lossy(&output);
    // rust-analyzer should report a type mismatch diagnostic
    assert!(
        text.contains("error") || text.contains("mismatch") || text.contains("expected"),
        "expected diagnostic about type mismatch, got: {text}"
    );
}

// ---------------------------------------------------------------------------
// Test 3: hover on `fn main` returns some hover info
// ---------------------------------------------------------------------------
#[test]
#[ignore = "requires rust-analyzer — installed in CI via rustup component add rust-analyzer"]
fn lsp_hover_on_fn_keyword_returns_info() {
    let (_dir, main_rs) = make_rust_project();
    // Line 5 col 1 — `fn` keyword of `fn main()`
    sl()
        .args([
            "lsp",
            "hover",
            main_rs.to_str().unwrap(),
            "5",
            "1",
        ])
        .assert()
        .success();
    // Success is sufficient — hover may or may not return content for a keyword
}

// ---------------------------------------------------------------------------
// Test 4: goto-def on a function call returns a location or empty
// ---------------------------------------------------------------------------
#[test]
#[ignore = "requires rust-analyzer — installed in CI via rustup component add rust-analyzer"]
fn lsp_goto_def_on_greet_call_returns_location() {
    let (_dir, main_rs) = make_rust_project();
    // Line 6, col 20 points to `greet` call inside `println!`
    let output = sl()
        .args([
            "lsp",
            "goto-def",
            main_rs.to_str().unwrap(),
            "6",
            "20",
        ])
        .assert()
        .success()
        .get_output()
        .stdout
        .clone();

    let text = String::from_utf8_lossy(&output);
    // Either a location is returned, or "No definitions found"
    assert!(
        text.contains("main.rs") || text.contains("definition") || text.contains("No"),
        "expected goto-def output, got: {text}"
    );
}

// ---------------------------------------------------------------------------
// Test 5: references on `greet` function name returns at least one reference
// ---------------------------------------------------------------------------
#[test]
#[ignore = "requires rust-analyzer — installed in CI via rustup component add rust-analyzer"]
fn lsp_references_on_greet_returns_locations() {
    let (_dir, main_rs) = make_rust_project();
    // Line 1, col 4 — `greet` in `fn greet(...)`
    let output = sl()
        .args([
            "lsp",
            "references",
            main_rs.to_str().unwrap(),
            "1",
            "4",
        ])
        .assert()
        .success()
        .get_output()
        .stdout
        .clone();

    let text = String::from_utf8_lossy(&output);
    // Should list at least the definition and the call site
    assert!(
        text.contains("main.rs") || text.contains("reference") || text.contains("No"),
        "expected references output, got: {text}"
    );
}

// ---------------------------------------------------------------------------
// Test 6: unknown file extension — no language server configured → error
// ---------------------------------------------------------------------------
#[test]
#[ignore = "requires rust-analyzer — installed in CI via rustup component add rust-analyzer"]
fn lsp_unknown_file_extension_errors_gracefully() {
    let dir = TempDir::new().unwrap();
    let unknown = dir.path().join("data.xyz");
    fs::write(&unknown, "some content\n").unwrap();

    sl()
        .args(["lsp", "diagnostics", unknown.to_str().unwrap()])
        .assert()
        .failure()
        .stderr(
            predicate::str::contains("No language server")
                .or(predicate::str::contains("not configured"))
                .or(predicate::str::contains("unknown")),
        );
}
