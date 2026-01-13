import Config

# Development-specific configuration
config :logger, level: :debug

config :auth_platform,
  # More lenient settings for development
  circuit_breaker: [
    failure_threshold: 10,
    timeout_ms: 10_000
  ]
