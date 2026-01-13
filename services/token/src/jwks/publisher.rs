//! JWKS Publisher per RFC 7517.
//!
//! Publishes JSON Web Key Sets with support for key rotation,
//! retaining previous keys during transition period.

use serde::{Deserialize, Serialize};
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;

/// JSON Web Key per RFC 7517.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Jwk {
    /// Key type (RSA, EC, oct).
    pub kty: String,
    /// Key ID.
    pub kid: String,
    /// Key use (sig, enc).
    #[serde(rename = "use")]
    pub key_use: String,
    /// Algorithm.
    pub alg: String,
    /// RSA modulus.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub n: Option<String>,
    /// RSA exponent.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub e: Option<String>,
    /// EC x coordinate.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub x: Option<String>,
    /// EC y coordinate.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub y: Option<String>,
    /// EC curve.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub crv: Option<String>,
}

/// JSON Web Key Set per RFC 7517.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Jwks {
    /// Array of JWK values.
    pub keys: Vec<Jwk>,
}

impl Jwks {
    /// Create an empty JWKS.
    #[must_use]
    pub fn new() -> Self {
        Self { keys: Vec::new() }
    }

    /// Add a key to the set.
    pub fn add_key(&mut self, key: Jwk) {
        self.keys.push(key);
    }

    /// Serialize to JSON string.
    #[must_use]
    pub fn to_json(&self) -> String {
        serde_json::to_string(self).unwrap_or_default()
    }

    /// Find a key by ID.
    #[must_use]
    pub fn find_key(&self, kid: &str) -> Option<&Jwk> {
        self.keys.iter().find(|k| k.kid == kid)
    }
}

impl Default for Jwks {
    fn default() -> Self {
        Self::new()
    }
}

/// Key with rotation metadata.
struct RotatedKey {
    key: Jwk,
    rotated_at: Instant,
}

/// JWKS Publisher with key rotation support.
pub struct JwksPublisher {
    current_keys: Arc<RwLock<Jwks>>,
    previous_keys: Arc<RwLock<Vec<RotatedKey>>>,
    retention_period: Duration,
}

impl JwksPublisher {
    /// Create a new publisher with default 24h retention.
    #[must_use]
    pub fn new() -> Self {
        Self::with_retention(Duration::from_secs(86400))
    }

    /// Create a publisher with custom retention period.
    #[must_use]
    pub fn with_retention(retention_period: Duration) -> Self {
        Self {
            current_keys: Arc::new(RwLock::new(Jwks::new())),
            previous_keys: Arc::new(RwLock::new(Vec::new())),
            retention_period,
        }
    }

    /// Add a key to the current set.
    pub async fn add_key(&self, key: Jwk) {
        let mut current = self.current_keys.write().await;
        current.add_key(key);
    }

    /// Rotate keys, moving current to previous.
    pub async fn rotate_keys(&self, new_key: Jwk) {
        let current = self.current_keys.read().await.clone();

        // Move current keys to previous with timestamp
        {
            let mut previous = self.previous_keys.write().await;
            for key in current.keys {
                previous.push(RotatedKey {
                    key,
                    rotated_at: Instant::now(),
                });
            }
            // Clean up expired keys
            previous.retain(|k| k.rotated_at.elapsed() < self.retention_period);
        }

        // Set new key as current
        {
            let mut current = self.current_keys.write().await;
            *current = Jwks::new();
            current.add_key(new_key);
        }
    }

    /// Get combined JWKS (current + retained previous).
    pub async fn get_jwks(&self) -> Jwks {
        let current = self.current_keys.read().await;
        let previous = self.previous_keys.read().await;

        let mut combined = Jwks::new();

        // Add current keys
        for key in &current.keys {
            combined.add_key(key.clone());
        }

        // Add non-expired previous keys
        for rotated in previous.iter() {
            if rotated.rotated_at.elapsed() < self.retention_period {
                combined.add_key(rotated.key.clone());
            }
        }

        combined
    }

    /// Get the current primary key ID.
    pub async fn get_current_key_id(&self) -> Option<String> {
        let current = self.current_keys.read().await;
        current.keys.first().map(|k| k.kid.clone())
    }

    /// Get count of all published keys.
    pub async fn key_count(&self) -> usize {
        let current = self.current_keys.read().await;
        let previous = self.previous_keys.read().await;
        current.keys.len() + previous.len()
    }
}

impl Default for JwksPublisher {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn create_test_key(kid: &str) -> Jwk {
        Jwk {
            kty: "RSA".to_string(),
            kid: kid.to_string(),
            key_use: "sig".to_string(),
            alg: "RS256".to_string(),
            n: Some("test-n".to_string()),
            e: Some("AQAB".to_string()),
            x: None,
            y: None,
            crv: None,
        }
    }

    #[tokio::test]
    async fn test_add_key() {
        let publisher = JwksPublisher::new();
        publisher.add_key(create_test_key("key-1")).await;

        let jwks = publisher.get_jwks().await;
        assert_eq!(jwks.keys.len(), 1);
        assert_eq!(jwks.keys[0].kid, "key-1");
    }

    #[tokio::test]
    async fn test_key_rotation_preserves_previous() {
        let publisher = JwksPublisher::new();
        publisher.add_key(create_test_key("key-1")).await;
        publisher.rotate_keys(create_test_key("key-2")).await;

        let jwks = publisher.get_jwks().await;
        assert_eq!(jwks.keys.len(), 2);

        let key_ids: Vec<&str> = jwks.keys.iter().map(|k| k.kid.as_str()).collect();
        assert!(key_ids.contains(&"key-1"));
        assert!(key_ids.contains(&"key-2"));
    }

    #[tokio::test]
    async fn test_current_key_id() {
        let publisher = JwksPublisher::new();
        assert!(publisher.get_current_key_id().await.is_none());

        publisher.add_key(create_test_key("key-1")).await;
        assert_eq!(publisher.get_current_key_id().await, Some("key-1".to_string()));

        publisher.rotate_keys(create_test_key("key-2")).await;
        assert_eq!(publisher.get_current_key_id().await, Some("key-2".to_string()));
    }

    #[tokio::test]
    async fn test_find_key() {
        let jwks = Jwks {
            keys: vec![create_test_key("key-1"), create_test_key("key-2")],
        };

        assert!(jwks.find_key("key-1").is_some());
        assert!(jwks.find_key("key-2").is_some());
        assert!(jwks.find_key("key-3").is_none());
    }

    #[tokio::test]
    async fn test_jwks_to_json() {
        let mut jwks = Jwks::new();
        jwks.add_key(create_test_key("key-1"));

        let json = jwks.to_json();
        assert!(json.contains("key-1"));
        assert!(json.contains("keys"));
    }
}
