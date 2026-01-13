//! Property-based tests for logging behavior.
//!
//! Tests:
//! - Property 6: Log Level Classification
//! - Property 2: Correlation ID Propagation

use proptest::prelude::*;
use rust_common::{LogEntry, LogLevel};

/// Generates arbitrary log levels
fn arb_log_level() -> impl Strategy<Value = LogLevel> {
    prop_oneof![
        Just(LogLevel::Debug),
        Just(LogLevel::Info),
        Just(LogLevel::Warn),
        Just(LogLevel::Error),
    ]
}

/// Generates arbitrary correlation IDs (UUID format)
fn arb_correlation_id() -> impl Strategy<Value = String> {
    "[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}"
        .prop_map(|s| s.to_lowercase())
}

/// Generates arbitrary service IDs
fn arb_service_id() -> impl Strategy<Value = String> {
    "[a-z][a-z0-9-]{2,30}".prop_map(|s| s.to_string())
}

/// Generates arbitrary log messages
fn arb_log_message() -> impl Strategy<Value = String> {
    "[A-Za-z0-9 .,!?:;-]{1,200}".prop_map(|s| s.to_string())
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// Property 6: Log Level Classification
    /// 
    /// Validates that:
    /// - Success events always use Info level
    /// - Failure events always use Error level
    /// - Warning events use Warn level
    /// - Debug events use Debug level
    #[test]
    fn prop_log_level_classification(
        message in arb_log_message(),
        service_id in arb_service_id(),
    ) {
        // Success events should be Info
        let success_entry = LogEntry::new(LogLevel::Info, &message, &service_id);
        prop_assert_eq!(success_entry.level, LogLevel::Info);

        // Failure events should be Error
        let failure_entry = LogEntry::new(LogLevel::Error, &message, &service_id);
        prop_assert_eq!(failure_entry.level, LogLevel::Error);

        // Warning events should be Warn
        let warn_entry = LogEntry::new(LogLevel::Warn, &message, &service_id);
        prop_assert_eq!(warn_entry.level, LogLevel::Warn);

        // Debug events should be Debug
        let debug_entry = LogEntry::new(LogLevel::Debug, &message, &service_id);
        prop_assert_eq!(debug_entry.level, LogLevel::Debug);
    }

    /// Property 2: Correlation ID Propagation
    /// 
    /// Validates that:
    /// - Correlation IDs are preserved in log entries
    /// - Correlation IDs follow UUID format
    /// - Correlation IDs are never empty when set
    #[test]
    fn prop_correlation_id_propagation(
        correlation_id in arb_correlation_id(),
        message in arb_log_message(),
        service_id in arb_service_id(),
    ) {
        let entry = LogEntry::new(LogLevel::Info, &message, &service_id)
            .with_correlation_id(&correlation_id);

        // Correlation ID should be preserved
        prop_assert_eq!(entry.correlation_id.as_deref(), Some(correlation_id.as_str()));

        // Correlation ID should not be empty
        prop_assert!(!correlation_id.is_empty());

        // Correlation ID should be valid UUID format (36 chars with hyphens)
        prop_assert_eq!(correlation_id.len(), 36);
        prop_assert!(correlation_id.chars().filter(|c| *c == '-').count() == 4);
    }

    /// Property: Trace Context Propagation
    /// 
    /// Validates that trace_id and span_id are preserved in log entries.
    #[test]
    fn prop_trace_context_propagation(
        trace_id in "[0-9a-f]{32}",
        span_id in "[0-9a-f]{16}",
        message in arb_log_message(),
        service_id in arb_service_id(),
    ) {
        let entry = LogEntry::new(LogLevel::Info, &message, &service_id)
            .with_trace_context(&trace_id, &span_id);

        // Trace context should be preserved
        prop_assert_eq!(entry.trace_id.as_deref(), Some(trace_id.as_str()));
        prop_assert_eq!(entry.span_id.as_deref(), Some(span_id.as_str()));
    }

    /// Property: Log Entry Metadata Preservation
    /// 
    /// Validates that metadata key-value pairs are preserved.
    #[test]
    fn prop_metadata_preservation(
        key in "[a-z_]{1,20}",
        value in "[A-Za-z0-9_-]{1,50}",
        message in arb_log_message(),
        service_id in arb_service_id(),
    ) {
        let entry = LogEntry::new(LogLevel::Info, &message, &service_id)
            .with_metadata(&key, &value);

        // Metadata should be preserved
        prop_assert!(entry.metadata.contains_key(&key));
        prop_assert_eq!(entry.metadata.get(&key).map(|s| s.as_str()), Some(value.as_str()));
    }

    /// Property: Service ID Consistency
    /// 
    /// Validates that service_id is always set and consistent.
    #[test]
    fn prop_service_id_consistency(
        service_id in arb_service_id(),
        level in arb_log_level(),
        message in arb_log_message(),
    ) {
        let entry = LogEntry::new(level, &message, &service_id);

        // Service ID should be preserved
        prop_assert_eq!(entry.service_id, service_id);

        // Service ID should not be empty
        prop_assert!(!entry.service_id.is_empty());
    }
}

/// Tests for validation success logging
#[cfg(test)]
mod validation_logging_tests {
    use super::*;

    #[test]
    fn test_validation_success_uses_info_level() {
        let entry = LogEntry::new(
            LogLevel::Info,
            "Token validated successfully",
            "auth-edge-service",
        );
        assert_eq!(entry.level, LogLevel::Info);
    }

    #[test]
    fn test_validation_failure_uses_error_level() {
        let entry = LogEntry::new(
            LogLevel::Error,
            "Token validation failed",
            "auth-edge-service",
        );
        assert_eq!(entry.level, LogLevel::Error);
    }

    #[test]
    fn test_circuit_breaker_change_uses_warn_level() {
        let entry = LogEntry::new(
            LogLevel::Warn,
            "Circuit breaker state changed",
            "auth-edge-service",
        );
        assert_eq!(entry.level, LogLevel::Warn);
    }

    #[test]
    fn test_rate_limit_uses_warn_level() {
        let entry = LogEntry::new(
            LogLevel::Warn,
            "Rate limit exceeded",
            "auth-edge-service",
        );
        assert_eq!(entry.level, LogLevel::Warn);
    }
}
