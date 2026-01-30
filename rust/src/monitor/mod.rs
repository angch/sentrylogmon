use anyhow::Result;
use once_cell::sync::Lazy;
use regex::Regex;
use std::sync::Arc;
use tokio::io::AsyncBufReadExt;
use tokio::sync::Mutex;
use tokio::time::{Duration, Instant};

use crate::detectors::Detector;
use crate::sources::LogSource;
use crate::sysstat::Collector;

const MAX_BUFFER_SIZE: usize = 1000;
const FLUSH_INTERVAL: Duration = Duration::from_secs(5);

static TIMESTAMP_REGEX: Lazy<Regex> = Lazy::new(|| {
    Regex::new(r"^\[\s*([0-9.]+)\]").unwrap()
});

struct RateLimiter {
    limit: u32,
    window: Duration,
    count: u32,
    window_start: Instant,
}

impl RateLimiter {
    fn new(limit: u32, window: Duration) -> Self {
        Self {
            limit,
            window,
            count: 0,
            window_start: Instant::now(),
        }
    }

    fn allow(&mut self) -> bool {
        if self.limit == 0 {
            return true;
        }

        let now = Instant::now();
        if now.duration_since(self.window_start) > self.window {
            self.window_start = now;
            self.count = 0;
        }

        if self.count < self.limit {
            self.count += 1;
            true
        } else {
            false
        }
    }
}

pub struct Monitor {
    source: Box<dyn LogSource>,
    detector: Box<dyn Detector>,
    exclusion_detector: Option<Box<dyn Detector>>,
    collector: Arc<Collector>,
    verbose: bool,
    stop_on_eof: bool,
    buffer: Arc<Mutex<Vec<String>>>,
    last_activity: Arc<Mutex<Instant>>,
    rate_limiter: Arc<Mutex<RateLimiter>>,
}

impl Monitor {
    pub fn new(
        source: Box<dyn LogSource>,
        detector: Box<dyn Detector>,
        collector: Arc<Collector>,
        verbose: bool,
        stop_on_eof: bool,
        exclude_pattern: Option<String>,
        rate_limit_burst: Option<u32>,
        rate_limit_window: Option<String>,
    ) -> Self {
        let exclusion_detector = if let Some(pattern) = exclude_pattern {
            if !pattern.is_empty() {
                // Use generic detector for exclusion
                crate::detectors::get_detector("custom", &pattern).ok()
            } else {
                None
            }
        } else {
            None
        };

        let burst = rate_limit_burst.unwrap_or(0);
        let mut window = Duration::from_secs(0);
        if let Some(w) = rate_limit_window {
            if let Some(val) = w.strip_suffix("s") {
                if let Ok(secs) = val.parse::<u64>() {
                    window = Duration::from_secs(secs);
                }
            } else if let Some(val) = w.strip_suffix("m") {
                if let Ok(mins) = val.parse::<u64>() {
                    window = Duration::from_secs(mins * 60);
                }
            }
        }

        Self {
            source,
            detector,
            exclusion_detector,
            collector,
            verbose,
            stop_on_eof,
            buffer: Arc::new(Mutex::new(Vec::new())),
            last_activity: Arc::new(Mutex::new(Instant::now())),
            rate_limiter: Arc::new(Mutex::new(RateLimiter::new(burst, window))),
        }
    }

    pub async fn start(&mut self) -> Result<()> {
        if self.verbose {
            tracing::info!("Starting monitor for {}", self.source.name());
        }

        // Start flush timer
        let buffer = self.buffer.clone();
        let last_activity = self.last_activity.clone();
        let source_name = self.source.name().to_string();
        let collector = self.collector.clone();
        let rate_limiter = self.rate_limiter.clone();
        let verbose = self.verbose;

        tokio::spawn(async move {
            loop {
                tokio::time::sleep(FLUSH_INTERVAL).await;

                let elapsed = last_activity.lock().await.elapsed();
                if elapsed < FLUSH_INTERVAL - Duration::from_millis(100) {
                    continue;
                }

                let mut buf = buffer.lock().await;
                if !buf.is_empty() {
                    let msg = buf.join("\n");
                    buf.clear();
                    drop(buf);
                    Self::send_to_sentry(&source_name, &msg, Some(&collector), &rate_limiter, verbose).await;
                }
            }
        });

        loop {
            let reader = match self.source.stream().await {
                Ok(r) => r,
                Err(e) => {
                    tracing::error!("Error starting source {}: {}", self.source.name(), e);
                    tokio::time::sleep(Duration::from_secs(1)).await;
                    continue;
                }
            };

            let mut lines = reader.lines();
            while let Ok(Some(line)) = lines.next_line().await {
                let line_bytes = line.as_bytes();
                if self.detector.detect(line_bytes) {
                    if let Some(ed) = &self.exclusion_detector {
                        if ed.detect(line_bytes) {
                            if self.verbose {
                                tracing::info!("[{}] Excluded: {}", self.source.name(), line);
                            }
                            continue;
                        }
                    }

                    if self.verbose {
                        tracing::info!("[{}] Matched: {}", self.source.name(), line);
                    }
                    self.process_match(line).await;
                }
            }

            // Flush remaining buffer
            self.force_flush().await;

            if self.stop_on_eof {
                if self.verbose {
                    tracing::info!("Monitor for {} stopped (StopOnEOF set)", self.source.name());
                }
                break;
            }

            if self.verbose {
                tracing::info!("Monitor for {} stopped, restarting in 1s...", self.source.name());
            }
            tokio::time::sleep(Duration::from_secs(1)).await;
        }

        Ok(())
    }

    async fn process_match(&self, line: String) {
        *self.last_activity.lock().await = Instant::now();

        let mut buffer = self.buffer.lock().await;
        buffer.push(line);

        if buffer.len() >= MAX_BUFFER_SIZE {
            let msg = buffer.join("\n");
            buffer.clear();
            drop(buffer);
            Self::send_to_sentry(self.source.name(), &msg, Some(&self.collector), &self.rate_limiter, self.verbose).await;
        }
    }

    async fn force_flush(&self) {
        let mut buffer = self.buffer.lock().await;
        if !buffer.is_empty() {
            let msg = buffer.join("\n");
            buffer.clear();
            drop(buffer);
            Self::send_to_sentry(self.source.name(), &msg, Some(&self.collector), &self.rate_limiter, self.verbose).await;
        }
    }

    async fn send_to_sentry(source_name: &str, message: &str, collector: Option<&Collector>, rate_limiter: &Mutex<RateLimiter>, verbose: bool) {
        {
            let mut limiter = rate_limiter.lock().await;
            if !limiter.allow() {
                if verbose {
                    tracing::info!("[{}] Rate limited, dropping event.", source_name);
                }
                return;
            }
        }

        let state_json = if let Some(c) = collector {
            let state = c.get_state().await;
            serde_json::to_value(state).ok()
        } else {
            None
        };

        sentry::with_scope(
            |scope| {
                scope.set_tag("source", source_name);
                
                // Try to extract timestamp
                if let Some(caps) = TIMESTAMP_REGEX.captures(message) {
                    if let Some(ts) = caps.get(1) {
                        scope.set_tag("log_timestamp", ts.as_str());
                    }
                }

                scope.set_extra("raw_line", serde_json::json!(message));

                if let Some(json) = state_json {
                    scope.set_extra("Server State", json);
                }
            },
            || {
                sentry::capture_message(message, sentry::Level::Error);
            },
        );
    }

    pub async fn close(&mut self) -> Result<()> {
        self.source.close().await
    }
}
