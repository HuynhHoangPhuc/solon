use anyhow::{bail, Context, Result};
use serde_json::Value as JsonValue;
use serde_json::{json, Value};
use std::io::{BufRead, BufReader, Write};
use std::path::Path;
use std::process::{Child, ChildStdin, ChildStdout, Command, Stdio};
use std::sync::atomic::{AtomicI64, Ordering};

use super::detect::ServerConfig;

static REQUEST_ID: AtomicI64 = AtomicI64::new(1);

/// A simple stdio LSP client (v1: fresh connection per invocation, no caching)
pub struct LspClient {
    process: Child,
    stdin: ChildStdin,
    reader: BufReader<ChildStdout>,
}

impl LspClient {
    /// Spawn the language server and perform LSP initialization handshake
    pub fn connect(config: &ServerConfig, root: &Path) -> Result<Self> {
        let root_uri = path_to_uri(root);

        let mut process = Command::new(&config.command)
            .args(&config.args)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .with_context(|| {
                format!(
                    "Failed to start language server '{}'. {}",
                    config.command,
                    super::detect::install_hint(config)
                )
            })?;

        let stdin = process.stdin.take().unwrap();
        let stdout = process.stdout.take().unwrap();
        let reader = BufReader::new(stdout);

        let mut client = LspClient {
            process,
            stdin,
            reader,
        };

        // Send initialize request
        let init_params = json!({
            "processId": std::process::id(),
            "rootUri": root_uri,
            "capabilities": {
                "textDocument": {
                    "hover": { "contentFormat": ["plaintext"] },
                    "publishDiagnostics": {}
                }
            },
            "trace": "off"
        });

        let id = client.next_id();
        client.send_request("initialize", id, init_params)?;
        client.read_response(id)?;

        // Send initialized notification
        client.send_notification("initialized", json!({}))?;

        Ok(client)
    }

    fn next_id(&self) -> i64 {
        REQUEST_ID.fetch_add(1, Ordering::SeqCst)
    }

    /// Send an LSP request (method + params) and return the parsed response
    pub fn request(&mut self, method: &str, params: Value) -> Result<Value> {
        let id = self.next_id();
        self.send_request(method, id, params)?;
        self.read_response(id)
    }

    fn send_request(&mut self, method: &str, id: i64, params: Value) -> Result<()> {
        let msg = json!({
            "jsonrpc": "2.0",
            "id": id,
            "method": method,
            "params": params
        });
        self.write_message(&msg)
    }

    fn send_notification(&mut self, method: &str, params: Value) -> Result<()> {
        let msg = json!({
            "jsonrpc": "2.0",
            "method": method,
            "params": params
        });
        self.write_message(&msg)
    }

    fn write_message(&mut self, msg: &Value) -> Result<()> {
        let body = serde_json::to_string(msg)?;
        let header = format!("Content-Length: {}\r\n\r\n", body.len());
        self.stdin.write_all(header.as_bytes())?;
        self.stdin.write_all(body.as_bytes())?;
        self.stdin.flush()?;
        Ok(())
    }

    /// Read messages from the server until we find the response with matching id.
    /// Notifications and other messages are discarded.
    fn read_response(&mut self, id: i64) -> Result<Value> {
        loop {
            let msg = self.read_message()?;

            // Check if this is the response we're waiting for
            if let Some(resp_id) = msg.get("id") {
                if resp_id.as_i64() == Some(id) {
                    if let Some(error) = msg.get("error") {
                        bail!("LSP error: {error}");
                    }
                    return Ok(msg.get("result").cloned().unwrap_or(Value::Null));
                }
            }
            // Skip notifications and other responses
        }
    }

    fn read_message(&mut self) -> Result<Value> {
        let mut content_length = 0usize;

        // Read headers
        loop {
            let mut line = String::new();
            self.reader.read_line(&mut line)?;
            let line = line.trim_end_matches(['\r', '\n']).to_string();
            if line.is_empty() {
                break; // End of headers
            }
            if let Some(rest) = line.strip_prefix("Content-Length: ") {
                content_length = rest.parse().context("Invalid Content-Length")?;
            }
        }

        if content_length == 0 {
            bail!("No Content-Length header received from language server");
        }

        let mut body = vec![0u8; content_length];
        use std::io::Read;
        self.reader.read_exact(&mut body)?;

        serde_json::from_slice(&body).context("Invalid JSON from language server")
    }

    /// Open a text document in the server
    pub fn open_document(&mut self, file_path: &Path) -> Result<()> {
        let uri = path_to_uri(file_path);
        let content = std::fs::read_to_string(file_path)
            .with_context(|| format!("Cannot read {}", file_path.display()))?;
        let ext = file_path.extension().and_then(|e| e.to_str()).unwrap_or("");
        let language_id = ext_to_language_id(ext);

        self.send_notification(
            "textDocument/didOpen",
            json!({
                "textDocument": {
                    "uri": uri,
                    "languageId": language_id,
                    "version": 1,
                    "text": content
                }
            }),
        )
    }

    /// Get diagnostics for an open document (wait for publishDiagnostics notification).
    /// Returns raw JSON diagnostic objects.
    pub fn get_diagnostics(&mut self, file_path: &Path) -> Result<Vec<JsonValue>> {
        let uri = path_to_uri(file_path);

        // Wait for publishDiagnostics notification (with timeout simulation via message limit)
        let max_msgs = 30;
        for _ in 0..max_msgs {
            let msg = self.read_message()?;
            if msg.get("method").and_then(Value::as_str) == Some("textDocument/publishDiagnostics")
            {
                if let Some(params) = msg.get("params") {
                    if params.get("uri").and_then(Value::as_str) == Some(&uri) {
                        let diags = params
                            .get("diagnostics")
                            .and_then(Value::as_array)
                            .cloned()
                            .unwrap_or_default();
                        return Ok(diags);
                    }
                }
            }
        }
        Ok(vec![]) // No diagnostics received in time
    }

    /// Graceful shutdown
    pub fn shutdown(&mut self) {
        let id = self.next_id();
        let _ = self.send_request("shutdown", id, json!(null));
        let _ = self.read_response(id);
        let _ = self.send_notification("exit", json!(null));
        let _ = self.process.wait();
    }
}

impl Drop for LspClient {
    fn drop(&mut self) {
        self.shutdown();
    }
}

/// Convert a filesystem path to a `file://` URI string
fn path_to_uri(path: &Path) -> String {
    let abs = if path.is_absolute() {
        path.to_string_lossy().into_owned()
    } else {
        std::env::current_dir()
            .unwrap_or_default()
            .join(path)
            .to_string_lossy()
            .into_owned()
    };
    format!("file://{abs}")
}

fn ext_to_language_id(ext: &str) -> &'static str {
    match ext {
        "rs" => "rust",
        "ts" | "tsx" => "typescript",
        "js" | "jsx" | "mjs" | "cjs" => "javascript",
        "py" => "python",
        "go" => "go",
        "java" => "java",
        _ => "plaintext",
    }
}
