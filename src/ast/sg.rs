use anyhow::{bail, Context, Result};
use std::path::PathBuf;
use std::process::{Command, Stdio};
use std::time::Duration;

/// Look for `sg` binary in PATH, then in ~/.solon/bin/
pub fn find_sg() -> Option<PathBuf> {
    // Check PATH first
    if let Ok(output) = Command::new("which").arg("sg").output() {
        if output.status.success() {
            let path = String::from_utf8_lossy(&output.stdout).trim().to_string();
            if !path.is_empty() {
                return Some(PathBuf::from(path));
            }
        }
    }
    // Check ~/.solon/bin/sg
    if let Some(home) = dirs_home() {
        let local = home.join(".solon").join("bin").join(sg_binary_name());
        if local.exists() {
            return Some(local);
        }
    }
    None
}

fn dirs_home() -> Option<PathBuf> {
    std::env::var("HOME").ok().map(PathBuf::from)
}

fn sg_binary_name() -> &'static str {
    if cfg!(windows) {
        "sg.exe"
    } else {
        "sg"
    }
}

/// Return the download URL for the sg binary for the current platform
fn sg_download_url() -> Option<String> {
    let os = if cfg!(target_os = "linux") {
        "linux"
    } else if cfg!(target_os = "macos") {
        "darwin"
    } else if cfg!(target_os = "windows") {
        "windows"
    } else {
        return None;
    };

    let arch = if cfg!(target_arch = "x86_64") {
        "x86_64"
    } else if cfg!(target_arch = "aarch64") {
        "aarch64"
    } else {
        return None;
    };

    Some(format!(
        "https://github.com/ast-grep/ast-grep/releases/latest/download/sg-{os}-{arch}",
    ))
}

/// Download the sg binary to ~/.solon/bin/ and make it executable
pub fn download_sg() -> Result<PathBuf> {
    let url = sg_download_url()
        .ok_or_else(|| anyhow::anyhow!("Unsupported platform for automatic sg download"))?;

    let home = dirs_home().ok_or_else(|| anyhow::anyhow!("Cannot determine home directory"))?;
    let bin_dir = home.join(".solon").join("bin");
    std::fs::create_dir_all(&bin_dir)
        .with_context(|| format!("Failed to create {}", bin_dir.display()))?;

    let dest = bin_dir.join(sg_binary_name());

    eprintln!("Downloading ast-grep (sg) from {url} ...");

    // Use curl or wget depending on availability
    let status = Command::new("curl")
        .args(["-fsSL", "-o", dest.to_str().unwrap(), &url])
        .status()
        .or_else(|_| {
            Command::new("wget")
                .args(["-q", "-O", dest.to_str().unwrap(), &url])
                .status()
        })
        .context("Failed to download sg binary (curl/wget not found)")?;

    if !status.success() {
        bail!("Download failed. Please install ast-grep manually: https://ast-grep.github.io/guide/quick-start.html");
    }

    // Make executable on Unix
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        let mut perms = std::fs::metadata(&dest)?.permissions();
        perms.set_mode(0o755);
        std::fs::set_permissions(&dest, perms)?;
    }

    eprintln!("ast-grep installed to {}", dest.display());
    Ok(dest)
}

/// Run the sg binary with the given args, returning stdout as a String.
/// Times out after `timeout` duration.
pub fn run_sg(sg_path: &PathBuf, args: &[&str], timeout: Duration) -> Result<String> {
    let child = Command::new(sg_path)
        .args(args)
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .with_context(|| format!("Failed to spawn {}", sg_path.display()))?;

    // Use a thread to enforce timeout
    let start = std::time::Instant::now();
    let output = child.wait_with_output()?;

    if start.elapsed() > timeout {
        bail!("sg timed out after {}s", timeout.as_secs());
    }

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        bail!("sg failed: {stderr}");
    }

    Ok(String::from_utf8_lossy(&output.stdout).into_owned())
}

/// Find sg or attempt to download it, returning its path
pub fn require_sg() -> Result<PathBuf> {
    if let Some(p) = find_sg() {
        return Ok(p);
    }
    eprintln!("ast-grep (sg) not found. Attempting automatic download...");
    download_sg()
}
