//! Token Service - JWT generation, DPoP validation, refresh token rotation.
//!
//! Uses platform libraries for caching, logging, and circuit breaker.

mod config;
mod dpop;
mod error;
mod grpc;
mod jwks;
mod jwt;
mod kms;
pub mod metrics;
mod refresh;
mod storage;

use crate::config::Config;
use crate::grpc::TokenServiceImpl;
use rust_common::{CacheClient, LoggingClient};
use std::net::SocketAddr;
use std::sync::Arc;
use tonic::transport::Server;
use tracing::{info, Level};
use tracing_subscriber::FmtSubscriber;

pub mod proto {
    pub mod common {
        tonic::include_proto!("auth.common");
    }
    pub mod token {
        tonic::include_proto!("auth.token");
    }
}

use proto::token::token_service_server::TokenServiceServer;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize tracing
    let _guard = FmtSubscriber::builder()
        .with_max_level(Level::INFO)
        .json()
        .try_init();

    info!("Starting Token Service");

    let config = Config::from_env()?;
    let addr: SocketAddr = format!("{}:{}", config.host, config.port).parse()?;

    // Initialize platform clients
    let cache_client = Arc::new(
        CacheClient::new(config.cache.clone())
            .await
            .expect("Failed to create cache client"),
    );

    let logging_client = Arc::new(
        LoggingClient::new(config.logging.clone())
            .await
            .expect("Failed to create logging client"),
    );

    info!(
        cache_namespace = %cache_client.namespace(),
        logging_service = %logging_client.service_id(),
        "Platform clients initialized"
    );

    let token_service = TokenServiceImpl::new(
        config,
        cache_client,
        logging_client,
    ).await?;

    info!("Token Service listening on {}", addr);

    // Graceful shutdown handling
    let (shutdown_tx, shutdown_rx) = tokio::sync::oneshot::channel::<()>();

    tokio::spawn(async move {
        tokio::signal::ctrl_c()
            .await
            .expect("Failed to listen for ctrl+c");
        info!("Shutdown signal received");
        let _ = shutdown_tx.send(());
    });

    Server::builder()
        .add_service(TokenServiceServer::new(token_service))
        .serve_with_shutdown(addr, async {
            shutdown_rx.await.ok();
        })
        .await?;

    info!("Token Service shutdown complete");
    Ok(())
}
