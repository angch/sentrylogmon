use anyhow::{Context, Result};
use clap::Parser;
use serde::{Deserialize, Serialize};
use std::path::PathBuf;

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct SentryConfig {
    pub dsn: String,
    #[serde(default = "default_environment")]
    pub environment: String,
    #[serde(default)]
    pub release: String,
}

fn default_environment() -> String {
    "production".to_string()
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MonitorConfig {
    pub name: String,
    #[serde(rename = "type")]
    pub monitor_type: String,
    #[serde(default)]
    pub path: String,
    #[serde(default)]
    pub args: String,
    #[serde(default = "default_pattern")]
    pub pattern: String,
    #[serde(default)]
    pub format: String,
    #[serde(default)]
    pub exclude_pattern: String,
    #[serde(default)]
    pub rate_limit_burst: Option<u32>,
    #[serde(default)]
    pub rate_limit_window: Option<String>,
}

fn default_pattern() -> String {
    "Error".to_string()
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct FileConfig {
    #[serde(default)]
    pub sentry: SentryConfig,
    #[serde(default)]
    pub monitors: Vec<MonitorConfig>,
}

#[derive(Parser, Debug)]
#[command(name = "sentrylogmon")]
#[command(about = "Lightweight log monitoring with Sentry integration", long_about = None)]
pub struct Args {
    /// Path to configuration file
    #[arg(long)]
    pub config: Option<PathBuf>,

    /// Sentry DSN
    #[arg(long, env = "SENTRY_DSN")]
    pub dsn: Option<String>,

    /// Monitor dmesg output
    #[arg(long)]
    pub dmesg: bool,

    /// Monitor a log file
    #[arg(long)]
    pub file: Option<PathBuf>,

    /// Monitor journalctl output (pass args)
    #[arg(long)]
    pub journalctl: Option<String>,

    /// Monitor custom command output
    #[arg(long)]
    pub command: Option<String>,

    /// Monitor syslog (e.g. udp:127.0.0.1:5514)
    #[arg(long)]
    pub syslog: Option<String>,

    /// Detector format (dmesg, nginx, custom)
    #[arg(long)]
    pub format: Option<String>,

    /// Pattern to match
    #[arg(long, default_value = "Error")]
    pub pattern: String,

    /// Pattern to exclude from reporting
    #[arg(long)]
    pub exclude: Option<String>,

    /// Sentry environment
    #[arg(long, default_value = "production")]
    pub environment: String,

    /// Sentry release version
    #[arg(long)]
    pub release: Option<String>,

    /// Verbose logging
    #[arg(short, long)]
    pub verbose: bool,

    /// Run once and exit when input stream ends
    #[arg(long)]
    pub oneshot: bool,

    /// List running instances
    #[arg(long)]
    pub status: bool,

    /// Update/Restart all running instances
    #[arg(long)]
    pub update: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub sentry: SentryConfig,
    pub monitors: Vec<MonitorConfig>,
    pub verbose: bool,
    pub oneshot: bool,
    #[serde(skip)]
    pub status: bool,
    #[serde(skip)]
    pub update: bool,
}

impl Config {
    pub fn load() -> Result<Self> {
        let args = Args::parse();
        Self::from_args(args)
    }

    fn from_args(args: Args) -> Result<Self> {
        let config = if let Some(config_path) = &args.config {
            let content = std::fs::read_to_string(config_path)
                .with_context(|| format!("Failed to read config file: {:?}", config_path))?;
            let file_config: FileConfig = serde_yaml::from_str(&content)
                .with_context(|| format!("Failed to parse config file: {:?}", config_path))?;

            let mut cfg = Config {
                sentry: file_config.sentry,
                monitors: file_config.monitors,
                verbose: args.verbose,
                oneshot: args.oneshot,
                status: args.status,
                update: args.update,
            };

            // Override with CLI args if provided
            if let Some(dsn) = &args.dsn {
                cfg.sentry.dsn = dsn.clone();
            }
            if args.environment != "production" {
                cfg.sentry.environment = args.environment.clone();
            }
            if let Some(release) = &args.release {
                cfg.sentry.release = release.clone();
            }

            cfg
        } else {
            // CLI mode
            let mut monitors = Vec::new();
            let format_arg = args.format.clone().unwrap_or_default();

            if args.dmesg {
                monitors.push(MonitorConfig {
                    name: "dmesg".to_string(),
                    monitor_type: "dmesg".to_string(),
                    path: String::new(),
                    args: String::new(),
                    pattern: args.pattern.clone(),
                    format: if format_arg.is_empty() {
                        "dmesg".to_string()
                    } else {
                        format_arg.clone()
                    },
                    exclude_pattern: args.exclude.clone().unwrap_or_default(),
                    rate_limit_burst: None,
                    rate_limit_window: None,
                });
            } else if let Some(file_path) = &args.file {
                monitors.push(MonitorConfig {
                    name: "file".to_string(),
                    monitor_type: "file".to_string(),
                    path: file_path.to_string_lossy().to_string(),
                    args: String::new(),
                    pattern: args.pattern.clone(),
                    format: format_arg.clone(),
                    exclude_pattern: args.exclude.clone().unwrap_or_default(),
                    rate_limit_burst: None,
                    rate_limit_window: None,
                });
            } else if let Some(journalctl_args) = &args.journalctl {
                monitors.push(MonitorConfig {
                    name: "journalctl".to_string(),
                    monitor_type: "journalctl".to_string(),
                    path: String::new(),
                    args: journalctl_args.clone(),
                    pattern: args.pattern.clone(),
                    format: format_arg.clone(),
                    exclude_pattern: args.exclude.clone().unwrap_or_default(),
                    rate_limit_burst: None,
                    rate_limit_window: None,
                });
            } else if let Some(cmd) = &args.command {
                monitors.push(MonitorConfig {
                    name: "command".to_string(),
                    monitor_type: "command".to_string(),
                    path: String::new(),
                    args: cmd.clone(),
                    pattern: args.pattern.clone(),
                    format: format_arg.clone(),
                    exclude_pattern: args.exclude.clone().unwrap_or_default(),
                    rate_limit_burst: None,
                    rate_limit_window: None,
                });
            } else if let Some(syslog_addr) = &args.syslog {
                monitors.push(MonitorConfig {
                    name: "syslog".to_string(),
                    monitor_type: "syslog".to_string(),
                    path: syslog_addr.clone(),
                    args: String::new(),
                    pattern: args.pattern.clone(),
                    format: format_arg.clone(),
                    exclude_pattern: args.exclude.clone().unwrap_or_default(),
                    rate_limit_burst: None,
                    rate_limit_window: None,
                });
            }

            Config {
                sentry: SentryConfig {
                    dsn: args.dsn.unwrap_or_default(),
                    environment: args.environment,
                    release: args.release.unwrap_or_default(),
                },
                monitors,
                verbose: args.verbose,
                oneshot: args.oneshot,
                status: args.status,
                update: args.update,
            }
        };

        if config.status || config.update {
            return Ok(config);
        }

        if config.sentry.dsn.is_empty() {
            anyhow::bail!("Sentry DSN is required. Set via --dsn flag, SENTRY_DSN environment variable, or config file");
        }

        if config.monitors.is_empty() {
            anyhow::bail!("No monitors configured. Use --file, --dmesg, --journalctl, --command, or config file.");
        }

        Ok(config)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_config_from_args_with_format() {
        let args = Args::parse_from(&[
            "sentrylogmon",
            "--file", "/tmp/test.log",
            "--format", "nginx",
            "--dsn", "https://example.com"
        ]);

        let config = Config::from_args(args).unwrap();
        assert_eq!(config.monitors.len(), 1);
        assert_eq!(config.monitors[0].format, "nginx");
    }

    #[test]
    fn test_config_from_args_without_format() {
        let args = Args::parse_from(&[
            "sentrylogmon",
            "--file", "/tmp/test.log",
            "--dsn", "https://example.com"
        ]);

        let config = Config::from_args(args).unwrap();
        assert_eq!(config.monitors.len(), 1);
        assert_eq!(config.monitors[0].format, "");
    }

    #[test]
    fn test_config_from_args_dmesg_default_format() {
         let args = Args::parse_from(&[
            "sentrylogmon",
            "--dmesg",
            "--dsn", "https://example.com"
        ]);

        let config = Config::from_args(args).unwrap();
        assert_eq!(config.monitors.len(), 1);
        assert_eq!(config.monitors[0].format, "dmesg");
    }

    #[test]
    fn test_config_from_args_dmesg_override_format() {
         let args = Args::parse_from(&[
            "sentrylogmon",
            "--dmesg",
            "--format", "custom",
            "--dsn", "https://example.com"
        ]);

        let config = Config::from_args(args).unwrap();
        assert_eq!(config.monitors.len(), 1);
        assert_eq!(config.monitors[0].format, "custom");
    }
}
