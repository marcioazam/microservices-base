# Implementation Plan: SMS Service

## Overview

Implementação incremental do microserviço SMS em Python/FastAPI, seguindo a ordem: infraestrutura → domínio → serviços → API → worker → providers → observabilidade → deploy.

## Tasks

- [-] 1. Setup do projeto e infraestrutura base
  - [x] 1.1 Criar estrutura de diretórios e pyproject.toml
    - Criar `services/sms-service/` com estrutura definida no design
    - Configurar dependências: fastapi, sqlalchemy, pydantic, aio-pika, redis, httpx
    - Configurar pytest, hypothesis, mypy, ruff
    - _Requirements: 9.1_

  - [x] 1.2 Configurar settings com Pydantic Settings
    - Criar `src/config/settings.py` com todas as configurações
    - Suporte a variáveis de ambiente e valores default
    - _Requirements: 6.6_

  - [-] 1.3 Criar modelos SQLAlchemy e migrations
    - Implementar `sms_requests`, `sms_events`, `otp_challenges`
    - Criar migration inicial com Alembic
    - Índices conforme design
    - _Requirements: 7.1, 7.3, 7.4, 7.5_

  - [ ] 1.4 Write property test for idempotency_key uniqueness
    - **Property 1: Idempotency Guarantee (database constraint)**
    - **Validates: Requirements 7.4**

- [ ] 2. Domain layer e validadores
  - [ ] 2.1 Implementar entidades de domínio
    - Criar `SMSRequest`, `SMSEvent`, `OTPChallenge` entities
    - Implementar enums `SMSStatus`, `OTPStatus`
    - _Requirements: 5.2_

  - [ ] 2.2 Implementar validadores
    - E.164 validator com regex
    - Message length validator (max 1600)
    - Charset validator
    - _Requirements: 1.6, 1.7, 6.2_

  - [ ] 2.3 Write property test for E.164 validation
    - **Property 2: E.164 Validation**
    - **Validates: Requirements 1.6, 6.2**

  - [ ] 2.4 Write property test for message length validation
    - **Property 3: Message Length Validation**
    - **Validates: Requirements 1.7, 6.2**

  - [ ] 2.5 Implementar utilitários de hashing e masking
    - SHA-256 para message_hash
    - bcrypt para otp_hash
    - Phone masking (últimos 4 dígitos)
    - _Requirements: 2.1, 6.4, 6.5_

  - [ ] 2.6 Write property test for phone masking
    - **Property 14: Phone Number Masking**
    - **Validates: Requirements 6.4**

- [ ] 3. Checkpoint - Validar fundação
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 4. Infrastructure layer - Database
  - [ ] 4.1 Implementar repositórios
    - `SMSRequestRepository` com CRUD e idempotency lookup
    - `SMSEventRepository` para event sourcing
    - `OTPChallengeRepository` com update de attempts/status
    - _Requirements: 7.1, 7.2, 7.3_

  - [ ] 4.2 Write property test for event sourcing
    - **Property 13: Event Sourcing Completeness**
    - **Validates: Requirements 5.4, 7.2**

- [ ] 5. Infrastructure layer - Queue e Cache
  - [ ] 5.1 Implementar RabbitMQ publisher
    - Conexão async com aio-pika
    - Publish para exchange sms.direct
    - Suporte a DLQ
    - _Requirements: 3.1, 3.3_

  - [ ] 5.2 Implementar Redis rate limiter
    - Sliding window rate limit
    - Keys por phone number e por client
    - _Requirements: 2.8, 6.3_

  - [ ] 5.3 Write property test for rate limiting
    - **Property 9: Rate Limit Enforcement**
    - **Validates: Requirements 2.8, 6.3, 6.7**

- [ ] 6. SMS Service implementation
  - [ ] 6.1 Implementar SMSService.send_sms
    - Idempotency check
    - Validation
    - Persist with status=accepted
    - Enqueue message
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

  - [ ] 6.2 Write property test for idempotency
    - **Property 1: Idempotency Guarantee**
    - **Validates: Requirements 1.2, 3.4**

  - [ ] 6.3 Implementar SMSService.get_status
    - Query by request_id
    - Include event history
    - _Requirements: 5.1_

  - [ ] 6.4 Implementar SMSService.process_webhook
    - Signature validation
    - Status update
    - Event creation
    - _Requirements: 5.3, 5.6_

  - [ ] 6.5 Write property test for webhook signature validation
    - **Property 16: Webhook Signature Validation**
    - **Validates: Requirements 5.6**

- [ ] 7. OTP Service implementation
  - [ ] 7.1 Implementar OTPService.generate_otp
    - Rate limit check
    - Secure random generation
    - Hash storage (bcrypt)
    - Trigger SMS send
    - _Requirements: 2.1, 2.2, 2.8, 2.9_

  - [ ] 7.2 Write property test for OTP hash storage
    - **Property 4: OTP Hash Storage**
    - **Validates: Requirements 2.1, 6.5, 7.6**

  - [ ] 7.3 Implementar OTPService.validate_otp
    - Expiration check
    - Attempt tracking
    - Hash verification
    - Status transitions
    - _Requirements: 2.3, 2.4, 2.5, 2.6, 2.7_

  - [ ] 7.4 Write property test for OTP round-trip
    - **Property 5: OTP Round-Trip Validation**
    - **Validates: Requirements 2.3, 2.4**

  - [ ] 7.5 Write property test for OTP single-use
    - **Property 6: OTP Single-Use**
    - **Validates: Requirements 2.4**

  - [ ] 7.6 Write property test for OTP exhaustion
    - **Property 7: OTP Attempt Exhaustion**
    - **Validates: Requirements 2.5, 2.6**

  - [ ] 7.7 Write property test for OTP expiration
    - **Property 8: OTP Expiration**
    - **Validates: Requirements 2.7**

- [ ] 8. Checkpoint - Validar services
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 9. Provider Gateway implementation
  - [ ] 9.1 Implementar interface base SMSProvider
    - Abstract methods: send, validate_webhook
    - ProviderResponse model
    - ProviderError with classification
    - _Requirements: 4.1_

  - [ ] 9.2 Implementar TwilioProvider
    - HTTP client com httpx
    - Error mapping (transient vs permanent)
    - Webhook signature validation
    - _Requirements: 4.1, 4.5_

  - [ ] 9.3 Implementar MessageBirdProvider (fallback)
    - HTTP client com httpx
    - Error mapping
    - _Requirements: 4.1, 4.5_

  - [ ] 9.4 Write property test for error classification
    - **Property 11: Error Classification Determinism**
    - **Validates: Requirements 4.2, 4.4, 4.5**

  - [ ] 9.5 Implementar CircuitBreaker
    - State machine: closed → open → half-open
    - Failure threshold e recovery timeout
    - _Requirements: 4.3, 4.6_

  - [ ] 9.6 Write property test for circuit breaker
    - **Property 12: Circuit Breaker State Machine**
    - **Validates: Requirements 4.3, 4.6**

  - [ ] 9.7 Implementar ProviderGateway
    - Primary + fallback routing
    - Circuit breaker integration
    - Fallback logging
    - _Requirements: 4.2, 4.3, 4.7_

- [ ] 10. Worker implementation
  - [ ] 10.1 Implementar SMSWorkerConsumer
    - RabbitMQ consumer com aio-pika
    - Message processing loop
    - Idempotency check before send
    - Status updates
    - _Requirements: 3.1, 3.4, 3.5_

  - [ ] 10.2 Implementar RetryHandler
    - Exponential backoff calculation
    - Max retries check
    - DLQ routing
    - _Requirements: 3.2, 3.3_

  - [ ] 10.3 Write property test for exponential backoff
    - **Property 10: Exponential Backoff Retry**
    - **Validates: Requirements 3.2, 3.3**

  - [ ] 10.4 Implementar ordering por phone number
    - Routing key por phone number
    - Single consumer per queue partition
    - _Requirements: 3.6_

- [ ] 11. Checkpoint - Validar worker e providers
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 12. API Layer implementation
  - [ ] 12.1 Implementar FastAPI app e dependencies
    - App factory com lifespan
    - Dependency injection container
    - Exception handlers
    - _Requirements: 1.1_

  - [ ] 12.2 Implementar JWT authentication middleware
    - Token validation
    - Service identity extraction
    - 401 on invalid/missing token
    - _Requirements: 6.1_

  - [ ] 12.3 Write property test for JWT authentication
    - **Property 15: JWT Authentication**
    - **Validates: Requirements 6.1**

  - [ ] 12.4 Implementar rate limit middleware
    - Per-client e per-phone limiting
    - 429 response com retry-after
    - _Requirements: 6.3, 6.7_

  - [ ] 12.5 Implementar SMS routes
    - POST /v1/sms/send
    - GET /v1/sms/{request_id}
    - POST /v1/sms/webhook
    - _Requirements: 1.1, 5.1, 5.3_

  - [ ] 12.6 Implementar OTP routes
    - POST /v1/otp/generate
    - POST /v1/otp/validate
    - _Requirements: 2.1, 2.3_

  - [ ] 12.7 Implementar health routes
    - GET /health/live
    - GET /health/ready
    - _Requirements: 8.5_

- [ ] 13. Observability implementation
  - [ ] 13.1 Configurar OpenTelemetry tracing
    - HTTP instrumentation
    - SQLAlchemy instrumentation
    - aio-pika instrumentation
    - W3C Trace Context propagation
    - _Requirements: 8.1, 8.4_

  - [ ] 13.2 Write property test for trace propagation
    - **Property 17: Trace Context Propagation**
    - **Validates: Requirements 5.5, 8.4**

  - [ ] 13.3 Configurar Prometheus metrics
    - sms_sent_total, sms_failed_total
    - provider_latency_ms histogram
    - otp_verifications_total
    - queue_lag gauge
    - _Requirements: 8.2_

  - [ ] 13.4 Configurar structured logging
    - JSON format com structlog
    - Required fields: request_id, idempotency_key, provider, to_masked, status
    - Log levels por tipo de erro
    - _Requirements: 8.3, 8.6_

- [ ] 14. Checkpoint - Validar API e observabilidade
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 15. Dockerfiles e build
  - [ ] 15.1 Criar Dockerfile para API
    - Multi-stage build
    - Non-root user
    - Health check
    - _Requirements: 9.1_

  - [ ] 15.2 Criar Dockerfile para Worker
    - Multi-stage build
    - Non-root user
    - _Requirements: 9.1_

- [ ] 16. Kubernetes manifests
  - [ ] 16.1 Criar Deployment para API
    - Resource requests/limits
    - Liveness/readiness probes
    - ConfigMap/Secret refs
    - _Requirements: 9.1, 9.3, 9.6_

  - [ ] 16.2 Criar Deployment para Worker
    - Resource requests/limits
    - ConfigMap/Secret refs
    - _Requirements: 9.1, 9.3, 9.6_

  - [ ] 16.3 Criar Service e HPA
    - ClusterIP service para API
    - HPA para API (CPU/memory)
    - HPA para Worker (queue lag)
    - _Requirements: 9.2, 9.5_

  - [ ] 16.4 Criar NetworkPolicy
    - Ingress: API Gateway only
    - Egress: PostgreSQL, RabbitMQ, Redis, provider APIs
    - _Requirements: 9.4_

  - [ ] 16.5 Criar ConfigMap, Secrets refs e PDB
    - ConfigMap para non-sensitive config
    - ExternalSecret refs para credentials
    - PodDisruptionBudget para HA
    - _Requirements: 9.3, 9.7_

- [ ] 17. Integration tests
  - [ ] 17.1 Write integration tests for API endpoints
    - Test full request flow
    - Test error responses
    - _Requirements: 1.1, 2.1, 5.1_

  - [ ] 17.2 Write integration tests for queue processing
    - Test message consumption
    - Test retry behavior
    - Test DLQ routing
    - _Requirements: 3.1, 3.2, 3.3_

- [ ] 18. Final checkpoint
  - Ensure all tests pass, ask the user if questions arise.
  - Verify all requirements are covered
  - Run full test suite with coverage report

## Notes

- All tasks are required for comprehensive coverage
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (17 properties)
- Unit tests validate specific examples and edge cases
- Python 3.12+ required for modern type hints
- Hypothesis framework for property-based testing
