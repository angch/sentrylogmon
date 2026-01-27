use anyhow::Result;
use async_trait::async_trait;

pub mod file;
pub mod journalctl;
pub mod dmesg;
pub mod command;

#[async_trait]
pub trait LogSource: Send + Sync {
    /// Returns the name of the source
    fn name(&self) -> &str;

    /// Start streaming log lines
    async fn stream(&mut self) -> Result<Box<dyn tokio::io::AsyncBufRead + Unpin + Send>>;

    /// Close the log source and release resources
    async fn close(&mut self) -> Result<()>;
}
