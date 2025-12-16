use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tokio::sync::RwLock;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Jwk {
    pub kty: String,
    pub kid: String,
    #[serde(rename = "use")]
    pub key_use: String,
    pub alg: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub n: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub e: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub x: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub y: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub crv: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Jwks {
    pub keys: Vec<Jwk>,
}

impl Jwks {
    pub fn new() -> Self {
        Jwks { keys: Vec::new() }
    }

    pub fn add_key(&mut self, key: Jwk) {
        self.keys.push(key);
    }

    pub fn to_json(&self) -> String {
        serde_json::to_string(self).unwrap_or_default()
    }
}

pub struct JwksPublisher {
    current_keys: Arc<RwLock<Jwks>>,
    previous_keys: Arc<RwLock<Jwks>>,
}

impl JwksPublisher {
    pub fn new() -> Self {
        JwksPublisher {
            current_keys: Arc::new(RwLock::new(Jwks::new())),
            previous_keys: Arc::new(RwLock::new(Jwks::new())),
        }
    }

    pub async fn add_key(&self, key: Jwk) {
        let mut current = self.current_keys.write().await;
        current.add_key(key);
    }

    pub async fn rotate_keys(&self, new_key: Jwk) {
        let current = self.current_keys.read().await.clone();
        
        // Move current to previous
        {
            let mut previous = self.previous_keys.write().await;
            *previous = current;
        }

        // Set new key as current
        {
            let mut current = self.current_keys.write().await;
            *current = Jwks::new();
            current.add_key(new_key);
        }
    }

    pub async fn get_jwks(&self) -> Jwks {
        let current = self.current_keys.read().await;
        let previous = self.previous_keys.read().await;

        let mut combined = Jwks::new();
        for key in &current.keys {
            combined.add_key(key.clone());
        }
        for key in &previous.keys {
            combined.add_key(key.clone());
        }

        combined
    }

    pub async fn get_current_key_id(&self) -> Option<String> {
        let current = self.current_keys.read().await;
        current.keys.first().map(|k| k.kid.clone())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_jwks_publisher() {
        let publisher = JwksPublisher::new();

        let key1 = Jwk {
            kty: "RSA".to_string(),
            kid: "key-1".to_string(),
            key_use: "sig".to_string(),
            alg: "RS256".to_string(),
            n: Some("test-n".to_string()),
            e: Some("AQAB".to_string()),
            x: None,
            y: None,
            crv: None,
        };

        publisher.add_key(key1).await;

        let jwks = publisher.get_jwks().await;
        assert_eq!(jwks.keys.len(), 1);
        assert_eq!(jwks.keys[0].kid, "key-1");
    }

    #[tokio::test]
    async fn test_key_rotation() {
        let publisher = JwksPublisher::new();

        let key1 = Jwk {
            kty: "RSA".to_string(),
            kid: "key-1".to_string(),
            key_use: "sig".to_string(),
            alg: "RS256".to_string(),
            n: Some("test-n-1".to_string()),
            e: Some("AQAB".to_string()),
            x: None,
            y: None,
            crv: None,
        };

        publisher.add_key(key1).await;

        let key2 = Jwk {
            kty: "RSA".to_string(),
            kid: "key-2".to_string(),
            key_use: "sig".to_string(),
            alg: "RS256".to_string(),
            n: Some("test-n-2".to_string()),
            e: Some("AQAB".to_string()),
            x: None,
            y: None,
            crv: None,
        };

        publisher.rotate_keys(key2).await;

        let jwks = publisher.get_jwks().await;
        // Should contain both current and previous keys
        assert_eq!(jwks.keys.len(), 2);
        
        let key_ids: Vec<&str> = jwks.keys.iter().map(|k| k.kid.as_str()).collect();
        assert!(key_ids.contains(&"key-1"));
        assert!(key_ids.contains(&"key-2"));
    }
}
