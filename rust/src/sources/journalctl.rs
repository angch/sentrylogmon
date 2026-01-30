use anyhow::Result;
use async_trait::async_trait;
use tokio::io::{AsyncBufRead, BufReader};
use tokio::process::{Child, Command};

use super::LogSource;

pub struct JournalctlSource {
    name: String,
    args: Vec<String>,
    process: Option<Child>,
}

impl JournalctlSource {
    pub fn new(name: String, args_str: &str) -> Self {
        let args: Vec<String> = args_str.split_whitespace().map(String::from).collect();
        Self {
            name,
            args,
            process: None,
        }
    }
}

#[async_trait]
impl LogSource for JournalctlSource {
    fn name(&self) -> &str {
        &self.name
    }

    async fn stream(&mut self) -> Result<Box<dyn AsyncBufRead + Unpin + Send>> {
        let mut child = Command::new("journalctl")
            .args(&self.args)
            .stdout(std::process::Stdio::piped())
            .spawn()?;

        let stdout = child
            .stdout
            .take()
            .ok_or_else(|| anyhow::anyhow!("Failed to capture stdout from journalctl"))?;

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
