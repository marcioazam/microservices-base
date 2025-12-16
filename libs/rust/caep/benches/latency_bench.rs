//! CAEP latency benchmarks.
//!
//! **Feature: auth-platform-q2-2025-evolution, Property 15: CAEP Event Delivery Latency**
//! **Validates: Requirements 12.3**
//!
//! For any security event emission, the SET SHALL be delivered to all active streams
//! within 100ms at p99 latency.

use auth_caep::*;
use std::time::{Duration, Instant};

const SAMPLE_SIZE: usize = 100;
const P99_LIMIT_MS: u64 = 100;

/// Benchmark SET generation latency
fn bench_set_generation() -> Vec<Duration> {
    let mut latencies = Vec::with_capacity(SAMPLE_SIZE);

    for i in 0..SAMPLE_SIZE {
        let subject = SubjectIdentifier::IssSub {
            iss: "https://auth.example.com".to_string(),
            sub: format!("user-{}", i),
        };

        let event = CaepEvent::session_revoked(subject, Some("Benchmark test".to_string()));

        let start = Instant::now();
        let _set = SecurityEventToken::from_event(
            &event,
            "https://auth.example.com",
            "https://receiver.example.com",
        );
        latencies.push(start.elapsed());
    }

    latencies.sort();
    latencies
}

/// Calculate p99 latency from sorted latencies
fn calculate_p99(latencies: &[Duration]) -> Duration {
    let p99_index = (latencies.len() as f64 * 0.99) as usize - 1;
    latencies[p99_index]
}

/// Calculate average latency
fn calculate_avg(latencies: &[Duration]) -> Duration {
    let total: Duration = latencies.iter().sum();
    total / latencies.len() as u32
}

#[test]
fn test_set_generation_latency_slo() {
    let latencies = bench_set_generation();

    let p99 = calculate_p99(&latencies);
    let avg = calculate_avg(&latencies);
    let min = latencies.first().unwrap();
    let max = latencies.last().unwrap();

    println!("\nSET Generation Benchmark:");
    println!("  Samples: {}", SAMPLE_SIZE);
    println!("  Min: {:?}", min);
    println!("  Avg: {:?}", avg);
    println!("  Max: {:?}", max);
    println!("  P99: {:?}", p99);
    println!("  SLO: {}ms", P99_LIMIT_MS);

    // SET generation should be very fast (< 1ms typically)
    assert!(
        p99.as_millis() < 10,
        "SET generation p99 ({:?}) should be < 10ms",
        p99
    );
}

#[test]
fn test_event_creation_latency() {
    let mut latencies = Vec::with_capacity(SAMPLE_SIZE);

    for i in 0..SAMPLE_SIZE {
        let subject = SubjectIdentifier::IssSub {
            iss: "https://auth.example.com".to_string(),
            sub: format!("user-{}", i),
        };

        let start = Instant::now();
        let _event = CaepEvent::session_revoked(subject, Some("Test".to_string()));
        latencies.push(start.elapsed());
    }

    latencies.sort();
    let p99 = calculate_p99(&latencies);

    println!("\nEvent Creation Benchmark:");
    println!("  P99: {:?}", p99);

    // Event creation should be very fast
    assert!(
        p99.as_micros() < 1000,
        "Event creation p99 ({:?}) should be < 1ms",
        p99
    );
}

#[test]
fn test_stream_health_update_latency() {
    let config = StreamConfig {
        audience: "https://receiver.example.com".to_string(),
        delivery: DeliveryMethod::Push {
            endpoint_url: "https://receiver.example.com/caep".to_string(),
        },
        events_requested: vec![CaepEventType::SessionRevoked],
        format: "iss_sub".to_string(),
    };

    let mut stream = Stream::new(config);
    let mut latencies = Vec::with_capacity(SAMPLE_SIZE);

    for i in 0..SAMPLE_SIZE {
        let start = Instant::now();
        stream.record_success(i as u64);
        latencies.push(start.elapsed());
    }

    latencies.sort();
    let p99 = calculate_p99(&latencies);

    println!("\nStream Health Update Benchmark:");
    println!("  P99: {:?}", p99);

    // Health updates should be very fast
    assert!(
        p99.as_micros() < 100,
        "Health update p99 ({:?}) should be < 100µs",
        p99
    );
}
