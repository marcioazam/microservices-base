//! CAEP (Continuous Access Evaluation Protocol) implementation.
//!
//! Implements OpenID CAEP 1.0 specification for real-time security event sharing.
//!
//! # Features
//! - SET (Security Event Token) generation and validation
//! - Event transmission to registered streams
//! - Event reception and processing
//! - Stream management

pub mod error;
pub mod event;
pub mod handler;
pub mod receiver;
pub mod set;
pub mod stream;
pub mod transmitter;

pub use error::CaepError;
pub use event::{CaepEvent, CaepEventType, SubjectIdentifier};
pub use handler::EventHandler;
pub use receiver::CaepReceiver;
pub use set::SecurityEventToken;
pub use stream::{DeliveryMethod, Stream, StreamConfig, StreamStatus};
pub use transmitter::CaepTransmitter;
