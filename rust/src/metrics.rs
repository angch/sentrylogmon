use axum::{routing::get, Router};
use prometheus::{Encoder, TextEncoder};
use std::net::SocketAddr;
use tokio::net::TcpListener;

pub async fn start_metrics_server(port: u16) -> anyhow::Result<()> {
    let app = Router::new()
        .route("/metrics", get(metrics_handler))
        .route("/healthz", get(healthz_handler));

    let addr = SocketAddr::from(([0, 0, 0, 0], port));
    tracing::info!("Starting metrics server on {}", addr);

    let listener = TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}

async fn metrics_handler() -> String {
    let encoder = TextEncoder::new();
    let metric_families = prometheus::gather();
    let mut buffer = vec![];
    if let Err(e) = encoder.encode(&metric_families, &mut buffer) {
        tracing::error!("Failed to encode metrics: {}", e);
        return String::new();
    }
    String::from_utf8(buffer).unwrap_or_default()
}

async fn healthz_handler() -> &'static str {
    "200 OK"
}
