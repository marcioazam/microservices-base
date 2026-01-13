//! Mock implementations for testing.
//!
//! This module provides mock implementations of service clients for use in tests.

use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;

/// Mock logging client for testing.
#[derive(Debug, Default)]
pub struct MockLoggingClient {
    logs: Arc<RwLock<Vec<MockLogEntry>>>,
}

/// A mock log entry.
#[derive(Debug, Clone)]
pub struct MockLogEntry {
    /// Log level
    pub level: String,
    /// Log message
    pub message: String,
    /// Service ID
    pub service_id: String,
    /// Correlation ID
    pub correlation_id: Option<String>,
    /// Trace ID
    pub trace_id: Option<String>,
}

impl MockLoggingClient {
    /// Create a new mock logging client.
    #[must_use]
    pub fn new() -> Self {
        Self::default()
    }

    /// Log a message.
    pub async fn log(&self, level: &str, message: &str, service_id: &str) {
        let entry = MockLogEntry {
            level: level.to_string(),
            message: message.to_string(),
            service_id: service_id.to_string(),
            correlation_id: None,
            trace_id: None,
        };
        self.logs.write().await.push(entry);
    }

    /// Log with context.
    pub async fn log_with_context(
        &self,
        level: &str,
        message: &str,
        service_id: &str,
        correlation_id: Option<&str>,
        trace_id: Option<&str>,
    ) {
        let entry = MockLogEntry {
            level: level.to_string(),
            message: message.to_string(),
            service_id: service_id.to_string(),
            correlation_id: correlation_id.map(String::from),
            trace_id: trace_id.map(String::from),
        };
        self.logs.write().await.push(entry);
    }

    /// Get all logged entries.
    pub async fn get_logs(&self) -> Vec<MockLogEntry> {
        self.logs.read().await.clone()
    }

    /// Clear all logs.
    pub async fn clear(&self) {
        self.logs.write().await.clear();
    }

    /// Get log count.
    pub async fn count(&self) -> usize {
        self.logs.read().await.len()
    }
}

/// Mock cache client for testing.
#[derive(Debug, Default)]
pub struct MockCacheClient {
    cache: Arc<RwLock<HashMap<String, Vec<u8>>>>,
    namespace: String,
    hits: Arc<RwLock<u64>>,
    misses: Arc<RwLock<u64>>,
}

impl MockCacheClient {
    /// Create a new mock cache client.
    #[must_use]
    pub fn new(namespace: &str) -> Self {
        Self {
            cache: Arc::new(RwLock::new(HashMap::new())),
            namespace: namespace.to_string(),
            hits: Arc::new(RwLock::new(0)),
            misses: Arc::new(RwLock::new(0)),
        }
    }

    /// Get a value from the cache.
    pub async fn get(&self, key: &str) -> Option<Vec<u8>> {
        let namespaced_key = format!("{}:{}", self.namespace, key);
        let cache = self.cache.read().await;
        if let Some(value) = cache.get(&namespaced_key) {
            *self.hits.write().await += 1;
            Some(value.clone())
        } else {
            *self.misses.write().await += 1;
            None
        }
    }

    /// Set a value in the cache.
    pub async fn set(&self, key: &str, value: &[u8]) {
        let namespaced_key = format!("{}:{}", self.namespace, key);
        self.cache.write().await.insert(namespaced_key, value.to_vec());
    }

    /// Delete a value from the cache.
    pub async fn delete(&self, key: &str) {
        let namespaced_key = format!("{}:{}", self.namespace, key);
        self.cache.write().await.remove(&namespaced_key);
    }

    /// Check if a key exists.
    pub async fn exists(&self, key: &str) -> bool {
        let namespaced_key = format!("{}:{}", self.namespace, key);
        self.cache.read().await.contains_key(&namespaced_key)
    }

    /// Get cache hit count.
    pub async fn hits(&self) -> u64 {
        *self.hits.read().await
    }

    /// Get cache miss count.
    pub async fn misses(&self) -> u64 {
        *self.misses.read().await
    }

    /// Get cache size.
    pub async fn size(&self) -> usize {
        self.cache.read().await.len()
    }

    /// Clear the cache.
    pub async fn clear(&self) {
        self.cache.write().await.clear();
        *self.hits.write().await = 0;
        *self.misses.write().await = 0;
    }
}

/// Mock Vault client for testing.
#[derive(Debug, Default)]
pub struct MockVaultClient {
    secrets: Arc<RwLock<HashMap<String, serde_json::Value>>>,
}

impl MockVaultClient {
    /// Create a new mock Vault client.
    #[must_use]
    pub fn new() -> Self {
        Self::default()
    }

    /// Set a secret.
    pub async fn set_secret(&self, path: &str, value: serde_json::Value) {
        self.secrets.write().await.insert(path.to_string(), value);
    }

    /// Get a secret.
    pub async fn get_secret(&self, path: &str) -> Option<serde_json::Value> {
        self.secrets.read().await.get(path).cloned()
    }

    /// Delete a secret.
    pub async fn delete_secret(&self, path: &str) {
        self.secrets.write().await.remove(path);
    }

    /// Check if a secret exists.
    pub async fn exists(&self, path: &str) -> bool {
        self.secrets.read().await.contains_key(path)
    }

    /// Clear all secrets.
    pub async fn clear(&self) {
        self.secrets.write().await.clear();
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_mock_logging_client() {
        let client = MockLoggingClient::new();

        client.log("INFO", "test message", "test-service").await;
        client.log("ERROR", "error message", "test-service").await;

        let logs = client.get_logs().await;
        assert_eq!(logs.len(), 2);
        assert_eq!(logs[0].level, "INFO");
        assert_eq!(logs[1].level, "ERROR");
    }

    #[tokio::test]
    async fn test_mock_cache_client() {
        let client = MockCacheClient::new("test");

        client.set("key1", b"value1").await;
        let result = client.get("key1").await;
        assert_eq!(result, Some(b"value1".to_vec()));

        assert_eq!(client.hits().await, 1);
        assert_eq!(client.misses().await, 0);

        let missing = client.get("missing").await;
        assert_eq!(missing, None);
        assert_eq!(client.misses().await, 1);
    }

    #[tokio::test]
    async fn test_mock_vault_client() {
        let client = MockVaultClient::new();

        let secret = serde_json::json!({"username": "admin", "password": "secret"});
        client.set_secret("database/creds", secret.clone()).await;

        let result = client.get_secret("database/creds").await;
        assert_eq!(result, Some(secret));

        assert!(client.exists("database/creds").await);
        assert!(!client.exists("missing").await);
    }
}
