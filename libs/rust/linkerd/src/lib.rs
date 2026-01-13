//! Linkerd service mesh types and utilities.
//!
//! Provides types for mTLS connections, trace context propagation,
//! and Linkerd metrics.

#![forbid(unsafe_code)]
#![warn(missing_docs)]

pub mod metrics;
pub mod mtls;
pub mod trace;

pub use metrics::LinkerdMetrics;
pub use mtls::MtlsConnection;
pub use trace::TraceContext;
