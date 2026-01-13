import Config

# Common configuration for all environments
config :logger, :console,
  format: "$time $metadata[$level] $message\n",
  metadata: [:request_id, :correlation_id]

# Auth Platform configuration
config :auth_platform,
  circuit_breaker: [
    failure_threshold: 5,
    success_threshold: 2,
    timeout_ms: 30_000,
    half_open_max_requests: 3
  ],
  retry: [
    max_retries: 3,
    initial_delay_ms: 100,
    max_delay_ms: 10_000,
    multiplier: 2.0,
    jitter: true
  ],
  rate_limiter: [
    rate: 100,
    burst_size: 100
  ],
  bulkhead: [
    max_concurrent: 10,
    max_queue: 100,
    queue_timeout_ms: 5_000
  ]

# Import environment specific config
import_config "#{config_env()}.exs"
