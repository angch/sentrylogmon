mod config;
mod detectors;
mod ipc;
mod monitor;
mod sources;
mod sysstat;

use anyhow::Result;
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

        println!("PID\tSTART TIME (Unix)\tVERSION\tMONITORS");
        for inst in instances {
            let monitors = inst.config.map(|c| c.monitors.len()).unwrap_or(0);
            let start_time = match inst.start_time.duration_since(SystemTime::UNIX_EPOCH) {
                Ok(d) => d.as_secs(),
                Err(_) => 0,
            };
            println!(
                "{}\t{}\t{}\t{}",
                inst.pid, start_time, inst.version, monitors
            );
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
