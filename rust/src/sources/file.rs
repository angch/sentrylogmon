use anyhow::Result;
use async_trait::async_trait;
use std::path::PathBuf;
use tokio::fs::File;
use tokio::io::{AsyncBufRead, BufReader};

use super::LogSource;

pub struct FileSource {
    name: String,
    path: PathBuf,
}

impl FileSource {
    pub fn new(name: String, path: PathBuf) -> Self {
        Self { name, path }
    }
}

#[async_trait]
impl LogSource for FileSource {
    fn name(&self) -> &str {
        &self.name
    }

    async fn stream(&mut self) -> Result<Box<dyn AsyncBufRead + Unpin + Send>> {
        let file = File::open(&self.path).await?;
        Ok(Box::new(BufReader::new(file)))
    }

    async fn close(&mut self) -> Result<()> {
        Ok(())
    }
}
