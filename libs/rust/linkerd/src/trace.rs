//! W3C Trace Context types for distributed tracing.

use serde::{Deserialize, Serialize};

/// W3C Trace Context for distributed tracing.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct TraceContext {
    /// W3C traceparent header value
    pub traceparent: String,
    /// Optional tracestate header value
    pub tracestate: Option<String>,
}

impl TraceContext {
    /// Create a new trace context.
    #[must_use]
    pub fn new(traceparent: impl Into<String>) -> Self {
        Self {
            traceparent: traceparent.into(),
            tracestate: None,
        }
    }

    /// Create with tracestate.
    #[must_use]
    pub fn with_tracestate(mut self, tracestate: impl Into<String>) -> Self {
        self.tracestate = Some(tracestate.into());
        self
    }

    /// Check if traceparent is valid W3C format.
    /// Format: version-trace_id-parent_id-flags (00-{32hex}-{16hex}-{2hex})
    #[must_use]
    pub fn is_valid(&self) -> bool {
        let parts: Vec<&str> = self.traceparent.split('-').collect();
        if parts.len() != 4 {
            return false;
        }

        parts[0].len() == 2
            && parts[1].len() == 32
            && parts[2].len() == 16
            && parts[3].len() == 2
            && parts[0].chars().all(|c| c.is_ascii_hexdigit())
            && parts[1].chars().all(|c| c.is_ascii_hexdigit())
            && parts[2].chars().all(|c| c.is_ascii_hexdigit())
            && parts[3].chars().all(|c| c.is_ascii_hexdigit())
    }

    /// Get the trace ID from traceparent.
    #[must_use]
    pub fn trace_id(&self) -> Option<&str> {
        self.traceparent.split('-').nth(1)
    }

    /// Get the parent span ID from traceparent.
    #[must_use]
    pub fn parent_id(&self) -> Option<&str> {
        self.traceparent.split('-').nth(2)
    }

    /// Get the flags from traceparent.
    #[must_use]
    pub fn flags(&self) -> Option<&str> {
        self.traceparent.split('-').nth(3)
    }

    /// Check if trace is sampled (flag bit 0 set).
    #[must_use]
    pub fn is_sampled(&self) -> bool {
        self.flags()
            .and_then(|f| u8::from_str_radix(f, 16).ok())
            .map(|f| f & 0x01 != 0)
            .unwrap_or(false)
    }

    /// Propagate context to a new span.
    #[must_use]
    pub fn propagate(&self, new_span_id: &str) -> Self {
        let parts: Vec<&str> = self.traceparent.split('-').collect();
        if parts.len() != 4 {
            return self.clone();
        }

        Self {
            traceparent: format!("{}-{}-{}-{}", parts[0], parts[1], new_span_id, parts[3]),
            tracestate: self.tracestate.clone(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_valid_traceparent() {
        let ctx = TraceContext::new("00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01");
        assert!(ctx.is_valid());
        assert_eq!(ctx.trace_id(), Some("0af7651916cd43dd8448eb211c80319c"));
        assert_eq!(ctx.parent_id(), Some("b7ad6b7169203331"));
        assert!(ctx.is_sampled());
    }

    #[test]
    fn test_invalid_traceparent() {
        let ctx = TraceContext::new("invalid");
        assert!(!ctx.is_valid());
    }

    #[test]
    fn test_propagation() {
        let ctx = TraceContext::new("00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01");
        let propagated = ctx.propagate("1234567890abcdef");

        assert!(propagated.is_valid());
        assert_eq!(propagated.trace_id(), ctx.trace_id());
        assert_eq!(propagated.parent_id(), Some("1234567890abcdef"));
    }

    #[test]
    fn test_not_sampled() {
        let ctx = TraceContext::new("00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-00");
        assert!(!ctx.is_sampled());
    }
}
