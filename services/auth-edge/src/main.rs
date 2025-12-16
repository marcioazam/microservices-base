mod config;
mod jwt;
mod mtls;
mod grpc;
mod error;
mod circuit_breaker;
mod rate_limiter;
mod middleware;
mod observability;
mod shutdown;

use std::net::SocketAddr;
use tonic::transport::Server;
use tracing::{info, Level};
use tracing_subscriber::FmtSubscriber;

pub mod proto {
    pub mod common {
        tonic::include_proto!("auth.common");
    }
    pub mod edge {
        tonic::include_proto!("auth.edge");
    }
}

use crate::grpc::AuthEdgeServiceImpl;
use proto::edge::auth_edge_service_server::AuthEdgeServiceServer;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let subscriber = FmtSubscriber::builder()
        .with_max_level(Level::INFO)
        .json()
        .init();

    info!("Starting Auth Edge Service");

    let config = config::Config::from_env()?;
    let addr: SocketAddr = format!("{}:{}", config.host, config.port).parse()?;

    let auth_edge_service = AuthEdgeServiceImpl::new(config).await?;

    info!("Auth Edge Service listening on {}", addr);

    Server::builder()
        .add_service(AuthEdgeServiceServer::new(auth_edge_service))
        .serve(addr)
        .await?;

    Ok(())
}
