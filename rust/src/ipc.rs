use crate::config::Config;
use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::fs;
use std::os::unix::fs::MetadataExt;
use std::os::unix::fs::PermissionsExt;
use std::os::unix::process::CommandExt;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::time::SystemTime;
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::UnixListener;

#[derive(Serialize, Deserialize, Debug)]
pub struct StatusResponse {
    pub pid: u32,
    pub start_time: SystemTime,
    pub version: String,
    pub config: Option<Config>,
}

pub fn ensure_secure_directory(path: &Path) -> Result<()> {
    if !path.exists() {
        fs::create_dir_all(path)
            .with_context(|| format!("Failed to create directory {:?}", path))?;
    }

    let metadata = fs::symlink_metadata(path)
        .with_context(|| format!("Failed to get metadata for {:?}", path))?;

    // Check if it is a directory
    if !metadata.is_dir() {
        anyhow::bail!("{:?} is not a directory", path);
    }

    // Check if it is a symlink
    if metadata.file_type().is_symlink() {
        anyhow::bail!("{:?} is a symlink", path);
    }

    // Check permissions (0700)
    let mode = metadata.permissions().mode() & 0o777;
    if mode != 0o700 {
        fs::set_permissions(path, fs::Permissions::from_mode(0o700))
            .with_context(|| format!("Failed to set permissions 0700 on {:?}", path))?;
    }

    // Check ownership
    let uid = unsafe { libc::getuid() };
    if metadata.uid() != uid {
        anyhow::bail!(
            "{:?} has incorrect ownership. Expected uid {}, got {}",
            path,
            uid,
            metadata.uid()
        );
    }

    Ok(())
}

pub async fn start_server(
    socket_path: PathBuf,
    config: Config,
    start_time: SystemTime,
) -> Result<()> {
    if socket_path.exists() {
        fs::remove_file(&socket_path).ok();
    }

    let listener = UnixListener::bind(&socket_path)
        .with_context(|| format!("Failed to bind to socket {:?}", socket_path))?;

    // Set socket permissions to 0600
    fs::set_permissions(&socket_path, fs::Permissions::from_mode(0o600))?;

    let config = std::sync::Arc::new(config);
    let socket_path = std::sync::Arc::new(socket_path);

    loop {
        let (mut socket, _) = listener.accept().await?;
        let config = config.clone();
        let socket_path = socket_path.clone();

        tokio::spawn(async move {
            let mut buf = [0; 1024];
            let n = match socket.read(&mut buf).await {
                Ok(n) if n > 0 => n,
                _ => return,
            };

            let request = String::from_utf8_lossy(&buf[..n]);
            let mut parts = request.split_whitespace();
            let method = parts.next().unwrap_or("");
            let path = parts.next().unwrap_or("");

            if method == "GET" && path == "/status" {
                let response = StatusResponse {
                    pid: std::process::id(),
                    start_time,
                    version: env!("CARGO_PKG_VERSION").to_string(),
                    config: Some((*config).clone()),
                };

                if let Ok(json) = serde_json::to_string(&response) {
                    let resp = format!(
                        "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\n\r\n{}",
                        json.len(),
                        json
                    );
                    let _ = socket.write_all(resp.as_bytes()).await;
                }
            } else if method == "POST" && path == "/update" {
                let resp = "HTTP/1.1 200 OK\r\n\r\nRestarting...";
                let _ = socket.write_all(resp.as_bytes()).await;
                // Give some time for response to flush
                tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;

                tracing::info!("Restarting process...");
                // Remove socket file
                let _ = fs::remove_file(&*socket_path);

                let args: Vec<String> = std::env::args().collect();
                let exe = std::env::current_exe().unwrap_or_else(|_| PathBuf::from("sentrylogmon"));

                let err = Command::new(exe).args(&args[1..]).exec();

                tracing::error!("Failed to restart: {}", err);
                std::process::exit(1);
            } else {
                let resp = "HTTP/1.1 404 Not Found\r\n\r\n";
                let _ = socket.write_all(resp.as_bytes()).await;
            }
        });
    }
}

pub fn list_instances(socket_dir: &Path) -> Result<Vec<StatusResponse>> {
    let mut instances = Vec::new();

    if !socket_dir.exists() {
        return Ok(instances);
    }

    for entry in fs::read_dir(socket_dir)? {
        let entry = entry?;
        let path = entry.path();
        if path.extension().and_then(|s| s.to_str()) == Some("sock") {
            use std::io::{Read, Write};
            use std::os::unix::net::UnixStream;

            if let Ok(mut stream) = UnixStream::connect(&path) {
                if stream.write_all(b"GET /status HTTP/1.1\r\n\r\n").is_ok() {
                    let mut resp = String::new();
                    if stream.read_to_string(&mut resp).is_ok() {
                        // Parse body (skip headers)
                        if let Some(body_start) = resp.find("\r\n\r\n") {
                            let body = &resp[body_start + 4..];
                            if let Ok(status) = serde_json::from_str::<StatusResponse>(body) {
                                instances.push(status);
                            }
                        }
                    }
                }
            }
        }
    }
    Ok(instances)
}

pub fn request_update(socket_path: &Path) -> Result<()> {
    use std::io::{Read, Write};
    use std::os::unix::net::UnixStream;

    let mut stream = UnixStream::connect(socket_path)?;
    stream.write_all(b"POST /update HTTP/1.1\r\n\r\n")?;
    let mut resp = String::new();
    stream.read_to_string(&mut resp)?;
    // We assume 200 OK
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    fn get_temp_dir() -> PathBuf {
        let mut path = std::env::temp_dir();
        path.push(format!("sentrylogmon_test_{}", std::process::id()));
        path
    }

    #[test]
    fn test_ensure_secure_directory() {
        let path = get_temp_dir();
        if path.exists() {
            fs::remove_dir_all(&path).ok();
        }

        // 1. Normal creation
        assert!(ensure_secure_directory(&path).is_ok());
        assert!(path.exists());
        let meta = fs::metadata(&path).unwrap();
        assert_eq!(meta.permissions().mode() & 0o777, 0o700);

        // 2. Already exists with wrong perms
        fs::set_permissions(&path, fs::Permissions::from_mode(0o755)).unwrap();
        assert!(ensure_secure_directory(&path).is_ok());
        let meta = fs::metadata(&path).unwrap();
        assert_eq!(meta.permissions().mode() & 0o777, 0o700);

        // Cleanup
        fs::remove_dir_all(&path).ok();
    }
}
