use crate::sources::LogSource;
use anyhow::{Context, Result};
use async_trait::async_trait;
use std::io;
use std::pin::Pin;
use std::task::{Context as TaskContext, Poll};
use tokio::io::{AsyncBufReadExt, AsyncRead, ReadBuf};
use tokio::net::{TcpListener, UdpSocket};
use tokio::sync::broadcast;
use tokio::sync::mpsc;
use tracing::{error, info};

pub struct SyslogSource {
    name: String,
    address: String,
    close_tx: Option<broadcast::Sender<()>>,
}

impl SyslogSource {
    pub fn new(name: String, address: String) -> Self {
        Self {
            name,
            address,
            close_tx: None,
        }
    }

    async fn start_udp(
        address: String,
        tx: mpsc::Sender<Vec<u8>>,
        mut close_rx: broadcast::Receiver<()>,
    ) -> Result<()> {
        let socket = UdpSocket::bind(&address)
            .await
            .with_context(|| format!("Failed to bind UDP socket {}", address))?;
        info!("Listening on UDP {}", address);

        let mut buf = vec![0u8; 65536];

        loop {
            tokio::select! {
                _ = close_rx.recv() => {
                    break;
                }
                res = socket.recv_from(&mut buf) => {
                    match res {
                        Ok((n, _addr)) => {
                            if n > 0 {
                                let mut data = buf[..n].to_vec();
                                // Ensure newline
                                if data.last() != Some(&b'\n') {
                                    data.push(b'\n');
                                }
                                if tx.send(data).await.is_err() {
                                    break;
                                }
                            }
                        }
                        Err(e) => {
                            error!("Error reading from UDP: {}", e);
                        }
                    }
                }
            }
        }
        Ok(())
    }

    async fn start_tcp(
        address: String,
        tx: mpsc::Sender<Vec<u8>>,
        mut close_rx: broadcast::Receiver<()>,
    ) -> Result<()> {
        let listener = TcpListener::bind(&address)
            .await
            .with_context(|| format!("Failed to bind TCP listener {}", address))?;
        info!("Listening on TCP {}", address);

        loop {
            tokio::select! {
                _ = close_rx.recv() => {
                    break;
                }
                res = listener.accept() => {
                    match res {
                        Ok((mut socket, _addr)) => {
                            let tx = tx.clone();
                            tokio::spawn(async move {
                                let (reader, _) = socket.split();
                                let mut buf_reader = tokio::io::BufReader::new(reader);
                                let mut line = Vec::new();

                                loop {
                                    line.clear();
                                    match buf_reader.read_until(b'\n', &mut line).await {
                                        Ok(0) => break, // EOF
                                        Ok(_) => {
                                            // read_until includes delimiter
                                            // Ensure we send complete line with newline
                                            // Actually read_until ensures newline if present, or EOF.
                                            // If EOF reached without newline, we should append it?
                                            if line.last() != Some(&b'\n') {
                                                line.push(b'\n');
                                            }
                                            if tx.send(line.clone()).await.is_err() {
                                                break;
                                            }
                                        }
                                        Err(e) => {
                                            error!("Error reading from TCP connection: {}", e);
                                            break;
                                        }
                                    }
                                }
                            });
                        }
                        Err(e) => {
                            error!("Error accepting TCP connection: {}", e);
                        }
                    }
                }
            }
        }
        Ok(())
    }
}

#[async_trait]
impl LogSource for SyslogSource {
    fn name(&self) -> &str {
        &self.name
    }

    async fn stream(&mut self) -> Result<Box<dyn tokio::io::AsyncBufRead + Unpin + Send>> {
        let (tx, rx) = mpsc::channel::<Vec<u8>>(100);
        let (close_tx, _) = broadcast::channel(1);
        self.close_tx = Some(close_tx.clone());

        let close_rx = close_tx.subscribe();
        let address = self.address.clone();

        let protocol_addr = if address.starts_with("tcp:") {
            ("tcp", address.trim_start_matches("tcp:").to_string())
        } else if address.starts_with("udp:") {
            ("udp", address.trim_start_matches("udp:").to_string())
        } else {
            ("udp", address) // Default to UDP
        };

        match protocol_addr.0 {
            "tcp" => {
                let addr = protocol_addr.1;
                tokio::spawn(async move {
                    if let Err(e) = Self::start_tcp(addr, tx, close_rx).await {
                        error!("Syslog TCP source error: {}", e);
                    }
                });
            }
            "udp" => {
                let addr = protocol_addr.1;
                tokio::spawn(async move {
                    if let Err(e) = Self::start_udp(addr, tx, close_rx).await {
                        error!("Syslog UDP source error: {}", e);
                    }
                });
            }
            _ => unreachable!(),
        }

        let reader = ChannelReader {
            rx,
            current_chunk: None,
        };

        Ok(Box::new(tokio::io::BufReader::new(reader)))
    }

    async fn close(&mut self) -> Result<()> {
        if let Some(tx) = &self.close_tx {
            let _ = tx.send(());
        }
        Ok(())
    }
}

struct ChannelReader {
    rx: mpsc::Receiver<Vec<u8>>,
    current_chunk: Option<io::Cursor<Vec<u8>>>,
}

impl AsyncRead for ChannelReader {
    fn poll_read(
        mut self: Pin<&mut Self>,
        cx: &mut TaskContext<'_>,
        buf: &mut ReadBuf<'_>,
    ) -> Poll<io::Result<()>> {
        loop {
            if let Some(cursor) = &mut self.current_chunk {
                let pos = cursor.position() as usize;
                let inner = cursor.get_ref();
                let remaining = inner.len() - pos;

                if remaining > 0 {
                    let to_read = std::cmp::min(remaining, buf.remaining());
                    buf.put_slice(&inner[pos..pos + to_read]);
                    cursor.set_position((pos + to_read) as u64);
                    return Poll::Ready(Ok(()));
                } else {
                    self.current_chunk = None;
                }
            }

            match self.rx.poll_recv(cx) {
                Poll::Ready(Some(data)) => {
                    self.current_chunk = Some(io::Cursor::new(data));
                }
                Poll::Ready(None) => return Poll::Ready(Ok(())), // EOF
                Poll::Pending => return Poll::Pending,
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio::io::AsyncReadExt;

    #[tokio::test]
    async fn test_parsing_address() {
        let source = SyslogSource::new("test".to_string(), "udp:127.0.0.1:1234".to_string());
        assert_eq!(source.name(), "test");
        // We can't easily test private logic without exposing it or running stream()
    }

    // We can test ChannelReader logic
    #[tokio::test]
    async fn test_channel_reader() {
        let (tx, rx) = mpsc::channel(10);
        let mut reader = ChannelReader {
            rx,
            current_chunk: None,
        };

        tx.send(b"hello\n".to_vec()).await.unwrap();
        tx.send(b"world\n".to_vec()).await.unwrap();
        drop(tx);

        let mut buf = Vec::new();
        reader.read_to_end(&mut buf).await.unwrap();

        assert_eq!(buf, b"hello\nworld\n");
    }
}
