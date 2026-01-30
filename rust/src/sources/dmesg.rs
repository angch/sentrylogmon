use anyhow::Result;
use async_trait::async_trait;
use tokio::io::{AsyncBufRead, BufReader};
use tokio::process::{Child, Command};

use super::LogSource;

pub struct DmesgSource {
    name: String,
    process: Option<Child>,
}

impl DmesgSource {
    pub fn new(name: String) -> Self {
        Self {
            name,
            process: None,
        }
    }
}

#[async_trait]
impl LogSource for DmesgSource {
    fn name(&self) -> &str {
        &self.name
    }

    async fn stream(&mut self) -> Result<Box<dyn AsyncBufRead + Unpin + Send>> {
        let mut child = Command::new("dmesg")
            .arg("-w")
            .stdout(std::process::Stdio::piped())
            .spawn()?;

        let stdout = child
            .stdout
            .take()
            .ok_or_else(|| anyhow::anyhow!("Failed to capture stdout from dmesg"))?;

        self.process = Some(child);
        Ok(Box::new(BufReader::new(stdout)))
    }

    async fn close(&mut self) -> Result<()> {
        if let Some(mut child) = self.process.take() {
            child.kill().await?;
        }
        Ok(())
    }
}
