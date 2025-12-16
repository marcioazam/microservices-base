//! Configuration Builder Unit Tests
//!
//! Tests for configuration building and validation.

#[derive(Debug, Clone)]
struct Config {
    host: String,
    port: u16,
    jwks_url: String,
    token_service_url: String,
    jwks_cache_ttl_seconds: u64,
    circuit_breaker_failure_threshold: u32,
    circuit_breaker_timeout_seconds: u64,
}

struct ConfigBuilder {
    host: Option<String>,
    port: Option<u16>,
    jwks_url: Option<String>,
    token_service_url: String,
    jwks_cache_ttl_seconds: u64,
    circuit_breaker_failure_threshold: u32,
    circuit_breaker_timeout_seconds: u64,
}

impl ConfigBuilder {
    fn new() -> Self {
        Self {
            host: None,
            port: None,
            jwks_url: None,
            token_service_url: "http://localhost:50051".to_string(),
            jwks_cache_ttl_seconds: 3600,
            circuit_breaker_failure_threshold: 5,
            circuit_breaker_timeout_seconds: 30,
        }
    }

    fn host(mut self, host: &str) -> Self {
        self.host = Some(host.to_string());
        self
    }

    fn port(mut self, port: u16) -> Self {
        self.port = Some(port);
        self
    }

    fn jwks_url(mut self, url: &str) -> Self {
        self.jwks_url = Some(url.to_string());
        self
    }

    fn token_service_url(mut self, url: &str) -> Self {
        self.token_service_url = url.to_string();
        self
    }

    fn jwks_cache_ttl(mut self, ttl: u64) -> Self {
        self.jwks_cache_ttl_seconds = ttl;
        self
    }

    fn circuit_breaker_threshold(mut self, threshold: u32) -> Self {
        self.circuit_breaker_failure_threshold = threshold;
        self
    }

    fn build(self) -> Result<Config, &'static str> {
        Ok(Config {
            host: self.host.ok_or("host is required")?,
            port: self.port.ok_or("port is required")?,
            jwks_url: self.jwks_url.ok_or("jwks_url is required")?,
            token_service_url: self.token_service_url,
            jwks_cache_ttl_seconds: self.jwks_cache_ttl_seconds,
            circuit_breaker_failure_threshold: self.circuit_breaker_failure_threshold,
            circuit_breaker_timeout_seconds: self.circuit_breaker_timeout_seconds,
        })
    }
}

#[test]
fn test_builder_all_required_fields() {
    let config = ConfigBuilder::new()
        .host("127.0.0.1")
        .port(8080)
        .jwks_url("http://localhost/jwks")
        .build()
        .unwrap();

    assert_eq!(config.host, "127.0.0.1");
    assert_eq!(config.port, 8080);
    assert_eq!(config.jwks_url, "http://localhost/jwks");
}

#[test]
fn test_builder_missing_host() {
    let result = ConfigBuilder::new()
        .port(8080)
        .jwks_url("http://localhost/jwks")
        .build();

    assert!(result.is_err());
}

#[test]
fn test_builder_missing_port() {
    let result = ConfigBuilder::new()
        .host("127.0.0.1")
        .jwks_url("http://localhost/jwks")
        .build();

    assert!(result.is_err());
}

#[test]
fn test_builder_default_values() {
    let config = ConfigBuilder::new()
        .host("localhost")
        .port(8080)
        .jwks_url("http://jwks")
        .build()
        .unwrap();

    assert_eq!(config.jwks_cache_ttl_seconds, 3600);
    assert_eq!(config.circuit_breaker_failure_threshold, 5);
    assert_eq!(config.circuit_breaker_timeout_seconds, 30);
}

#[test]
fn test_builder_custom_optional_values() {
    let config = ConfigBuilder::new()
        .host("0.0.0.0")
        .port(50052)
        .jwks_url("http://auth/jwks")
        .token_service_url("http://token:50051")
        .jwks_cache_ttl(7200)
        .circuit_breaker_threshold(10)
        .build()
        .unwrap();

    assert_eq!(config.token_service_url, "http://token:50051");
    assert_eq!(config.jwks_cache_ttl_seconds, 7200);
    assert_eq!(config.circuit_breaker_failure_threshold, 10);
}

#[test]
fn test_builder_chain_order_independent() {
    let config1 = ConfigBuilder::new()
        .host("localhost")
        .port(8080)
        .jwks_url("http://jwks")
        .build()
        .unwrap();

    let config2 = ConfigBuilder::new()
        .jwks_url("http://jwks")
        .host("localhost")
        .port(8080)
        .build()
        .unwrap();

    assert_eq!(config1.host, config2.host);
    assert_eq!(config1.port, config2.port);
}
