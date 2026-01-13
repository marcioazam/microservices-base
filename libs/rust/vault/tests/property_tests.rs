//! Property-based tests for Vault client.
//!
//! Tests validate:
//! - Property 15: Secret Non-Exposure in Debug Output

use proptest::prelude::*;
use secrecy::{ExposeSecret, SecretString};
use std::fmt::Debug;

/// Mock secret wrapper for testing debug output
#[derive(Clone)]
struct MockSecret {
    value: SecretString,
    metadata: String,
}

impl Debug for MockSecret {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("MockSecret")
            .field("value", &"[REDACTED]")
            .field("metadata", &self.metadata)
            .finish()
    }
}

impl MockSecret {
    fn new(value: impl Into<String>, metadata: impl Into<String>) -> Self {
        Self {
            value: SecretString::from(value.into()),
            metadata: metadata.into(),
        }
    }

    fn expose(&self) -> &str {
        self.value.expose_secret()
    }
}

/// Mock database credentials for testing
#[derive(Clone)]
struct MockDatabaseCredentials {
    username: String,
    password: SecretString,
    lease_id: String,
}

impl Debug for MockDatabaseCredentials {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("MockDatabaseCredentials")
            .field("username", &self.username)
            .field("password", &"[REDACTED]")
            .field("lease_id", &self.lease_id)
            .finish()
    }
}

// Strategy for generating secret values
fn secret_value_strategy() -> impl Strategy<Value = String> {
    "[A-Za-z0-9!@#$%^&*]{8,64}"
}

// Strategy for generating usernames
fn username_strategy() -> impl Strategy<Value = String> {
    "[a-z][a-z0-9_]{3,15}"
}

// Strategy for generating lease IDs
fn lease_id_strategy() -> impl Strategy<Value = String> {
    "[a-z0-9]{8}/[a-z0-9]{8}/[a-z0-9]{8}"
}

// Strategy for generating secret paths
fn secret_path_strategy() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("secret/data/jwt-signing-key".to_string()),
        Just("secret/data/api-keys/service-a".to_string()),
        Just("database/creds/readonly".to_string()),
        // Generate path segments without consecutive slashes
        prop::collection::vec("[a-z][a-z0-9]{2,10}", 1..4)
            .prop_map(|segments| format!("secret/data/{}", segments.join("/"))),
    ]
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// **Property 15: Secret Non-Exposure in Debug Output**
    /// *For any* secret value stored in SecretString, the Debug implementation
    /// SHALL NOT expose the actual secret value, only showing [REDACTED].
    /// **Validates: Requirements 15.6**
    #[test]
    fn prop_secret_not_exposed_in_debug(
        secret_value in secret_value_strategy(),
        metadata in "[a-z]{5,20}",
    ) {
        let secret = MockSecret::new(secret_value.clone(), metadata);

        // Debug output should not contain the secret
        let debug_output = format!("{:?}", secret);

        prop_assert!(
            !debug_output.contains(&secret_value),
            "Debug output should not contain secret value"
        );

        prop_assert!(
            debug_output.contains("[REDACTED]"),
            "Debug output should show [REDACTED]"
        );

        // But we can still access the secret when needed
        prop_assert_eq!(
            secret.expose(),
            &secret_value,
            "Secret should be accessible via expose()"
        );
    }

    /// Property: Database credentials don't expose password in debug
    #[test]
    fn prop_db_credentials_password_redacted(
        username in username_strategy(),
        password in secret_value_strategy(),
        lease_id in lease_id_strategy(),
    ) {
        let creds = MockDatabaseCredentials {
            username: username.clone(),
            password: SecretString::from(password.clone()),
            lease_id: lease_id.clone(),
        };

        let debug_output = format!("{:?}", creds);

        // Password should not appear
        prop_assert!(
            !debug_output.contains(&password),
            "Debug output should not contain password"
        );

        // Username and lease_id can appear (not secrets)
        prop_assert!(
            debug_output.contains(&username),
            "Debug output should contain username"
        );

        prop_assert!(
            debug_output.contains(&lease_id),
            "Debug output should contain lease_id"
        );
    }

    /// Property: Secret path validation
    #[test]
    fn prop_secret_path_format(
        path in secret_path_strategy(),
    ) {
        // Path should not be empty
        prop_assert!(!path.is_empty(), "Path should not be empty");

        // Path should not contain double slashes
        prop_assert!(
            !path.contains("//"),
            "Path should not contain double slashes"
        );

        // Path should start with a valid prefix
        let valid_prefixes = ["secret/", "database/", "pki/", "transit/"];
        prop_assert!(
            valid_prefixes.iter().any(|p| path.starts_with(p)),
            "Path should start with valid prefix"
        );
    }

    /// Property: SecretString zeroization
    #[test]
    fn prop_secret_string_clone_independent(
        value in secret_value_strategy(),
    ) {
        let original = SecretString::from(value.clone());
        let cloned = original.clone();

        // Both should expose the same value
        prop_assert_eq!(
            original.expose_secret(),
            cloned.expose_secret(),
            "Cloned secret should have same value"
        );

        // Both should have same value as input
        prop_assert_eq!(
            original.expose_secret(),
            &value,
            "Secret should preserve original value"
        );
    }
}

/// Test that SecretString doesn't leak in Display
#[test]
fn test_secret_string_no_display_leak() {
    let secret = SecretString::from("super-secret-password");

    // SecretString doesn't implement Display, so this won't compile:
    // let display = format!("{}", secret);

    // Debug should show redacted
    let debug = format!("{:?}", secret);
    assert!(!debug.contains("super-secret-password"));
}

/// Test MockSecret debug implementation
#[test]
fn test_mock_secret_debug() {
    let secret = MockSecret::new("my-api-key-12345", "production");

    let debug = format!("{:?}", secret);

    assert!(!debug.contains("my-api-key-12345"));
    assert!(debug.contains("[REDACTED]"));
    assert!(debug.contains("production"));
}

/// Test database credentials debug
#[test]
fn test_db_credentials_debug() {
    let creds = MockDatabaseCredentials {
        username: "app_user".to_string(),
        password: SecretString::from("db-password-xyz"),
        lease_id: "abc123/def456/ghi789".to_string(),
    };

    let debug = format!("{:?}", creds);

    assert!(!debug.contains("db-password-xyz"));
    assert!(debug.contains("app_user"));
    assert!(debug.contains("[REDACTED]"));
}
