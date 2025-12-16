mod config;
mod dpop;
mod jwt;
mod refresh;
mod jwks;
mod kms;
mod storage;
mod grpc;
mod error;

use std::net::SocketAddr;
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

use crate::grpc::TokenServiceImpl;
use proto::token::token_service_server::TokenServiceServer;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize tracing
    let subscriber = FmtSubscriber::builder()
        .with_max_level(Level::INFO)
        .json()
        .init();

    info!("Starting Token Service");

    let config = config::Config::from_env()?;
    let addr: SocketAddr = format!("{}:{}", config.host, config.port).parse()?;

    let token_service = TokenServiceImpl::new(config).await?;

    info!("Token Service listening on {}", addr);

    Server::builder()
        .add_service(TokenServiceServer::new(token_service))
        .serve(addr)
        .await?;

    Ok(())
}
