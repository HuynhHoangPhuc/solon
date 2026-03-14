use std::path::{Path, PathBuf};

/// Configuration for a language server
#[derive(Debug, Clone)]
pub struct ServerConfig {
    pub language: &'static str,
    pub command: String,
    pub args: Vec<String>,
    /// Files that indicate the project root (searched upward from file)
    pub root_markers: Vec<&'static str>,
}

/// Detect the appropriate language server from a file's extension
pub fn detect_server(file_path: &Path) -> Option<ServerConfig> {
    let ext = file_path.extension()?.to_str()?;
    match ext {
        "rs" => Some(ServerConfig {
            language: "rust",
            command: "rust-analyzer".to_string(),
            args: vec![],
            root_markers: vec!["Cargo.toml"],
        }),
        "ts" | "tsx" | "js" | "jsx" | "mjs" | "cjs" => Some(ServerConfig {
            language: "typescript",
            command: "typescript-language-server".to_string(),
            args: vec!["--stdio".to_string()],
            root_markers: vec!["package.json", "tsconfig.json"],
        }),
        "py" => Some(ServerConfig {
            language: "python",
            command: "pyright-langserver".to_string(),
            args: vec!["--stdio".to_string()],
            root_markers: vec!["pyproject.toml", "setup.py", "requirements.txt"],
        }),
        "go" => Some(ServerConfig {
            language: "go",
            command: "gopls".to_string(),
            args: vec!["serve".to_string()],
            root_markers: vec!["go.mod"],
        }),
        "java" => Some(ServerConfig {
            language: "java",
            command: "jdtls".to_string(),
            args: vec![],
            root_markers: vec!["pom.xml", "build.gradle"],
        }),
        _ => None,
    }
}

/// Walk up from `start` looking for any of the marker files; returns the directory containing one
pub fn find_project_root(start: &Path, markers: &[&str]) -> PathBuf {
    let mut dir = if start.is_file() {
        start.parent().unwrap_or(start).to_path_buf()
    } else {
        start.to_path_buf()
    };

    loop {
        for marker in markers {
            if dir.join(marker).exists() {
                return dir;
            }
        }
        match dir.parent() {
            Some(parent) => dir = parent.to_path_buf(),
            None => return start.parent().unwrap_or(start).to_path_buf(),
        }
    }
}

/// Return a human-readable install hint for missing servers
pub fn install_hint(config: &ServerConfig) -> String {
    match config.language {
        "rust" => "Install rust-analyzer: https://rust-analyzer.github.io/".to_string(),
        "typescript" => "Run: npm install -g typescript-language-server typescript".to_string(),
        "python" => "Run: pip install pyright".to_string(),
        "go" => "Run: go install golang.org/x/tools/gopls@latest".to_string(),
        "java" => {
            "Install Eclipse JDT Language Server: https://github.com/eclipse-jdtls/eclipse.jdt.ls"
                .to_string()
        }
        lang => format!("Install language server for {lang}"),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    // --- detect_server ---

    #[test]
    fn detect_server_rust() {
        let cfg = detect_server(Path::new("main.rs")).unwrap();
        assert_eq!(cfg.language, "rust");
        assert_eq!(cfg.command, "rust-analyzer");
        assert!(cfg.root_markers.contains(&"Cargo.toml"));
    }

    #[test]
    fn detect_server_typescript_extensions() {
        for ext in &["ts", "tsx", "js", "jsx", "mjs", "cjs"] {
            let path = format!("file.{ext}");
            let cfg = detect_server(Path::new(&path)).unwrap();
            assert_eq!(cfg.language, "typescript", "failed for .{ext}");
            assert_eq!(cfg.command, "typescript-language-server");
        }
    }

    #[test]
    fn detect_server_python() {
        let cfg = detect_server(Path::new("app.py")).unwrap();
        assert_eq!(cfg.language, "python");
        assert_eq!(cfg.command, "pyright-langserver");
    }

    #[test]
    fn detect_server_go() {
        let cfg = detect_server(Path::new("main.go")).unwrap();
        assert_eq!(cfg.language, "go");
        assert_eq!(cfg.command, "gopls");
    }

    #[test]
    fn detect_server_java() {
        let cfg = detect_server(Path::new("Main.java")).unwrap();
        assert_eq!(cfg.language, "java");
        assert_eq!(cfg.command, "jdtls");
    }

    #[test]
    fn detect_server_unknown_extension_returns_none() {
        assert!(detect_server(Path::new("readme.txt")).is_none());
    }

    #[test]
    fn detect_server_no_extension_returns_none() {
        assert!(detect_server(Path::new("Makefile")).is_none());
    }

    // --- find_project_root ---

    #[test]
    fn find_project_root_finds_marker_in_same_dir() {
        let tmp = std::env::temp_dir().join("solon_test_root_same");
        fs::create_dir_all(&tmp).unwrap();
        fs::write(tmp.join("Cargo.toml"), "").unwrap();

        let result = find_project_root(&tmp, &["Cargo.toml"]);
        assert_eq!(result, tmp);

        // cleanup
        let _ = fs::remove_dir_all(&tmp);
    }

    #[test]
    fn find_project_root_walks_up_from_nested_file() {
        let tmp = std::env::temp_dir().join("solon_test_root_nested");
        let nested = tmp.join("src").join("lib");
        fs::create_dir_all(&nested).unwrap();
        fs::write(tmp.join("Cargo.toml"), "").unwrap();
        let file = nested.join("lib.rs");
        fs::write(&file, "").unwrap();

        let result = find_project_root(&file, &["Cargo.toml"]);
        assert_eq!(result, tmp);

        let _ = fs::remove_dir_all(&tmp);
    }

    #[test]
    fn find_project_root_no_marker_returns_parent_of_start() {
        // Use a directory that definitely has no marker with an unusual name
        let tmp = std::env::temp_dir().join("solon_test_no_marker_xyz987");
        fs::create_dir_all(&tmp).unwrap();
        let file = tmp.join("some_file.rs");
        fs::write(&file, "").unwrap();

        let result = find_project_root(&file, &["__nonexistent_marker__.toml"]);
        // When no marker found, returns parent of start (start is a file → its parent)
        assert_eq!(result, tmp);

        let _ = fs::remove_dir_all(&tmp);
    }

    // --- install_hint ---

    #[test]
    fn install_hint_rust() {
        let cfg = detect_server(Path::new("main.rs")).unwrap();
        assert!(install_hint(&cfg).contains("rust-analyzer"));
    }

    #[test]
    fn install_hint_typescript() {
        let cfg = detect_server(Path::new("app.ts")).unwrap();
        assert!(install_hint(&cfg).contains("typescript-language-server"));
    }

    #[test]
    fn install_hint_python() {
        let cfg = detect_server(Path::new("app.py")).unwrap();
        assert!(install_hint(&cfg).contains("pyright"));
    }

    #[test]
    fn install_hint_go() {
        let cfg = detect_server(Path::new("main.go")).unwrap();
        assert!(install_hint(&cfg).contains("gopls"));
    }

    #[test]
    fn install_hint_java() {
        let cfg = detect_server(Path::new("Main.java")).unwrap();
        assert!(
            install_hint(&cfg).contains("eclipse")
                || install_hint(&cfg).to_lowercase().contains("jdt")
        );
    }
}
