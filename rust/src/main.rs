mod config;
mod detectors;
mod ipc;
mod metrics;
mod monitor;
mod sources;
mod sysstat;

use anyhow::Result;
use chrono::prelude::*;
use std::cmp::max;
use std::path::PathBuf;
use std::sync::Arc;
use std::time::SystemTime;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize tracing
    tracing_subscriber::fmt::init();

    // Load configuration
    let cfg = config::Config::load()?;

    if cfg.status {
        let socket_dir = PathBuf::from("/tmp/sentrylogmon");
        let instances = ipc::list_instances(&socket_dir)?;

        if is_terminal() {
            print_instance_table(instances);
        } else {
            println!("{}", serde_json::to_string_pretty(&instances)?);
        }
        return Ok(());
    }

    if cfg.update {
        let socket_dir = PathBuf::from("/tmp/sentrylogmon");
        let instances = ipc::list_instances(&socket_dir)?;
        for inst in instances {
            let socket_path = socket_dir.join(format!("sentrylogmon.{}.sock", inst.pid));
            println!("Requesting update for PID {}...", inst.pid);
            if let Err(e) = ipc::request_update(&socket_path) {
                println!("Failed to update PID {}: {}", inst.pid, e);
            } else {
                println!("Update requested for PID {}", inst.pid);
            }
        }
        return Ok(());
    }

    if cfg.sentry.dsn.is_empty() {
        anyhow::bail!("Sentry DSN is required");
    }

    // Initialize Sentry
    let _guard = sentry::init((
        cfg.sentry.dsn.clone(),
        sentry::ClientOptions {
            release: if !cfg.sentry.release.is_empty() {
                Some(cfg.sentry.release.clone().into())
            } else {
                None
            },
            environment: Some(cfg.sentry.environment.clone().into()),
            ..Default::default()
        },
    ));

    if cfg.verbose {
        tracing::info!(
            "Initialized Sentry (env={}, release={})",
            cfg.sentry.environment,
            cfg.sentry.release
        );
    }

    if cfg.monitors.is_empty() {
        anyhow::bail!("No monitors configured");
    }

    // Start system stats collector
    let collector = Arc::new(sysstat::Collector::new());
    collector.run().await;

    // Start IPC server
    let socket_dir = PathBuf::from("/tmp/sentrylogmon");
    if let Err(e) = ipc::ensure_secure_directory(&socket_dir) {
        tracing::error!("Failed to ensure secure IPC directory: {}", e);
    } else {
        let socket_path = socket_dir.join(format!("sentrylogmon.{}.sock", std::process::id()));

        let cfg_clone = cfg.clone();
        tokio::spawn(async move {
            if let Err(e) = ipc::start_server(socket_path, cfg_clone, SystemTime::now()).await {
                tracing::error!("IPC Server error: {}", e);
            }
        });
    }

    // Start Metrics Server
    if cfg.metrics_port > 0 {
        let port = cfg.metrics_port;
        tokio::spawn(async move {
            if let Err(e) = metrics::start_metrics_server(port).await {
                tracing::error!("Metrics Server error: {}", e);
            }
        });
    }

    // Start monitors
    let mut handles = Vec::new();

    for mon_cfg in cfg.monitors.iter() {
        let source: Box<dyn sources::LogSource> = match mon_cfg.monitor_type.as_str() {
            "file" => {
                if mon_cfg.path.is_empty() {
                    tracing::warn!("Skipping file monitor '{}': path is empty", mon_cfg.name);
                    continue;
                }
                Box::new(sources::file::FileSource::new(
                    mon_cfg.name.clone(),
                    PathBuf::from(&mon_cfg.path),
                ))
            }
            "journalctl" => Box::new(sources::journalctl::JournalctlSource::new(
                mon_cfg.name.clone(),
                &mon_cfg.args,
            )),
            "dmesg" => Box::new(sources::dmesg::DmesgSource::new(mon_cfg.name.clone())),
            "command" => {
                let parts: Vec<&str> = mon_cfg.args.split_whitespace().collect();
                if parts.is_empty() {
                    tracing::warn!(
                        "Skipping command monitor '{}': command is empty",
                        mon_cfg.name
                    );
                    continue;
                }
                let cmd = parts[0].to_string();
                let args: Vec<String> = parts[1..].iter().map(|s| s.to_string()).collect();
                Box::new(sources::command::CommandSource::new(
                    mon_cfg.name.clone(),
                    cmd,
                    args,
                ))
            }
            "syslog" => Box::new(sources::syslog::SyslogSource::new(
                mon_cfg.name.clone(),
                mon_cfg.path.clone(),
            )),
            _ => {
                tracing::warn!("Unknown monitor type: {}", mon_cfg.monitor_type);
                continue;
            }
        };

        let detector_format = determine_detector_format(mon_cfg);
        let detector = match detectors::get_detector(&detector_format, &mon_cfg.pattern) {
            Ok(d) => d,
            Err(e) => {
                tracing::error!(
                    "Failed to create detector for monitor '{}': {}",
                    mon_cfg.name,
                    e
                );
                continue;
            }
        };

        let mut monitor = monitor::Monitor::new(
            source,
            detector,
            collector.clone(),
            cfg.verbose,
            cfg.oneshot,
            Some(mon_cfg.exclude_pattern.clone()),
            mon_cfg.rate_limit_burst,
            mon_cfg.rate_limit_window.clone(),
        );

        let handle = tokio::spawn(async move {
            if let Err(e) = monitor.start().await {
                tracing::error!("Monitor error: {}", e);
            }
            if let Err(e) = monitor.close().await {
                tracing::error!("Error closing monitor: {}", e);
            }
        });

        handles.push(handle);
    }

    if handles.is_empty() {
        anyhow::bail!("No valid monitors to start");
    }

    // Wait for all monitors
    for handle in handles {
        let _ = handle.await;
    }

    Ok(())
}

fn determine_detector_format(mon_cfg: &config::MonitorConfig) -> String {
    if !mon_cfg.format.is_empty() {
        return mon_cfg.format.clone();
    }
    if !mon_cfg.pattern.is_empty() {
        return "custom".to_string();
    }
    if mon_cfg.monitor_type == "dmesg" {
        return "dmesg".to_string();
    }
    "custom".to_string()
}

fn is_terminal() -> bool {
    unsafe { libc::isatty(libc::STDOUT_FILENO) == 1 }
}

fn format_duration(seconds: u64) -> String {
    let days = seconds / 86400;
    let hours = (seconds % 86400) / 3600;
    let minutes = (seconds % 3600) / 60;
    let secs = seconds % 60;

    if days > 0 {
        format!("{}d {}h {}m", days, hours, minutes)
    } else if hours > 0 {
        format!("{}h {}m {}s", hours, minutes, secs)
    } else if minutes > 0 {
        format!("{}m {}s", minutes, secs)
    } else {
        format!("{}s", secs)
    }
}

fn format_bytes(b: u64) -> String {
    const UNIT: u64 = 1024;
    if b < UNIT {
        return format!("{} B", b);
    }
    let mut div = UNIT;
    let mut exp = 0;
    let mut n = b / UNIT;
    while n >= UNIT {
        div *= UNIT;
        exp += 1;
        n /= UNIT;
    }
    let suffix = "KMGTPE".chars().nth(exp).unwrap_or('?');
    format!("{:.1} {}iB", b as f64 / div as f64, suffix)
}

fn print_instance_table(instances: Vec<ipc::StatusResponse>) {
    if instances.is_empty() {
        println!("No running instances found.");
        return;
    }

    let output = format_instance_table(&instances, SystemTime::now());
    print!("{}", output);
}

struct InstanceRow {
    pid: String,
    started: String,
    uptime: String,
    mem: String,
    version: String,
    monitors: String,
}

fn format_instance_table(instances: &[ipc::StatusResponse], now: SystemTime) -> String {
    let mut rows = Vec::with_capacity(instances.len());

    for inst in instances {
        let start_dt: DateTime<Local> = inst.start_time.into();
        let start_str = start_dt.format("%Y-%m-%d %H:%M:%S").to_string();

        let uptime_secs = now
            .duration_since(inst.start_time)
            .map(|d| d.as_secs())
            .unwrap_or(0);
        let uptime_str = format_duration(uptime_secs);

        let mem_str = format_bytes(inst.memory_alloc);

        let version = if inst.version.is_empty() {
            "-".to_string()
        } else {
            inst.version.clone()
        };

        rows.push(InstanceRow {
            pid: inst.pid.to_string(),
            started: start_str,
            uptime: uptime_str,
            mem: mem_str,
            version,
            monitors: format_details(inst.config.as_ref()),
        });
    }

    let headers = ["PID", "STARTED", "UPTIME", "MEM", "VERSION", "MONITORS"];
    let mut widths = [
        headers[0].len(),
        headers[1].len(),
        headers[2].len(),
        headers[3].len(),
        headers[4].len(),
    ];

    for row in &rows {
        widths[0] = max(widths[0], row.pid.len());
        widths[1] = max(widths[1], row.started.len());
        widths[2] = max(widths[2], row.uptime.len());
        widths[3] = max(widths[3], row.mem.len());
        widths[4] = max(widths[4], row.version.len());
    }

    let mut output = String::new();
    output.push_str(&format!(
        "{:<w0$} {:<w1$} {:<w2$} {:<w3$} {:<w4$} {}\n",
        headers[0],
        headers[1],
        headers[2],
        headers[3],
        headers[4],
        headers[5],
        w0 = widths[0],
        w1 = widths[1],
        w2 = widths[2],
        w3 = widths[3],
        w4 = widths[4]
    ));

    for row in rows {
        output.push_str(&format!(
            "{:<w0$} {:<w1$} {:<w2$} {:<w3$} {:<w4$} {}\n",
            row.pid,
            row.started,
            row.uptime,
            row.mem,
            row.version,
            row.monitors,
            w0 = widths[0],
            w1 = widths[1],
            w2 = widths[2],
            w3 = widths[3],
            w4 = widths[4]
        ));
    }

    output
}

fn format_details(config: Option<&config::Config>) -> String {
    let Some(cfg) = config else {
        return "-".to_string();
    };

    let limit = 60;
    let monitors = &cfg.monitors;
    let mut parts = Vec::new();

    for m in monitors {
        parts.push(format!("{}({})", m.name, m.monitor_type));
    }

    let mut buffer = String::new();
    for (i, part) in parts.iter().enumerate() {
        let sep = if i > 0 { ", " } else { "" };

        if i == 0 {
            let remaining = parts.len() - 1;
            let suffix_len = if remaining > 0 { 12 } else { 0 };

            let mut part_display = part.clone();
            if part.len() + suffix_len > limit {
                let avail = limit - suffix_len - 3; // -3 for ...
                let avail = if avail < 10 { 10 } else { avail };
                if part.len() > avail {
                    let safe_part: String = part.chars().take(avail).collect();
                    part_display = format!("{}...", safe_part);
                }
            }
            buffer.push_str(&part_display);
            continue;
        }

        let reserved = if i == parts.len() - 1 { 0 } else { 12 };

        if buffer.len() + sep.len() + part.len() + reserved <= limit {
            buffer.push_str(sep);
            buffer.push_str(part);
        } else {
            let remaining = parts.len() - i;
            buffer.push_str(&format!(" (+{} more)", remaining));
            break;
        }
    }

    if buffer.is_empty() {
        "-".to_string()
    } else {
        buffer
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::{Config, MonitorConfig, SentryConfig};
    use crate::ipc::StatusResponse;
    use std::time::Duration;

    fn sample_config(name: &str, monitor_type: &str) -> Config {
        Config {
            sentry: SentryConfig::default(),
            monitors: vec![MonitorConfig {
                name: name.to_string(),
                monitor_type: monitor_type.to_string(),
                path: String::new(),
                args: String::new(),
                pattern: "Error".to_string(),
                format: String::new(),
                exclude_pattern: String::new(),
                rate_limit_burst: None,
                rate_limit_window: None,
            }],
            verbose: false,
            oneshot: false,
            status: false,
            update: false,
            metrics_port: 0,
        }
    }

    #[test]
    fn format_instance_table_aligns_monitors_column() {
        let start_time = SystemTime::UNIX_EPOCH + Duration::from_secs(1_000);
        let now = start_time + Duration::from_secs(65);

        let instances = vec![
            StatusResponse {
                pid: 9,
                start_time,
                version: "0.1.0".to_string(),
                config: Some(sample_config("alpha", "file")),
                memory_alloc: 1024 * 1024,
            },
            StatusResponse {
                pid: 1010,
                start_time,
                version: "0.10.0".to_string(),
                config: Some(sample_config("beta", "file")),
                memory_alloc: 2048 * 1024,
            },
        ];

        let output = format_instance_table(&instances, now);
        let lines: Vec<&str> = output.lines().collect();
        assert!(lines.len() >= 3);

        let header_monitors_idx = lines[0].find("MONITORS").expect("header monitors");
        let first_monitors_idx = lines[1].find("alpha(file)").expect("first monitors");
        let second_monitors_idx = lines[2].find("beta(file)").expect("second monitors");

        assert_eq!(header_monitors_idx, first_monitors_idx);
        assert_eq!(header_monitors_idx, second_monitors_idx);
    }
}
