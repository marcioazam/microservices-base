import Config

# Test-specific configuration
config :logger, level: :warning

config :auth_platform,
  # Fast settings for testing
  circuit_breaker: [
    failure_threshold: 3,
    timeout_ms: 100,
    half_open_max_requests: 1
  ],
  retry: [
    max_retries: 2,
    initial_delay_ms: 10,
    max_delay_ms: 100
  ],
  rate_limiter: [
    rate: 1000,
    burst_size: 1000
  ]

# Configure StreamData for property tests
config :stream_data,
  max_runs: 100
