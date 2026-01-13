//! Vault HTTP client with circuit breaker and logging integration.

use crate::{
    config::VaultConfig,
    error::{VaultError, VaultResult},
    provider::{DatabaseCredentialProvider, DatabaseCredentials, SecretMetadata, SecretProvider},
    secrets::{AuthResponse, DatabaseCredsResponse, KvResponse},
};
use reqwest::Client;
use rust_common::{CircuitBreaker, CircuitBreakerConfig};
use secrecy::SecretString;
use serde::de::DeserializeOwned;
use std::{sync::Arc, time::Duration};
use tokio::sync::RwLock;
use tracing::{debug, error, info, instrument, warn};

/// Vault client with automatic token renewal and circuit breaker.
pub struct VaultClient {
    config: VaultConfig,
    http: Client,
    token: Arc<RwLock<Option<String>>>,
    token_expiry: Arc<RwLock<Option<std::time::Instant>>>,
    circuit_breaker: CircuitBreaker,
}

impl VaultClient {
    /// Create a new Vault client.
    pub fn new(config: VaultConfig) -> VaultResult<Self> {
        let http = Client::builder()
            .timeout(config.timeout)
            .danger_accept_invalid_certs(true)
            .build()
            .map_err(VaultError::Http)?;

        let cb_config = CircuitBreakerConfig {
            failure_threshold: config.circuit_breaker_threshold,
            success_threshold: 2,
            timeout: config.circuit_breaker_timeout,
            half_open_max_requests: 3,
        };

        Ok(Self {
            config,
            http,
            token: Arc::new(RwLock::new(None)),
            token_expiry: Arc::new(RwLock::new(None)),
            circuit_breaker: CircuitBreaker::new(cb_config),
        })
    }

    /// Authenticate with Kubernetes auth method.
    #[instrument(skip(self), fields(role = %self.config.role))]
    pub async fn authenticate(&self) -> VaultResult<()> {
        let jwt = tokio::fs::read_to_string(&self.config.token_path)
            .await
            .map_err(|e| VaultError::auth_failed(e.to_string()))?;

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
            .map_err(|e| VaultError::unavailable(e.to_string()))?;

        if !response.status().is_success() {
            let status = response.status();
            let text = response.text().await.unwrap_or_default();
            return Err(VaultError::auth_failed(format!("Status {status}: {text}")));
        }

        let auth_response: AuthResponse = response.json().await?;
        let ttl = Duration::from_secs(auth_response.auth.lease_duration);
        let expiry = std::time::Instant::now() + ttl;

        *self.token.write().await = Some(auth_response.auth.client_token);
        *self.token_expiry.write().await = Some(expiry);

        info!(ttl_secs = ttl.as_secs(), "Authenticated with Vault");
        Ok(())
    }

    async fn get_token(&self) -> VaultResult<String> {
        let needs_auth = {
            let token = self.token.read().await;
            let expiry = self.token_expiry.read().await;

            match (&*token, &*expiry) {
                (Some(_), Some(exp)) => {
                    let remaining = exp.saturating_duration_since(std::time::Instant::now());
                    remaining.as_secs_f64() < self.config.grace_period.as_secs_f64()
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
            .ok_or_else(|| VaultError::auth_failed("No token available"))
    }

    async fn request<T: DeserializeOwned>(
        &self,
        method: reqwest::Method,
        path: &str,
        body: Option<serde_json::Value>,
    ) -> VaultResult<T> {
        if !self.circuit_breaker.allow_request().await {
            warn!(path, "Circuit breaker open for Vault");
            return Err(VaultError::CircuitBreakerOpen);
        }

        let result = self.do_request(method, path, body).await;

        match &result {
            Ok(_) => self.circuit_breaker.record_success().await,
            Err(e) if e.is_retryable() => self.circuit_breaker.record_failure().await,
            Err(_) => {}
        }

        result
    }

    async fn do_request<T: DeserializeOwned>(
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
            .map_err(|e| VaultError::unavailable(e.to_string()))?;

        let status = response.status();
        match status.as_u16() {
            404 => return Err(VaultError::not_found(path)),
            403 => return Err(VaultError::PermissionDenied(path.to_string())),
            429 => return Err(VaultError::RateLimited),
            s if s >= 500 => {
                let text = response.text().await.unwrap_or_default();
                return Err(VaultError::unavailable(format!("Status {status}: {text}")));
            }
            _ if !status.is_success() => {
                let text = response.text().await.unwrap_or_default();
                return Err(VaultError::unavailable(format!("Status {status}: {text}")));
            }
            _ => {}
        }

        response.json().await.map_err(VaultError::from)
    }
}

impl SecretProvider for VaultClient {
    type Error = VaultError;

    #[instrument(skip(self), fields(path))]
    async fn get_secret<T>(&self, path: &str) -> VaultResult<(T, SecretMetadata)>
    where
        T: DeserializeOwned + Send,
    {
        debug!(path, "Getting secret");

        let response: KvResponse<T> = self
            .request(reqwest::Method::GET, &format!("secret/data/{path}"), None)
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

    async fn get_secret_version<T>(&self, path: &str, version: u32) -> VaultResult<(T, SecretMetadata)>
    where
        T: DeserializeOwned + Send,
    {
        let response: KvResponse<T> = self
            .request(
                reqwest::Method::GET,
                &format!("secret/data/{path}?version={version}"),
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

    async fn revoke_lease(&self, lease_id: &str) -> VaultResult<()> {
        let body = serde_json::json!({ "lease_id": lease_id });
        self.request::<serde_json::Value>(reqwest::Method::PUT, "sys/leases/revoke", Some(body))
            .await?;
        Ok(())
    }
}

impl DatabaseCredentialProvider for VaultClient {
    type Error = VaultError;

    #[instrument(skip(self), fields(role))]
    async fn get_credentials(&self, role: &str) -> VaultResult<DatabaseCredentials> {
        debug!(role, "Getting database credentials");

        let response: DatabaseCredsResponse = self
            .request(
                reqwest::Method::GET,
                &format!("database/auth-platform/creds/{role}"),
                None,
            )
            .await?;

        Ok(DatabaseCredentials {
            username: response.data.username,
            password: SecretString::from(response.data.password),
            lease_id: response.lease_id,
            ttl: Duration::from_secs(response.lease_duration),
            renewable: response.renewable,
        })
    }
}
