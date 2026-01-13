//! Auth Edge Service - Main Entry Point
//!
//! Provides JWT validation, SPIFFE identity extraction, and token introspection
//! with modern observability and graceful shutdown.

mod config;
mod jwt;
mod mtls;
mod grpc;
mod error;
mod rate_limiter;
mod middleware;
mod observability;
mod shutdown;

use std::net::SocketAddr;
use std::sync::Arc;
use std::time::Duration;

use tonic::transport::Server;
use tracing::info;

use crate::config::Config;
use crate::grpc::AuthEdgeServiceImpl;
use crate::observability::{init_telemetry, TelemetryConfig, shutdown_telemetry};
use crate::shutdown::{ShutdownCoordinator, run_with_graceful_shutdown};

pub mod proto {
    pub mod common {
        tonic::include_proto!("auth.common");
    }
    pub mod edge {
        tonic::include_proto!("auth.edge");
    }
}

use proto::edge::auth_edge_service_server::AuthEdgeServiceServer;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Load configuration
    let config = Config::from_env()?;

    // Initialize OpenTelemetry
    let telemetry_config = TelemetryConfig {
        service_name: "auth-edge-service".to_string(),
        otlp_endpoint: config.otlp_endpoint_str().to_string(),
        sampling_ratio: 1.0,
        enable_console: true,
    };
    init_telemetry(&telemetry_config)?;

    info!("Starting Auth Edge Service");

    let addr: SocketAddr = format!("{}:{}", config.host, config.port).parse()?;

    // Create service implementation
    let auth_edge_service = AuthEdgeServiceImpl::new(config.clone()).await?;

    info!("Auth Edge Service listening on {}", addr);

    // Create shutdown coordinator
    let shutdown_coordinator = ShutdownCoordinator::new();
    let shutdown_timeout = Duration::from_secs(config.shutdown_timeout_seconds);

    // Build and run server with graceful shutdown
    let server = Server::builder()
        .add_service(AuthEdgeServiceServer::new(auth_edge_service))
        .serve(addr);

    run_with_graceful_shutdown(server, shutdown_coordinator, shutdown_timeout).await;

    // Cleanup OpenTelemetry
    shutdown_telemetry();

    info!("Auth Edge Service stopped");

    Ok(())
}
