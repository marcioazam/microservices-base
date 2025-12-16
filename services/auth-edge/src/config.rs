//! Type-Safe Configuration with Builder Pattern
//!
//! Uses type-state pattern to ensure all required fields are set at compile time.

use serde::Deserialize;
use std::env;
use std::marker::PhantomData;

/// Marker for unset required field
pub struct Unset;
/// Marker for set required field
pub struct Set<T>(T);

/// Type-state configuration builder
/// 
/// Required fields use phantom types to track whether they've been set.
/// The build() method is only available when all required fields are Set.
pub struct ConfigBuilder<Host = Unset, Port = Unset, JwksUrl = Unset> {
    host: Host,
    port: Port,
    jwks_url: JwksUrl,
    // Optional fields with defaults
    token_service_url: String,
    session_service_url: String,
    iam_service_url: String,
    jwks_cache_ttl_seconds: u64,
    circuit_breaker_failure_threshold: u32,
    circuit_breaker_timeout_seconds: u64,
}

impl ConfigBuilder<Unset, Unset, Unset> {
    /// Creates a new configuration builder with defaults
    pub fn new() -> Self {
        ConfigBuilder {
            host: Unset,
            port: Unset,
            jwks_url: Unset,
            token_service_url: "http://localhost:50051".to_string(),
            session_service_url: "http://localhost:50053".to_string(),
            iam_service_url: "http://localhost:50054".to_string(),
            jwks_cache_ttl_seconds: 3600,
            circuit_breaker_failure_threshold: 5,
            circuit_breaker_timeout_seconds: 30,
        }
    }
}

impl Default for ConfigBuilder<Unset, Unset, Unset> {
    fn default() -> Self {
        Self::new()
    }
}

// Host setter
impl<Port, JwksUrl> ConfigBuilder<Unset, Port, JwksUrl> {
    /// Sets the host address (required)
    pub fn host(self, host: impl Into<String>) -> ConfigBuilder<Set<String>, Port, JwksUrl> {
        ConfigBuilder {
            host: Set(host.into()),
            port: self.port,
            jwks_url: self.jwks_url,
            token_service_url: self.token_service_url,
            session_service_url: self.session_service_url,
            iam_service_url: self.iam_service_url,
            jwks_cache_ttl_seconds: self.jwks_cache_ttl_seconds,
            circuit_breaker_failure_threshold: self.circuit_breaker_failure_threshold,
            circuit_breaker_timeout_seconds: self.circuit_breaker_timeout_seconds,
        }
    }
}

// Port setter
impl<Host, JwksUrl> ConfigBuilder<Host, Unset, JwksUrl> {
    /// Sets the port (required)
    pub fn port(self, port: u16) -> ConfigBuilder<Host, Set<u16>, JwksUrl> {
        ConfigBuilder {
            host: self.host,
            port: Set(port),
            jwks_url: self.jwks_url,
            token_service_url: self.token_service_url,
            session_service_url: self.session_service_url,
            iam_service_url: self.iam_service_url,
            jwks_cache_ttl_seconds: self.jwks_cache_ttl_seconds,
            circuit_breaker_failure_threshold: self.circuit_breaker_failure_threshold,
            circuit_breaker_timeout_seconds: self.circuit_breaker_timeout_seconds,
        }
    }
}

// JWKS URL setter
impl<Host, Port> ConfigBuilder<Host, Port, Unset> {
    /// Sets the JWKS URL (required)
    pub fn jwks_url(self, url: impl Into<String>) -> ConfigBuilder<Host, Port, Set<String>> {
        ConfigBuilder {
            host: self.host,
            port: self.port,
            jwks_url: Set(url.into()),
            token_service_url: self.token_service_url,
            session_service_url: self.session_service_url,
            iam_service_url: self.iam_service_url,
            jwks_cache_ttl_seconds: self.jwks_cache_ttl_seconds,
            circuit_breaker_failure_threshold: self.circuit_breaker_failure_threshold,
            circuit_breaker_timeout_seconds: self.circuit_breaker_timeout_seconds,
        }
    }
}

// Optional setters (available in any state)
impl<Host, Port, JwksUrl> ConfigBuilder<Host, Port, JwksUrl> {
    /// Sets the token service URL
    pub fn token_service_url(mut self, url: impl Into<String>) -> Self {
        self.token_service_url = url.into();
        self
    }

    /// Sets the session service URL
    pub fn session_service_url(mut self, url: impl Into<String>) -> Self {
        self.session_service_url = url.into();
        self
    }

    /// Sets the IAM service URL
    pub fn iam_service_url(mut self, url: impl Into<String>) -> Self {
        self.iam_service_url = url.into();
        self
    }

    /// Sets the JWKS cache TTL in seconds
    pub fn jwks_cache_ttl(mut self, ttl_seconds: u64) -> Self {
        self.jwks_cache_ttl_seconds = ttl_seconds;
        self
    }

    /// Sets the circuit breaker failure threshold
    pub fn circuit_breaker_threshold(mut self, threshold: u32) -> Self {
        self.circuit_breaker_failure_threshold = threshold;
        self
    }

    /// Sets the circuit breaker timeout in seconds
    pub fn circuit_breaker_timeout(mut self, timeout_seconds: u64) -> Self {
        self.circuit_breaker_timeout_seconds = timeout_seconds;
        self
    }
}

// Build method - only available when all required fields are set
impl ConfigBuilder<Set<String>, Set<u16>, Set<String>> {
    /// Builds the configuration
    /// 
    /// This method is only available when host, port, and jwks_url are all set.
    pub fn build(self) -> Config {
        Config {
            host: self.host.0,
            port: self.port.0,
            jwks_url: self.jwks_url.0,
            token_service_url: self.token_service_url,
            session_service_url: self.session_service_url,
            iam_service_url: self.iam_service_url,
            jwks_cache_ttl_seconds: self.jwks_cache_ttl_seconds,
            circuit_breaker_failure_threshold: self.circuit_breaker_failure_threshold,
            circuit_breaker_timeout_seconds: self.circuit_breaker_timeout_seconds,
        }
    }
}

/// Service configuration
#[derive(Debug, Clone, Deserialize)]
pub struct Config {
    pub host: String,
    pub port: u16,
    pub token_service_url: String,
    pub session_service_url: String,
    pub iam_service_url: String,
    pub jwks_url: String,
    pub jwks_cache_ttl_seconds: u64,
    pub circuit_breaker_failure_threshold: u32,
    pub circuit_breaker_timeout_seconds: u64,
}

impl Config {
    /// Creates a new configuration builder
    pub fn builder() -> ConfigBuilder<Unset, Unset, Unset> {
        ConfigBuilder::new()
    }

    /// Loads configuration from environment variables
    pub fn from_env() -> Result<Self, Box<dyn std::error::Error>> {
        dotenvy::dotenv().ok();

        Ok(Config {
            host: env::var("HOST").unwrap_or_else(|_| "0.0.0.0".to_string()),
            port: env::var("PORT")
                .unwrap_or_else(|_| "50052".to_string())
                .parse()?,
            token_service_url: env::var("TOKEN_SERVICE_URL")
                .unwrap_or_else(|_| "http://localhost:50051".to_string()),
            session_service_url: env::var("SESSION_SERVICE_URL")
                .unwrap_or_else(|_| "http://localhost:50053".to_string()),
            iam_service_url: env::var("IAM_SERVICE_URL")
                .unwrap_or_else(|_| "http://localhost:50054".to_string()),
            jwks_url: env::var("JWKS_URL")
                .unwrap_or_else(|_| "http://localhost:50051/.well-known/jwks.json".to_string()),
            jwks_cache_ttl_seconds: env::var("JWKS_CACHE_TTL")
                .unwrap_or_else(|_| "3600".to_string())
                .parse()?,
            circuit_breaker_failure_threshold: env::var("CB_FAILURE_THRESHOLD")
                .unwrap_or_else(|_| "5".to_string())
                .parse()?,
            circuit_breaker_timeout_seconds: env::var("CB_TIMEOUT")
                .unwrap_or_else(|_| "30".to_string())
                .parse()?,
        })
    }
}
