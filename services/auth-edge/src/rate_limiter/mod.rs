//! Adaptive Rate Limiter
//!
//! Implements rate limiting with HTTP 429 responses and adaptive adjustment
//! based on system load and client trust level.

use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;

/// Rate limit decision
#[derive(Debug, Clone)]
pub enum RateLimitDecision {
    /// Request allowed
    Allowed,
    /// Request denied with retry-after duration
    Denied { retry_after: Duration },
}

/// Client trust level for adaptive rate limiting
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum TrustLevel {
    /// Unknown or new client
    Unknown,
    /// Client with suspicious behavior
    Suspicious,
    /// Normal client
    Normal,
    /// Trusted client with good history
    Trusted,
}

/// Rate limit configuration
#[derive(Debug, Clone)]
pub struct RateLimitConfig {
    /// Base requests per window
    pub base_limit: u32,
    /// Window duration
    pub window: Duration,
    /// System load threshold for reduction (0.0-1.0)
    pub load_threshold: f64,
    /// Reduction factor when load exceeded
    pub load_reduction_factor: f64,
    /// Trust multiplier for trusted clients
    pub trust_multiplier: f64,
    /// Suspicious client reduction factor
    pub suspicious_reduction_factor: f64,
}

impl Default for RateLimitConfig {
    fn default() -> Self {
        RateLimitConfig {
            base_limit: 100,
            window: Duration::from_secs(60),
            load_threshold: 0.8,
            load_reduction_factor: 0.5,
            trust_multiplier: 2.0,
            suspicious_reduction_factor: 0.25,
        }
    }
}

/// Client rate limit state
#[derive(Debug, Clone)]
struct ClientState {
    request_count: u32,
    window_start: Instant,
    trust_level: TrustLevel,
    last_request: Instant,
}

/// Adaptive Rate Limiter
pub struct AdaptiveRateLimiter {
    config: RateLimitConfig,
    clients: Arc<RwLock<HashMap<String, ClientState>>>,
    system_load: Arc<RwLock<f64>>,
}

impl AdaptiveRateLimiter {
    pub fn new(config: RateLimitConfig) -> Self {
        AdaptiveRateLimiter {
            config,
            clients: Arc::new(RwLock::new(HashMap::new())),
            system_load: Arc::new(RwLock::new(0.0)),
        }
    }

    /// Checks if a request should be allowed
    pub async fn check(&self, client_id: &str) -> RateLimitDecision {
        let mut clients = self.clients.write().await;
        let now = Instant::now();
        
        let state = clients.entry(client_id.to_string()).or_insert(ClientState {
            request_count: 0,
            window_start: now,
            trust_level: TrustLevel::Unknown,
            last_request: now,
        });

        // Reset window if expired
        if now.duration_since(state.window_start) >= self.config.window {
            state.request_count = 0;
            state.window_start = now;
        }

        // Calculate effective limit
        let effective_limit = self.calculate_effective_limit(state.trust_level).await;

        // Check if limit exceeded
        if state.request_count >= effective_limit {
            let retry_after = self.config.window
                .checked_sub(now.duration_since(state.window_start))
                .unwrap_or(Duration::from_secs(1));
            
            return RateLimitDecision::Denied { retry_after };
        }

        // Allow request
        state.request_count += 1;
        state.last_request = now;
        
        RateLimitDecision::Allowed
    }

    /// Records request outcome for trust level adjustment
    pub async fn record_outcome(&self, client_id: &str, success: bool) {
        let mut clients = self.clients.write().await;
        
        if let Some(state) = clients.get_mut(client_id) {
            // Adjust trust level based on behavior
            if success {
                state.trust_level = match state.trust_level {
                    TrustLevel::Unknown => TrustLevel::Normal,
                    TrustLevel::Suspicious => TrustLevel::Unknown,
                    TrustLevel::Normal => TrustLevel::Trusted,
                    TrustLevel::Trusted => TrustLevel::Trusted,
                };
            } else {
                state.trust_level = match state.trust_level {
                    TrustLevel::Trusted => TrustLevel::Normal,
                    TrustLevel::Normal => TrustLevel::Unknown,
                    TrustLevel::Unknown => TrustLevel::Suspicious,
                    TrustLevel::Suspicious => TrustLevel::Suspicious,
                };
            }
        }
    }

    /// Updates system load metric
    pub async fn update_system_load(&self, load: f64) {
        let mut system_load = self.system_load.write().await;
        *system_load = load.clamp(0.0, 1.0);
    }

    /// Sets trust level for a client
    pub async fn set_trust_level(&self, client_id: &str, level: TrustLevel) {
        let mut clients = self.clients.write().await;
        
        if let Some(state) = clients.get_mut(client_id) {
            state.trust_level = level;
        }
    }

    /// Calculates effective limit based on trust and load
    async fn calculate_effective_limit(&self, trust_level: TrustLevel) -> u32 {
        let base = self.config.base_limit as f64;
        let load = *self.system_load.read().await;

        // Apply load reduction if threshold exceeded
        let load_adjusted = if load > self.config.load_threshold {
            base * self.config.load_reduction_factor
        } else {
            base
        };

        // Apply trust level adjustment
        let trust_adjusted = match trust_level {
            TrustLevel::Trusted => load_adjusted * self.config.trust_multiplier,
            TrustLevel::Normal => load_adjusted,
            TrustLevel::Unknown => load_adjusted * 0.75,
            TrustLevel::Suspicious => load_adjusted * self.config.suspicious_reduction_factor,
        };

        trust_adjusted.max(1.0) as u32
    }

    /// Gets current rate limit info for a client
    pub async fn get_limit_info(&self, client_id: &str) -> RateLimitInfo {
        let clients = self.clients.read().await;
        let load = *self.system_load.read().await;

        let (remaining, reset_at, trust_level) = if let Some(state) = clients.get(client_id) {
            let effective_limit = self.calculate_effective_limit(state.trust_level).await;
            let remaining = effective_limit.saturating_sub(state.request_count);
            let reset_at = state.window_start + self.config.window;
            (remaining, reset_at, state.trust_level)
        } else {
            let effective_limit = self.calculate_effective_limit(TrustLevel::Unknown).await;
            (effective_limit, Instant::now() + self.config.window, TrustLevel::Unknown)
        };

        RateLimitInfo {
            remaining,
            reset_at,
            trust_level,
            system_load: load,
        }
    }
}

/// Rate limit information for headers
#[derive(Debug, Clone)]
pub struct RateLimitInfo {
    pub remaining: u32,
    pub reset_at: Instant,
    pub trust_level: TrustLevel,
    pub system_load: f64,
}
