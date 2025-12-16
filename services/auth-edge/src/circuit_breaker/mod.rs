//! Generic Circuit Breaker with Tower Service integration
//!
//! Implements the circuit breaker pattern as a Tower Service for composable middleware.
//! Supports const generics for compile-time configuration validation.

mod state;

use std::sync::Arc;
use std::task::{Context, Poll};
use std::time::Duration;

use futures::future::BoxFuture;
use tokio::sync::RwLock;
use tower::{Layer, Service};
use tracing::info;

use crate::error::AuthEdgeError;

pub use state::{CircuitBreakerState, CircuitState};

/// Configuration for circuit breaker with const generics for compile-time validation
#[derive(Debug, Clone)]
pub struct CircuitBreakerConfig<
    const FAILURE_THRESHOLD: u32 = 5,
    const SUCCESS_THRESHOLD: u32 = 3,
    const TIMEOUT_SECS: u64 = 30,
> {
    /// Name for metrics and logging
    pub name: String,
}


impl<const F: u32, const S: u32, const T: u64> CircuitBreakerConfig<F, S, T> {
    /// Creates a new circuit breaker configuration
    pub fn new(name: impl Into<String>) -> Self {
        Self { name: name.into() }
    }

    /// Returns the failure threshold
    pub const fn failure_threshold(&self) -> u32 {
        F
    }

    /// Returns the success threshold
    pub const fn success_threshold(&self) -> u32 {
        S
    }

    /// Returns the timeout duration
    pub const fn timeout(&self) -> Duration {
        Duration::from_secs(T)
    }
}

/// Generic Circuit Breaker implementing Tower Service trait
pub struct CircuitBreaker<S, const FAILURE_THRESHOLD: u32 = 5, const SUCCESS_THRESHOLD: u32 = 3, const TIMEOUT_SECS: u64 = 30> {
    inner: S,
    state: Arc<RwLock<CircuitBreakerState>>,
    config: CircuitBreakerConfig<FAILURE_THRESHOLD, SUCCESS_THRESHOLD, TIMEOUT_SECS>,
}

impl<S, const F: u32, const S_T: u32, const T: u64> CircuitBreaker<S, F, S_T, T> {
    /// Creates a new circuit breaker wrapping the given service
    pub fn new(inner: S, config: CircuitBreakerConfig<F, S_T, T>) -> Self {
        Self {
            inner,
            state: Arc::new(RwLock::new(CircuitBreakerState::new())),
            config,
        }
    }

    /// Gets the current circuit state
    pub async fn get_state(&self) -> CircuitState {
        self.state.read().await.state
    }
}

impl<S: Clone, const F: u32, const S_T: u32, const T: u64> Clone for CircuitBreaker<S, F, S_T, T> {
    fn clone(&self) -> Self {
        Self {
            inner: self.inner.clone(),
            state: self.state.clone(),
            config: self.config.clone(),
        }
    }
}


impl<S, Req, const F: u32, const S_T: u32, const T: u64> Service<Req> for CircuitBreaker<S, F, S_T, T>
where
    S: Service<Req> + Clone + Send + 'static,
    S::Response: Send + 'static,
    S::Error: Into<AuthEdgeError> + Send + 'static,
    S::Future: Send + 'static,
    Req: Send + 'static,
{
    type Response = S::Response;
    type Error = AuthEdgeError;
    type Future = BoxFuture<'static, Result<Self::Response, Self::Error>>;

    fn poll_ready(&mut self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        self.inner.poll_ready(cx).map_err(Into::into)
    }

    fn call(&mut self, req: Req) -> Self::Future {
        let state = self.state.clone();
        let config_name = self.config.name.clone();
        let mut inner = self.inner.clone();
        let timeout = Duration::from_secs(T);

        Box::pin(async move {
            // Check if circuit allows requests
            let is_available = {
                let mut s = state.write().await;
                s.is_available(timeout, &config_name)
            };

            if !is_available {
                return Err(AuthEdgeError::CircuitOpen {
                    service: config_name,
                    retry_after: timeout,
                });
            }

            // Execute the request
            match inner.call(req).await {
                Ok(response) => {
                    let mut s = state.write().await;
                    s.record_success(S_T, &config_name);
                    Ok(response)
                }
                Err(err) => {
                    let mut s = state.write().await;
                    s.record_failure(F, &config_name);
                    Err(err.into())
                }
            }
        })
    }
}

/// Tower Layer for CircuitBreaker
pub struct CircuitBreakerLayer<const F: u32 = 5, const S: u32 = 3, const T: u64 = 30> {
    config: CircuitBreakerConfig<F, S, T>,
}

impl<const F: u32, const S: u32, const T: u64> CircuitBreakerLayer<F, S, T> {
    /// Creates a new circuit breaker layer
    pub fn new(name: impl Into<String>) -> Self {
        Self {
            config: CircuitBreakerConfig::new(name),
        }
    }
}

impl<Svc, const F: u32, const S: u32, const T: u64> Layer<Svc> for CircuitBreakerLayer<F, S, T>
where
    Svc: Clone,
{
    type Service = CircuitBreaker<Svc, F, S, T>;

    fn layer(&self, inner: Svc) -> Self::Service {
        CircuitBreaker::new(inner, self.config.clone())
    }
}


// ============================================================================
// Standalone Circuit Breaker (for non-Tower use cases)
// ============================================================================

/// Standalone circuit breaker for use outside Tower middleware stack.
/// Provides async-safe circuit breaker functionality with shared state.
#[derive(Clone)]
pub struct StandaloneCircuitBreaker {
    name: String,
    state: Arc<RwLock<CircuitBreakerState>>,
    failure_threshold: u32,
    success_threshold: u32,
    timeout: Duration,
}

impl StandaloneCircuitBreaker {
    /// Creates a new standalone circuit breaker
    pub fn new(
        name: impl Into<String>,
        failure_threshold: u32,
        success_threshold: u32,
        timeout: Duration,
    ) -> Self {
        Self {
            name: name.into(),
            state: Arc::new(RwLock::new(CircuitBreakerState::new())),
            failure_threshold,
            success_threshold,
            timeout,
        }
    }

    /// Creates from config values (convenience constructor)
    pub fn from_config(name: impl Into<String>, failure_threshold: u32, timeout_secs: u64) -> Self {
        Self::new(name, failure_threshold, 3, Duration::from_secs(timeout_secs))
    }

    /// Checks if the circuit allows requests
    pub async fn is_available(&self) -> bool {
        let mut state = self.state.write().await;
        state.is_available(self.timeout, &self.name)
    }

    /// Records a successful operation
    pub async fn record_success(&self) {
        let mut state = self.state.write().await;
        state.record_success(self.success_threshold, &self.name);
    }

    /// Records a failed operation
    pub async fn record_failure(&self) {
        let mut state = self.state.write().await;
        state.record_failure(self.failure_threshold, &self.name);
    }

    /// Gets the current circuit state
    pub async fn get_state(&self) -> CircuitState {
        self.state.read().await.state
    }

    /// Returns the circuit breaker name
    pub fn name(&self) -> &str {
        &self.name
    }

    /// Executes an async operation with circuit breaker protection
    pub async fn call<F, T, E>(&self, f: F) -> Result<T, CircuitBreakerError<E>>
    where
        F: std::future::Future<Output = Result<T, E>>,
    {
        if !self.is_available().await {
            return Err(CircuitBreakerError::Open {
                service: self.name.clone(),
                retry_after: self.timeout,
            });
        }

        match f.await {
            Ok(result) => {
                self.record_success().await;
                Ok(result)
            }
            Err(err) => {
                self.record_failure().await;
                Err(CircuitBreakerError::ServiceError(err))
            }
        }
    }
}

/// Error type for standalone circuit breaker operations
#[derive(Debug)]
pub enum CircuitBreakerError<E> {
    /// Circuit is open, requests are being rejected
    Open {
        service: String,
        retry_after: Duration,
    },
    /// The underlying service returned an error
    ServiceError(E),
}

impl<E: std::fmt::Display> std::fmt::Display for CircuitBreakerError<E> {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Open { service, retry_after } => {
                write!(f, "Circuit open for {}, retry after {:?}", service, retry_after)
            }
             Self::ServiceError(e) => write!(f, "Service error: {}", e),
        }
    }
}

impl<E: std::error::Error + 'static> std::error::Error for CircuitBreakerError<E> {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::ServiceError(e) => Some(e),
            _ => None,
        }
    }
}


