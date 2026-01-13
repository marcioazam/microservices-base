# Implementation Plan: Auth Platform Elixir Library

## Overview

This implementation plan breaks down the Elixir shared library into discrete, incremental tasks. Each task builds on previous work, ensuring no orphaned code. The library follows functional programming principles with behaviours, protocols, GenServer for stateful components, and StreamData for property-based testing.

## Tasks

- [x] 1. Project Setup and Core Structure
  - [x] 1.1 Create umbrella project structure under `libs/elixir/`
    - Initialize mix umbrella project with `mix new auth_platform_umbrella --umbrella`
    - Create three apps: `auth_platform`, `auth_platform_clients`, `auth_platform_testing`
    - Configure mix.exs with Elixir 1.15+, OTP 26+ requirements
    - Add dependencies: jason, telemetry, stream_data, dialyxir, credo, ex_doc
    - _Requirements: 13.1, 13.2_

  - [x] 1.2 Configure development tooling
    - Create `.formatter.exs` with project-wide formatting rules
    - Create `.credo.exs` with strict linting configuration
    - Configure Dialyzer PLT paths in mix.exs
    - Add test configuration with StreamData support
    - _Requirements: 13.5, 13.6_

- [x] 2. Functional Types Module
  - [x] 2.1 Implement Result type (`AuthPlatform.Functional.Result`)
    - Create `ok/1`, `error/1` constructors
    - Implement `map/2`, `flat_map/2`, `match/3` operations
    - Implement `unwrap!/1`, `unwrap_or/2`, `unwrap_error!/1`
    - Implement `is_ok?/1`, `is_error?/1` predicates
    - Add `try_result/1` macro for exception wrapping
    - Add typespecs for all functions
    - _Requirements: 1.1, 1.3, 1.4, 1.5, 1.7, 1.8_

  - [x] 2.2 Write property tests for Result type
    - **Property 1: Functional Types Round-Trip**
    - **Property 3: Result Constructor Consistency**
    - **Validates: Requirements 1.1, 1.3, 1.4**

  - [x] 2.3 Write property tests for Result functor laws
    - **Property 2: Functor Law Compliance (Result)**
    - **Validates: Requirements 1.5**

  - [x] 2.4 Implement Option type (`AuthPlatform.Functional.Option`)
    - Create `some/1`, `none/0` constructors
    - Implement `from_nullable/1` for nil conversion
    - Implement `map/2`, `flat_map/2`, `match/3` operations
    - Implement `unwrap!/1`, `unwrap_or/2`
    - Implement `is_some?/1`, `is_none?/1` predicates
    - Add typespecs for all functions
    - _Requirements: 1.2, 1.6_

  - [x] 2.5 Write property tests for Option type
    - **Property 1: Functional Types Round-Trip (Option)**
    - **Property 2: Functor Law Compliance (Option)**
    - **Property 3: Option Constructor Consistency**
    - **Validates: Requirements 1.2, 1.6**

- [x] 3. Checkpoint - Functional Types Complete
  - Ensure all tests pass, ask the user if questions arise.
  - Run `mix test` and `mix dialyzer`

- [x] 4. Error Handling Module
  - [x] 4.1 Implement AppError struct (`AuthPlatform.Errors.AppError`)
    - Define struct with code, message, details, correlation_id, cause, retryable fields
    - Define error_code type with all supported codes
    - Implement HTTP status mapping (`http_status/1`)
    - Implement gRPC code mapping (`grpc_code/1`)
    - Implement `is_retryable?/1` classification
    - _Requirements: 2.1, 2.4, 2.5, 2.6_

  - [x] 4.2 Implement error factory functions
    - Create `not_found/1`, `validation/1`, `unauthorized/1`
    - Create `internal/1`, `rate_limited/0`, `timeout/1`, `unavailable/1`
    - Implement `with_details/2`, `with_correlation_id/2`
    - Implement `to_api_response/1` with PII redaction
    - _Requirements: 2.2, 2.3, 2.7_

  - [x] 4.3 Write property tests for error handling
    - **Property 4: Error Code Mapping Consistency**
    - **Property 5: Retryable Error Classification**
    - **Validates: Requirements 2.4, 2.5, 2.6**

- [x] 5. Validation Module
  - [x] 5.1 Implement core validation infrastructure (`AuthPlatform.Validation`)
    - Define validation_error and validation_result types
    - Implement `validate_all/1` for error accumulation
    - Implement `validate_field/3` for named field validation
    - _Requirements: 3.1, 3.6, 3.7_

  - [x] 5.2 Implement string validators
    - Create `required/0` validator
    - Create `min_length/1`, `max_length/1` validators
    - Create `matches_regex/1` validator
    - Create `one_of/1` validator for enum values
    - _Requirements: 3.2_

  - [x] 5.3 Implement numeric validators
    - Create `positive/0`, `non_negative/0` validators
    - Create `in_range/2`, `min/1`, `max/1` validators
    - _Requirements: 3.3_

  - [x] 5.4 Implement collection validators and composition
    - Create `min_size/1`, `max_size/1`, `unique_elements/0` validators
    - Implement `all/1`, `any/1`, `not_/1` composition functions
    - _Requirements: 3.4, 3.5_

  - [x] 5.5 Write property tests for validation
    - **Property 6: Validation Error Accumulation**
    - **Property 7: Validator Composition**
    - **Property 8: String Validator Correctness**
    - **Property 9: Numeric Validator Correctness**
    - **Validates: Requirements 3.2, 3.3, 3.5, 3.6**

- [x] 6. Checkpoint - Core Modules Complete
  - Ensure all tests pass, ask the user if questions arise.
  - Run `mix test` and `mix dialyzer`

- [x] 7. Domain Primitives Module
  - [x] 7.1 Implement Email type (`AuthPlatform.Domain.Email`)
    - Create struct with value field
    - Implement `new/1` with RFC 5322 regex validation
    - Implement `new!/1` raising variant
    - Implement `String.Chars` protocol
    - Implement `Jason.Encoder` protocol
    - _Requirements: 4.1, 4.7, 4.8, 4.9_

  - [x] 7.2 Implement UUID type (`AuthPlatform.Domain.UUID`)
    - Create struct with value field
    - Implement `generate/0` for UUID v4 generation
    - Implement `new/1` with RFC 4122 validation
    - Implement `String.Chars` and `Jason.Encoder` protocols
    - _Requirements: 4.2, 4.8, 4.9_

  - [x] 7.3 Implement ULID type (`AuthPlatform.Domain.ULID`)
    - Create struct with value field
    - Implement `generate/0` for time-ordered ID generation
    - Implement `new/1` with ULID format validation
    - Implement `String.Chars` and `Jason.Encoder` protocols
    - _Requirements: 4.3, 4.8, 4.9_

  - [x] 7.4 Implement Money type (`AuthPlatform.Domain.Money`)
    - Create struct with amount (integer cents) and currency fields
    - Define supported currencies (USD, EUR, GBP, BRL, JPY)
    - Implement `new/2` with validation
    - Implement `add/2` with currency matching check
    - Implement `Jason.Encoder` protocol
    - _Requirements: 4.4, 4.8_

  - [x] 7.5 Implement PhoneNumber type (`AuthPlatform.Domain.PhoneNumber`)
    - Create struct with value field
    - Implement `new/1` with E.164 format validation
    - Implement `String.Chars` and `Jason.Encoder` protocols
    - _Requirements: 4.5, 4.8, 4.9_

  - [x] 7.6 Implement URL type (`AuthPlatform.Domain.URL`)
    - Create struct with value and scheme fields
    - Implement `new/1` with scheme validation (http, https)
    - Implement `String.Chars` and `Jason.Encoder` protocols
    - _Requirements: 4.6, 4.8, 4.9_

  - [x] 7.7 Write property tests for domain primitives
    - **Property 10: Domain Primitive Validation**
    - **Property 11: Domain Primitive Serialization Round-Trip**
    - **Validates: Requirements 4.1-4.9**

- [x] 8. Checkpoint - Domain Primitives Complete
  - Ensure all tests pass, ask the user if questions arise.
  - Run `mix test` and `mix dialyzer`

- [x] 9. Resilience Registry Setup
  - [x] 9.1 Create resilience registry and supervisor
    - Create `AuthPlatform.Resilience.Registry` using Elixir Registry
    - Create `AuthPlatform.Resilience.Supervisor` for managing GenServers
    - Add to application supervision tree
    - _Requirements: 5.8, 7.5, 8.1_

- [x] 10. Circuit Breaker Module
  - [x] 10.1 Implement CircuitBreaker GenServer (`AuthPlatform.Resilience.CircuitBreaker`)
    - Define state struct with name, config, state, failures, successes, timestamps
    - Define config struct with failure_threshold, success_threshold, timeout_ms, half_open_max_requests
    - Implement `start_link/1` with Registry registration
    - Implement `allow_request?/1` with state transition logic
    - Implement `record_success/1`, `record_failure/1`
    - Implement `get_state/1`, `reset/1`
    - Implement `execute/2` convenience function
    - Add telemetry events for state transitions
    - _Requirements: 5.1-5.9_

  - [x] 10.2 Write property tests for circuit breaker
    - **Property 12: Circuit Breaker State Machine**
    - **Validates: Requirements 5.2, 5.3, 5.4, 5.5, 5.6**

- [x] 11. Retry Policy Module
  - [x] 11.1 Implement Retry module (`AuthPlatform.Resilience.Retry`)
    - Define config struct with max_retries, initial_delay_ms, max_delay_ms, multiplier, jitter
    - Implement `default_config/0`
    - Implement `delay_for_attempt/2` with exponential backoff calculation
    - Implement `should_retry?/3` with error classification
    - Implement `execute/2` with automatic retry loop
    - Add telemetry events for retry attempts
    - _Requirements: 6.1-6.7_

  - [x] 11.2 Write property tests for retry policy
    - **Property 13: Retry Policy Exponential Backoff**
    - **Property 14: Retry Policy Execution**
    - **Validates: Requirements 6.2, 6.4, 6.5, 6.6**

- [x] 12. Rate Limiter Module
  - [x] 12.1 Implement RateLimiter GenServer (`AuthPlatform.Resilience.RateLimiter`)
    - Define state struct with name, config, tokens, last_refill
    - Define config struct with rate (tokens/sec) and burst_size
    - Implement `start_link/1` with Registry registration
    - Implement `allow?/1` for non-blocking check
    - Implement `acquire/2` with blocking wait and timeout
    - Implement token refill logic based on elapsed time
    - Add telemetry events for rate limit exceeded
    - _Requirements: 7.1-7.6_

  - [x] 12.2 Write property tests for rate limiter
    - **Property 15: Rate Limiter Token Bucket**
    - **Validates: Requirements 7.1, 7.2, 7.3**

- [x] 13. Bulkhead Module
  - [x] 13.1 Implement Bulkhead GenServer (`AuthPlatform.Resilience.Bulkhead`)
    - Define state struct with name, config, active count, queue
    - Define config struct with max_concurrent, max_queue, queue_timeout_ms
    - Implement `start_link/1` with Registry registration
    - Implement `execute/2` with semaphore-like behavior
    - Implement `available_permits/1`
    - Implement queue management with timeout handling
    - Add telemetry events for rejection metrics
    - _Requirements: 8.1-8.6_

  - [x] 13.2 Write property tests for bulkhead
    - **Property 16: Bulkhead Isolation**
    - **Validates: Requirements 8.1, 8.3, 8.4**

- [x] 14. Checkpoint - Resilience Patterns Complete
  - Ensure all tests pass, ask the user if questions arise.
  - Run `mix test` and `mix dialyzer`

- [x] 15. Codec Module
  - [x] 15.1 Implement JSON codec (`AuthPlatform.Codec.JSON`)
    - Implement `encode/1`, `encode!/1` using Jason
    - Implement `encode_pretty/1` for formatted output
    - Implement `decode/1`, `decode!/1` with error handling
    - _Requirements: 9.1, 9.3, 9.4, 9.5, 9.6_

  - [x] 15.2 Implement Base64 codec (`AuthPlatform.Codec.Base64`)
    - Implement `encode/1` for standard Base64
    - Implement `encode_url_safe/1` for URL-safe variant
    - Implement `decode/1`, `decode!/1` with error handling
    - Implement `decode_url_safe/1` for URL-safe decoding
    - _Requirements: 9.2, 9.3, 9.4, 9.5_

  - [x] 15.3 Write property tests for codecs
    - **Property 17: JSON Codec Round-Trip**
    - **Property 18: Base64 Codec Round-Trip**
    - **Validates: Requirements 9.1, 9.2**

- [x] 16. Security Module
  - [x] 16.1 Implement security utilities (`AuthPlatform.Security`)
    - Implement `constant_time_compare/2` using :crypto.hash_equals
    - Implement `generate_token/1` using :crypto.strong_rand_bytes
    - Implement `mask_sensitive/2` with configurable visible chars
    - Implement `sanitize_html/1` with entity encoding
    - Implement `sanitize_sql/1` for basic SQL escaping
    - Implement `detect_sql_injection/1` with pattern matching
    - _Requirements: 11.1-11.5_

  - [x] 16.2 Write property tests for security utilities
    - **Property 19: Constant Time Compare Correctness**
    - **Property 20: Token Generation Uniqueness**
    - **Property 21: HTML Sanitization**
    - **Property 22: Sensitive Data Masking**
    - **Validates: Requirements 11.1, 11.2, 11.3, 11.4**

- [x] 17. Observability Module
  - [x] 17.1 Implement structured logging (`AuthPlatform.Observability.Logger`)
    - Create JSON formatter for Logger backend
    - Implement correlation ID propagation via process dictionary
    - Implement `with_correlation_id/2` macro for scoped correlation
    - Implement PII redaction for sensitive fields
    - _Requirements: 10.1, 10.2, 10.3, 10.4_

  - [x] 17.2 Implement telemetry definitions (`AuthPlatform.Observability.Telemetry`)
    - Define telemetry event specs for all resilience patterns
    - Create telemetry handler attachment helpers
    - Integrate with OpenTelemetry for distributed tracing
    - _Requirements: 10.5, 10.6_

- [x] 18. Checkpoint - Core Library Complete
  - Ensure all tests pass, ask the user if questions arise.
  - Run `mix test`, `mix dialyzer`, and `mix credo --strict`

- [x] 19. Testing Utilities App
  - [x] 19.1 Implement StreamData generators (`AuthPlatform.Testing.Generators`)
    - Create `email_generator/0` producing valid RFC 5322 emails
    - Create `uuid_generator/0` producing valid UUID v4 strings
    - Create `ulid_generator/0` producing valid ULID strings
    - Create `money_generator/0` producing valid Money structs
    - Create `phone_number_generator/0` producing E.164 format numbers
    - Create `url_generator/0` producing valid URLs
    - _Requirements: 12.1-12.6_

  - [x] 19.2 Implement test helpers (`AuthPlatform.Testing.Helpers`)
    - Create circuit breaker test helpers (force state transitions)
    - Create retry test helpers (mock retryable operations)
    - Create rate limiter test helpers
    - _Requirements: 12.7_

- [x] 20. Platform Clients App
  - [x] 20.1 Implement Logging Service client (`AuthPlatform.Clients.Logging`)
    - Create client module with logging interface
    - Implement `log/3` with level, message, metadata
    - Support log levels: debug, info, warn, error
    - Add circuit breaker protection
    - _Requirements: 14.1, 14.3, 14.5_

  - [x] 20.2 Implement Cache Service client (`AuthPlatform.Clients.Cache`)
    - Create client module with cache interface
    - Implement `get/1`, `set/3` (with TTL), `delete/1`
    - Add circuit breaker protection
    - Add telemetry events
    - _Requirements: 14.2, 14.4, 14.5, 14.6_

- [x] 21. Documentation and Final Polish
  - [x] 21.1 Add comprehensive documentation
    - Write module-level @moduledoc for all public modules
    - Write function-level @doc for all public functions
    - Add usage examples in documentation
    - Create README.md with getting started guide
    - Create CHANGELOG.md
    - _Requirements: 13.3_

  - [x] 21.2 Ensure typespec coverage
    - Add @spec for all public functions
    - Run Dialyzer and fix all warnings
    - _Requirements: 13.4, 13.5_

- [x] 22. Final Checkpoint - Library Complete
  - Ensure all tests pass with 80%+ coverage on core modules
  - Run full quality suite: `mix test`, `mix dialyzer`, `mix credo --strict`
  - Generate documentation with `mix docs`
  - _Requirements: 13.7, 13.8_

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties using StreamData
- Unit tests validate specific examples and edge cases
- The umbrella structure allows independent development of core lib, clients, and testing utilities
