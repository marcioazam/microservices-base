//! Rate Limiter Unit Tests
//!
//! Tests for rate limiting, trust levels, and window-based limiting.

use std::time::{Duration, Instant};

// ============================================================================
// Trust Level Types
// ============================================================================

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum TrustLevel {
    Unknown,
    Suspicious,
    Normal,
    Trusted,
}

// ============================================================================
// Rate Limit Configuration
// ============================================================================

struct RateLimitConfig {
    base_limit: u32,
    load_threshold: f64,
    load_reduction_factor: f64,
    trust_multiplier: f64,
    suspicious_reduction_factor: f64,
}

impl Default for RateLimitConfig {
    fn default() -> Self {
        Self {
            base_limit: 100,
            load_threshold: 0.8,
            load_reduction_factor: 0.5,
            trust_multiplier: 2.0,
            suspicious_reduction_factor: 0.25,
        }
    }
}

fn calculate_effective_limit(config: &RateLimitConfig, trust: TrustLevel, load: f64) -> u32 {
    let base = config.base_limit as f64;

    let load_adjusted = if load > config.load_threshold {
        base * config.load_reduction_factor
    } else {
        base
    };

    let trust_adjusted = match trust {
        TrustLevel::Trusted => load_adjusted * config.trust_multiplier,
        TrustLevel::Normal => load_adjusted,
        TrustLevel::Unknown => load_adjusted * 0.75,
        TrustLevel::Suspicious => load_adjusted * config.suspicious_reduction_factor,
    };

    trust_adjusted.max(1.0) as u32
}

// ============================================================================
// Trust Level Transitions
// ============================================================================

fn transition_on_success(current: TrustLevel) -> TrustLevel {
    match current {
        TrustLevel::Unknown => TrustLevel::Normal,
        TrustLevel::Suspicious => TrustLevel::Unknown,
        TrustLevel::Normal => TrustLevel::Trusted,
        TrustLevel::Trusted => TrustLevel::Trusted,
    }
}

fn transition_on_failure(current: TrustLevel) -> TrustLevel {
    match current {
        TrustLevel::Trusted => TrustLevel::Normal,
        TrustLevel::Normal => TrustLevel::Unknown,
        TrustLevel::Unknown => TrustLevel::Suspicious,
        TrustLevel::Suspicious => TrustLevel::Suspicious,
    }
}

// ============================================================================
// Window-based Rate Limiting
// ============================================================================

struct WindowState {
    count: u32,
    window_start: Instant,
    window_duration: Duration,
}

impl WindowState {
    fn new(window_secs: u64) -> Self {
        Self {
            count: 0,
            window_start: Instant::now(),
            window_duration: Duration::from_secs(window_secs),
        }
    }

    fn should_reset(&self) -> bool {
        self.window_start.elapsed() >= self.window_duration
    }

    fn reset(&mut self) {
        self.count = 0;
        self.window_start = Instant::now();
    }

    fn increment(&mut self) {
        if self.should_reset() {
            self.reset();
        }
        self.count += 1;
    }

    fn is_over_limit(&self, limit: u32) -> bool {
        self.count >= limit
    }
}

// ============================================================================
// Rate Limit Config Tests
// ============================================================================

#[test]
fn test_default_config() {
    let config = RateLimitConfig::default();
    assert_eq!(config.base_limit, 100);
    assert_eq!(config.load_threshold, 0.8);
}

#[test]
fn test_normal_trust_normal_load() {
    let config = RateLimitConfig::default();
    let limit = calculate_effective_limit(&config, TrustLevel::Normal, 0.5);
    assert_eq!(limit, 100);
}

#[test]
fn test_trusted_client_gets_higher_limit() {
    let config = RateLimitConfig::default();
    let limit = calculate_effective_limit(&config, TrustLevel::Trusted, 0.5);
    assert_eq!(limit, 200);
}

#[test]
fn test_suspicious_client_gets_lower_limit() {
    let config = RateLimitConfig::default();
    let limit = calculate_effective_limit(&config, TrustLevel::Suspicious, 0.5);
    assert_eq!(limit, 25);
}

#[test]
fn test_unknown_client_gets_reduced_limit() {
    let config = RateLimitConfig::default();
    let limit = calculate_effective_limit(&config, TrustLevel::Unknown, 0.5);
    assert_eq!(limit, 75);
}

#[test]
fn test_high_load_reduces_limit() {
    let config = RateLimitConfig::default();
    let limit = calculate_effective_limit(&config, TrustLevel::Normal, 0.9);
    assert_eq!(limit, 50);
}

#[test]
fn test_high_load_with_trusted() {
    let config = RateLimitConfig::default();
    let limit = calculate_effective_limit(&config, TrustLevel::Trusted, 0.9);
    assert_eq!(limit, 100);
}

#[test]
fn test_minimum_limit_is_one() {
    let config = RateLimitConfig {
        base_limit: 1,
        suspicious_reduction_factor: 0.01,
        ..Default::default()
    };
    let limit = calculate_effective_limit(&config, TrustLevel::Suspicious, 0.9);
    assert!(limit >= 1);
}

#[test]
fn test_load_at_threshold_boundary() {
    let config = RateLimitConfig::default();

    let limit_at = calculate_effective_limit(&config, TrustLevel::Normal, 0.8);
    assert_eq!(limit_at, 100);

    let limit_above = calculate_effective_limit(&config, TrustLevel::Normal, 0.81);
    assert_eq!(limit_above, 50);
}

// ============================================================================
// Trust Level Transition Tests
// ============================================================================

#[test]
fn test_trust_level_upgrade_on_success() {
    assert_eq!(transition_on_success(TrustLevel::Unknown), TrustLevel::Normal);
    assert_eq!(transition_on_success(TrustLevel::Normal), TrustLevel::Trusted);
    assert_eq!(transition_on_success(TrustLevel::Trusted), TrustLevel::Trusted);
}

#[test]
fn test_trust_level_recovery_from_suspicious() {
    assert_eq!(transition_on_success(TrustLevel::Suspicious), TrustLevel::Unknown);
}

#[test]
fn test_trust_level_downgrade_on_failure() {
    assert_eq!(transition_on_failure(TrustLevel::Trusted), TrustLevel::Normal);
    assert_eq!(transition_on_failure(TrustLevel::Normal), TrustLevel::Unknown);
    assert_eq!(transition_on_failure(TrustLevel::Unknown), TrustLevel::Suspicious);
}

#[test]
fn test_trust_level_stays_suspicious() {
    assert_eq!(transition_on_failure(TrustLevel::Suspicious), TrustLevel::Suspicious);
}

#[test]
fn test_trust_level_full_cycle() {
    let mut level = TrustLevel::Unknown;

    level = transition_on_success(level);
    assert_eq!(level, TrustLevel::Normal);

    level = transition_on_success(level);
    assert_eq!(level, TrustLevel::Trusted);

    level = transition_on_failure(level);
    assert_eq!(level, TrustLevel::Normal);

    level = transition_on_failure(level);
    assert_eq!(level, TrustLevel::Unknown);

    level = transition_on_failure(level);
    assert_eq!(level, TrustLevel::Suspicious);
}

// ============================================================================
// Window Rate Limit Tests
// ============================================================================

#[test]
fn test_window_initial_state() {
    let state = WindowState::new(60);
    assert_eq!(state.count, 0);
    assert!(!state.is_over_limit(100));
}

#[test]
fn test_window_increment() {
    let mut state = WindowState::new(60);
    state.increment();
    assert_eq!(state.count, 1);
}

#[test]
fn test_window_over_limit() {
    let mut state = WindowState::new(60);
    for _ in 0..10 {
        state.increment();
    }
    assert!(state.is_over_limit(10));
    assert!(!state.is_over_limit(11));
}

#[test]
fn test_window_reset() {
    let mut state = WindowState::new(60);
    state.increment();
    state.increment();
    assert_eq!(state.count, 2);

    state.reset();
    assert_eq!(state.count, 0);
}
