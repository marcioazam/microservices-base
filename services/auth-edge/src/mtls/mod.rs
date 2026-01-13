pub mod spiffe;
pub mod verifier;

// Re-export commonly used types
pub use spiffe::{SpiffeValidator, SpiffeId, OwnedSpiffeId, SpiffeError};
pub use verifier::CertificateVerifier;
