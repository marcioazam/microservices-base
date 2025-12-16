//! Vault HTTP client implementation
//! Requirements: 1.1, 1.2, 1.3, 1.4, 1.5

use crate::{
    config::VaultConfig,
    error::{VaultError, VaultResult},
    provider::{DatabaseCredentialProvider, DatabaseCredentials, SecretMetadata, SecretProvider},
    secrets::{AuthResponse, DatabaseCredsResponse, KvResponse},
};
use async_trait::async_trait;
use reqwest::Client;
use secrecy::SecretString;
use serde::de::DeserializeOwned;
use std::{sync::Arc, time::Duration};
use tokio::sync::RwLock;
use tracing::{debug, error, info, warn};

/// Vault client with automatic token renewal
pub struct VaultClient {
    config: VaultConfig,
    http: Client,
    token: Arc<RwLock<Option<String>>>,
    token_expiry: Arc<RwLock<Option<std::time::Instant>>>,
}

impl VaultClient {
    /// Create a new Vault client
    pub fn new(config: VaultConfig) -> VaultResult<Self> {
        let http = Client::builder()
            .timeout(config.timeout)
            .danger_accept_invalid_certs(true) // For dev; use proper certs in prod
            .build()
            .map_err(VaultError::Http)?;

        Ok(Self {
            config,
            http,
            token: Arc::new(RwLock::new(None)),
            token_expiry: Arc::new(RwLock::new(None)),
        })
    }

    /// Authenticate with Kubernetes auth method
    /// Requirements: 1.1 - Kubernetes authentication
    pub async fn authenticate(&self) -> VaultResult<()> {
        let jwt = tokio::fs::read_to_string(&self.config.token_path)
            .await
            .map_err(|e| VaultError::AuthenticationFailed(e.to_string()))?;

        let url = format!("{}/v1/auth/kubernetes/login", self.config.addr);
        let body = serde_json::json!({
            "role": self.config.role,
            "jwt": jwt.trim()
        });

        let response = self
            .http
            .post(&url)
            .json(&body)
            .send()
            .await
            .map_err(|e| VaultError::Unavailable(e.to_string()))?;

        if !response.status().is_success() {
            let status = response.status();
            let text = response.text().await.unwrap_or_default();
            return Err(VaultError::AuthenticationFailed(format!(
                "Status {}: {}",
                status, text
            )));
        }

        let auth_response: AuthResponse = response.json().await?;
        let ttl = Duration::from_secs(auth_response.auth.lease_duration);
        let expiry = std::time::Instant::now() + ttl;

        *self.token.write().await = Some(auth_response.auth.client_token);
        *self.token_expiry.write().await = Some(expiry);

        info!("Authenticated with Vault, token valid for {:?}", ttl);
        Ok(())
    }

    /// Get current token, re-authenticating if needed
    async fn get_token(&self) -> VaultResult<String> {
        // Check if token needs renewal
        let needs_auth = {
            let token = self.token.read().await;
            let expiry = self.token_expiry.read().await;
            
            match (&*token, &*expiry) {
                (Some(_), Some(exp)) => {
                    let remaining = exp.saturating_duration_since(std::time::Instant::now());
                    let threshold = Duration::from_secs_f64(
                        exp.duration_since(std::time::Instant::now() - remaining).as_secs_f64()
                            * self.config.renewal_threshold,
                    );
                    remaining < threshold
                }
                _ => true,
            }
        };

        if needs_auth {
            self.authenticate().await?;
        }

        self.token
            .read()
            .await
            .clone()
            .ok_or_else(|| VaultError::AuthenticationFailed("No token available".to_string()))
    }

    /// Make authenticated request to Vault
    async fn request<T: DeserializeOwned>(
        &self,
        method: reqwest::Method,
        path: &str,
        body: Option<serde_json::Value>,
    ) -> VaultResult<T> {
        let token = self.get_token().await?;
        let url = format!("{}/v1/{}", self.config.addr, path);

        let mut request = self.http.request(method, &url).header("X-Vault-Token", token);

        if let Some(b) = body {
            request = request.json(&b);
        }

        let response = request
            .send()
            .await
            .map_err(|e| VaultError::Unavailable(e.to_string()))?;

        let status = response.status();
        if status == reqwest::StatusCode::NOT_FOUND {
            return Err(VaultError::SecretNotFound(path.to_string()));
        }
        if status == reqwest::StatusCode::FORBIDDEN {
            return Err(VaultError::PermissionDenied(path.to_string()));
        }
        if status == reqwest::StatusCode::TOO_MANY_REQUESTS {
            return Err(VaultError::RateLimited);
        }
        if !status.is_success() {
            let text = response.text().await.unwrap_or_default();
            return Err(VaultError::Unavailable(format!("Status {}: {}", status, text)));
        }

        response.json().await.map_err(VaultError::from)
    }
}

#[async_trait]
impl SecretProvider for VaultClient {
    type Error = VaultError;

    /// Get a secret from KV v2 engine
    /// Requirements: 1.1, 1.4 (access is logged by Vault)
    async fn get_secret<T>(&self, path: &str) -> VaultResult<(T, SecretMetadata)>
    where
        T: DeserializeOwned + Send,
    {
        debug!("Getting secret from path: {}", path);
        
        let response: KvResponse<T> = self
            .request(reqwest::Method::GET, &format!("secret/data/{}", path), None)
            .await?;

        let metadata = SecretMetadata {
            lease_id: if response.lease_id.is_empty() {
                None
            } else {
                Some(response.lease_id)
            },
            ttl: Duration::from_secs(response.lease_duration),
            renewable: response.renewable,
            version: Some(response.data.metadata.version),
        };

        Ok((response.data.data, metadata))
    }

    /// Get a specific version of a secret
    async fn get_secret_version<T>(
        &self,
        path: &str,
        version: u32,
    ) -> VaultResult<(T, SecretMetadata)>
    where
        T: DeserializeOwned + Send,
    {
        let response: KvResponse<T> = self
            .request(
                reqwest::Method::GET,
                &format!("secret/data/{}?version={}", path, version),
                None,
            )
            .await?;

        let metadata = SecretMetadata {
            lease_id: if response.lease_id.is_empty() {
                None
            } else {
                Some(response.lease_id)
            },
            ttl: Duration::from_secs(response.lease_duration),
            renewable: response.renewable,
            version: Some(response.data.metadata.version),
        };

        Ok((response.data.data, metadata))
    }

    /// Renew a lease
    /// Requirements: 1.3 - Automatic renewal
    async fn renew_lease(&self, lease_id: &str, increment: Duration) -> VaultResult<Duration> {
        #[derive(serde::Deserialize)]
        struct RenewResponse {
            lease_duration: u64,
        }

        let body = serde_json::json!({
            "lease_id": lease_id,
            "increment": increment.as_secs()
        });

        let response: RenewResponse = self
            .request(reqwest::Method::PUT, "sys/leases/renew", Some(body))
            .await
            .map_err(|e| VaultError::LeaseRenewalFailed(e.to_string()))?;

        Ok(Duration::from_secs(response.lease_duration))
    }

    /// Revoke a lease
    async fn revoke_lease(&self, lease_id: &str) -> VaultResult<()> {
        let body = serde_json::json!({
            "lease_id": lease_id
        });

        self.request::<serde_json::Value>(reqwest::Method::PUT, "sys/leases/revoke", Some(body))
            .await?;

        Ok(())
    }
}

#[async_trait]
impl DatabaseCredentialProvider for VaultClient {
    type Error = VaultError;

    /// Get dynamic database credentials
    /// Requirements: 1.2 - Dynamic credentials with 1h TTL
    async fn get_credentials(&self, role: &str) -> VaultResult<DatabaseCredentials> {
        debug!("Getting database credentials for role: {}", role);

        let response: DatabaseCredsResponse = self
            .request(
                reqwest::Method::GET,
                &format!("database/auth-platform/creds/{}", role),
                None,
            )
            .await?;

        Ok(DatabaseCredentials {
            username: response.data.username,
            password: SecretString::new(response.data.password),
            lease_id: response.lease_id,
            ttl: Duration::from_secs(response.lease_duration),
            renewable: response.renewable,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_config_defaults() {
        let config = VaultConfig::default();
        assert_eq!(config.grace_period, Duration::from_secs(300));
        assert_eq!(config.renewal_threshold, 0.2);
    }

    #[test]
    fn test_database_credentials_should_renew() {
        let creds = DatabaseCredentials {
            username: "test".to_string(),
            password: SecretString::new("pass".to_string()),
            lease_id: "lease-123".to_string(),
            ttl: Duration::from_secs(3600), // 1 hour
            renewable: true,
        };

        // At 70% elapsed, should not renew
        assert!(!creds.should_renew(Duration::from_secs(2520)));
        
        // At 85% elapsed, should renew
        assert!(creds.should_renew(Duration::from_secs(3060)));
    }
}
