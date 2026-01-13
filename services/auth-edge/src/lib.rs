//! Auth Edge Service - Ultra-low latency JWT validation and edge routing.
//!
//! This crate provides the core functionality for the Auth Edge Service,
//! including JWT validation with type-state pattern, SPIFFE/mTLS support,
//! and integration with platform services.

#![forbid(unsafe_code)]
#![warn(missing_docs)]

pub mod config;
pub mod crypto;
pub mod error;
pub mod grpc;
pub mod jwt;
pub mod middleware;
pub mod mtls;
pub mod observability;
pub mod rate_limiter;
pub mod shutdown;

// Include generated protobuf code
pub mod proto {
    // crypto-service client
    pub mod crypto {
        pub mod v1 {
            tonic::include_proto!("crypto.v1");
        }
    }

    // auth-edge server
    pub mod auth {
        pub mod v1 {
            tonic::include_proto!("auth.v1");
        }
    }
}

pub use config::Config;
pub use crypto::{CryptoClient, CryptoClientConfig, CryptoError, EncryptedCacheClient};
pub use error::{AuthEdgeError, ErrorCode, ErrorResponse};
