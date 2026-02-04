use axum::{
    extract::Query,
    http::{header, StatusCode},
    response::IntoResponse,
    routing::get,
    Router,
};
use prometheus::{Encoder, TextEncoder};
use serde::Deserialize;
use std::net::SocketAddr;
use tokio::net::TcpListener;

#[derive(Deserialize)]
struct ProfileParams {
    seconds: Option<u64>,
}

pub async fn start_metrics_server(port: u16) -> anyhow::Result<()> {
    let app = Router::new()
        .route("/metrics", get(metrics_handler))
        .route("/healthz", get(healthz_handler))
        .route("/debug/pprof/profile", get(pprof_profile_handler));

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

async fn pprof_profile_handler(Query(params): Query<ProfileParams>) -> impl IntoResponse {
    let seconds = params.seconds.unwrap_or(30).clamp(1, 60);

    let result = tokio::task::spawn_blocking(move || {
        let guard = pprof::ProfilerGuardBuilder::default()
            .frequency(100)
            .blocklist(&["libc", "libgcc", "pthread", "vdso"])
            .build()
            .map_err(|e| e.to_string())?;

        std::thread::sleep(std::time::Duration::from_secs(seconds));

        let report = guard.report().build().map_err(|e| e.to_string())?;
        let profile = report.pprof().map_err(|e| e.to_string())?;

        let mut body = Vec::new();
        use pprof::protos::Message;
        profile
            .write_to_vec(&mut body)
            .map_err(|e| e.to_string())?;

        Ok::<Vec<u8>, String>(body)
    })
    .await;

    match result {
        Ok(Ok(body)) => (
            [(header::CONTENT_TYPE, "application/octet-stream")],
            body,
        )
            .into_response(),
        Ok(Err(e)) => (StatusCode::INTERNAL_SERVER_ERROR, e).into_response(),
        Err(e) => (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()).into_response(),
    }
}
