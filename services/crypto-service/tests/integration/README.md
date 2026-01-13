# Integration Tests

Integration tests for the crypto-service using Testcontainers.

## Prerequisites

- Docker installed and running
- Testcontainers C++ library

## Test Categories

### Platform Service Integration

- `logging_service_integration_test.cpp` - Tests LoggingClient with real logging-service
- `cache_service_integration_test.cpp` - Tests CacheClient with real cache-service

### End-to-End Tests

- `end_to_end_test.cpp` - Full crypto operation flows

## Running Integration Tests

```bash
# Build with integration tests
cmake -B build -DENABLE_TESTING=ON -DENABLE_INTEGRATION_TESTS=ON

# Run integration tests only
./build/crypto_integration_tests

# Run with specific test
./build/crypto_integration_tests --gtest_filter="*LoggingService*"
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TESTCONTAINERS_RYUK_DISABLED` | Disable Ryuk container | false |
| `DOCKER_HOST` | Docker daemon socket | unix:///var/run/docker.sock |

## Test Containers Used

- `logging-service:test` - Logging service container
- `cache-service:test` - Cache service container
- `redis:7-alpine` - Redis for cache service backend
