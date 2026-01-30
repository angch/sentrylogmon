mod config;
mod detectors;
mod monitor;
mod sources;
mod sysstat;

use anyhow::Result;
use std::path::PathBuf;
use std::sync::Arc;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize tracing
    tracing_subscriber::fmt::init();

    // Load configuration
    let cfg = config::Config::load()?;

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
