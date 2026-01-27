use anyhow::Result;
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

pub struct Monitor {
    source: Box<dyn LogSource>,
    detector: Box<dyn Detector>,
    collector: Arc<Collector>,
    verbose: bool,
    stop_on_eof: bool,
    buffer: Arc<Mutex<Vec<String>>>,
    last_activity: Arc<Mutex<Instant>>,
}

impl Monitor {
    pub fn new(
        source: Box<dyn LogSource>,
        detector: Box<dyn Detector>,
        collector: Arc<Collector>,
        verbose: bool,
        stop_on_eof: bool,
    ) -> Self {
        Self {
            source,
            detector,
            collector,
            verbose,
            stop_on_eof,
            buffer: Arc::new(Mutex::new(Vec::new())),
            last_activity: Arc::new(Mutex::new(Instant::now())),
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
                    Self::send_to_sentry(&source_name, &msg).await;
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
            Self::send_to_sentry(self.source.name(), &msg).await;
        }
    }

    async fn force_flush(&self) {
        let mut buffer = self.buffer.lock().await;
        if !buffer.is_empty() {
            let msg = buffer.join("\n");
            buffer.clear();
            drop(buffer);
            Self::send_to_sentry(self.source.name(), &msg).await;
        }
    }

    async fn send_to_sentry(source_name: &str, message: &str) {
        sentry::with_scope(
            |scope| {
                scope.set_tag("source", source_name);
                
                // Try to extract timestamp
                let timestamp_regex = Regex::new(r"^\[\s*([0-9.]+)\]").unwrap();
                if let Some(caps) = timestamp_regex.captures(message) {
                    if let Some(ts) = caps.get(1) {
                        scope.set_tag("log_timestamp", ts.as_str());
                    }
                }

                scope.set_extra("raw_line", serde_json::json!(message));
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
