//! CAEP (Continuous Access Evaluation Protocol) implementation.
//!
//! Implements OpenID CAEP 1.0 specification for real-time security event sharing.
//!
//! # Features
//! - SET (Security Event Token) generation and validation with ES256 default
//! - Event transmission to registered streams with logging integration
//! - Event reception and processing with cache integration
//! - Stream management
//!
//! # December 2025 Modernization
//! - Native async traits (Rust 2024)
//! - thiserror 2.0 for error handling
//! - Platform service integration (logging, cache)

#![forbid(unsafe_code)]
#![warn(missing_docs)]

pub mod error;
pub mod event;
pub mod handler;
pub mod receiver;
pub mod set;
pub mod stream;
pub mod transmitter;

pub use error::{CaepError, CaepResult};
pub use event::{CaepEvent, CaepEventType, SubjectIdentifier};
pub use handler::EventHandler;
pub use receiver::CaepReceiver;
pub use set::SecurityEventToken;
pub use stream::{DeliveryMethod, Stream, StreamConfig, StreamStatus};
pub use transmitter::CaepTransmitter;
