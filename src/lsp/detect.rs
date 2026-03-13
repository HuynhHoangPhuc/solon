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
        "java" => "Install Eclipse JDT Language Server: https://github.com/eclipse-jdtls/eclipse.jdt.ls".to_string(),
        lang => format!("Install language server for {lang}"),
    }
}
