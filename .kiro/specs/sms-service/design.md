# Design Document: SMS Service

## Overview

O `sms-service` é um microserviço Python/FastAPI para envio de SMS transacionais e OTP. A arquitetura segue o padrão de processamento assíncrono com filas, garantindo alta disponibilidade, idempotência end-to-end e integração resiliente com múltiplos provedores.

### Decisões Arquiteturais

| Decisão | Escolha | Justificativa |
|---------|---------|---------------|
| Framework HTTP | FastAPI | Async nativo, OpenAPI automático, validação Pydantic |
| Message Queue | RabbitMQ | Suporte robusto a DLQ, acknowledgments, ordering por routing key |
| Autenticação | JWT | Stateless, padrão da plataforma, suporte a service identity |
| Cache | Redis | Rate limiting, circuit breaker state, TTL nativo |
| ORM | SQLAlchemy 2.0 | Async support, type hints, migrations via Alembic |

### Trade-offs

- **RabbitMQ vs Kafka**: RabbitMQ escolhido por simplicidade operacional e suporte nativo a DLQ. Kafka seria melhor para volume extremo (>100k msg/s).
- **JWT vs mTLS**: JWT escolhido por compatibilidade com API Gateway existente. mTLS adicionaria complexidade de PKI.
- **Redis para rate limit**: Alternativa seria rate limit no API Gateway, mas Redis permite granularidade por phone number.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              SMS Service                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐  │
│  │   FastAPI   │    │   Worker    │    │  Provider   │    │   Health    │  │
│  │     API     │───▶│  Consumer   │───▶│   Gateway   │    │   Probes    │  │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘    └─────────────┘  │
│         │                  │                  │                             │
│         ▼                  ▼                  ▼                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                     │
│  │  Idempotency│    │   Retry     │    │  Circuit    │                     │
│  │    Store    │    │   Handler   │    │  Breaker    │                     │
│  └─────────────┘    └─────────────┘    └─────────────┘                     │
└─────────────────────────────────────────────────────────────────────────────┘
         │                  │                  │
         ▼                  ▼                  ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ PostgreSQL  │    │  RabbitMQ   │    │   Twilio    │    │   Redis     │
│  (persist)  │    │   (queue)   │    │ MessageBird │    │  (cache)    │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
```

## Components and Interfaces

### 1. API Layer (`src/api/`)

```python
# src/api/routes/sms.py
class SendSMSRequest(BaseModel):
    to: str  # E.164 format
    message: str  # max 1600 chars
    sender_id: str | None = None
    metadata: dict[str, str] | None = None
    idempotency_key: str  # UUID, required

class SendSMSResponse(BaseModel):
    request_id: UUID
    status: SMSStatus
    created_at: datetime
    accepted_at: datetime

# Endpoints
POST /v1/sms/send          # Enviar SMS
GET  /v1/sms/{request_id}  # Consultar status
POST /v1/sms/webhook       # Callback do provedor

# src/api/routes/otp.py
class GenerateOTPRequest(BaseModel):
    to: str  # E.164 format
    ttl_seconds: int = 300  # default 5 min
    max_attempts: int = 3
    idempotency_key: str

class ValidateOTPRequest(BaseModel):
    challenge_id: UUID
    code: str

# Endpoints
POST /v1/otp/generate      # Gerar e enviar OTP
POST /v1/otp/validate      # Validar OTP
```

### 2. Domain Layer (`src/domain/`)

```python
# src/domain/entities/sms.py
class SMSStatus(str, Enum):
    ACCEPTED = "accepted"
    QUEUED = "queued"
    SENDING = "sending"
    SENT = "sent"
    DELIVERED = "delivered"
    FAILED = "failed"
    EXPIRED = "expired"

class SMSRequest:
    id: UUID
    idempotency_key: str
    to: str  # E.164
    message_hash: str  # SHA-256
    sender_id: str | None
    metadata: dict | None
    provider: str | None
    status: SMSStatus
    error_code: str | None
    error_detail: str | None
    created_at: datetime
    updated_at: datetime

# src/domain/entities/otp.py
class OTPStatus(str, Enum):
    PENDING = "pending"
    VERIFIED = "verified"
    EXHAUSTED = "exhausted"
    EXPIRED = "expired"

class OTPChallenge:
    id: UUID
    to: str
    otp_hash: str  # bcrypt
    ttl_expires_at: datetime
    attempts: int
    max_attempts: int
    status: OTPStatus
```

### 3. Service Layer (`src/services/`)

```python
# src/services/sms_service.py
class SMSService:
    async def send_sms(self, request: SendSMSRequest) -> SendSMSResponse:
        """
        1. Check idempotency (return cached if exists)
        2. Validate E.164 format
        3. Check rate limits
        4. Persist request with status=ACCEPTED
        5. Enqueue to RabbitMQ
        6. Return response
        """

    async def get_status(self, request_id: UUID) -> SMSStatusResponse:
        """Query status and event history"""

    async def process_webhook(self, payload: WebhookPayload) -> None:
        """Validate signature, update status, create event"""

# src/services/otp_service.py
class OTPService:
    async def generate_otp(self, request: GenerateOTPRequest) -> GenerateOTPResponse:
        """
        1. Check rate limit for phone number
        2. Generate 6-digit secure random code
        3. Hash with bcrypt, store hash only
        4. Trigger SMS send with plaintext code
        5. Return challenge_id
        """

    async def validate_otp(self, request: ValidateOTPRequest) -> ValidateOTPResponse:
        """
        1. Load challenge by ID
        2. Check expiration
        3. Check attempt count
        4. Verify code against hash
        5. Update status (verified/exhausted)
        """
```

### 4. Worker Layer (`src/worker/`)

```python
# src/worker/consumer.py
class SMSWorkerConsumer:
    async def process_message(self, message: SMSQueueMessage) -> None:
        """
        1. Check idempotency (skip if already sent)
        2. Update status to SENDING
        3. Call Provider Gateway
        4. Update status to SENT/FAILED
        5. ACK message (or NACK for retry)
        """

# src/worker/retry_handler.py
class RetryHandler:
    BASE_DELAY = 2  # seconds
    MAX_RETRIES = 5

    def calculate_delay(self, attempt: int) -> float:
        """Exponential backoff: 2^attempt seconds"""
        return min(self.BASE_DELAY ** attempt, 60)

    def should_retry(self, error: ProviderError) -> bool:
        """Only retry transient errors"""
        return error.is_transient
```

### 5. Provider Gateway (`src/providers/`)

```python
# src/providers/base.py
class SMSProvider(ABC):
    @abstractmethod
    async def send(self, to: str, message: str, sender_id: str | None) -> ProviderResponse:
        """Send SMS via provider API"""

    @abstractmethod
    def validate_webhook(self, payload: bytes, signature: str) -> bool:
        """Validate webhook signature"""

# src/providers/twilio.py
class TwilioProvider(SMSProvider):
    """Primary provider implementation"""

# src/providers/messagebird.py
class MessageBirdProvider(SMSProvider):
    """Fallback provider implementation"""

# src/providers/gateway.py
class ProviderGateway:
    def __init__(self, primary: SMSProvider, fallback: SMSProvider):
        self.primary = primary
        self.fallback = fallback
        self.circuit_breaker = CircuitBreaker(
            failure_threshold=5,
            recovery_timeout=30
        )

    async def send(self, to: str, message: str, sender_id: str | None) -> ProviderResponse:
        """
        1. Check circuit breaker state
        2. If open, use fallback
        3. If closed/half-open, try primary
        4. On failure, update circuit state
        5. On transient failure, raise for retry
        6. On permanent failure, return failed
        """
```

### 6. Infrastructure Layer (`src/infrastructure/`)

```python
# src/infrastructure/database/repositories.py
class SMSRequestRepository:
    async def create(self, request: SMSRequest) -> SMSRequest
    async def get_by_id(self, id: UUID) -> SMSRequest | None
    async def get_by_idempotency_key(self, key: str) -> SMSRequest | None
    async def update_status(self, id: UUID, status: SMSStatus, **kwargs) -> None

class SMSEventRepository:
    async def create(self, event: SMSEvent) -> SMSEvent
    async def get_by_request_id(self, request_id: UUID) -> list[SMSEvent]

class OTPChallengeRepository:
    async def create(self, challenge: OTPChallenge) -> OTPChallenge
    async def get_by_id(self, id: UUID) -> OTPChallenge | None
    async def update(self, challenge: OTPChallenge) -> None

# src/infrastructure/queue/publisher.py
class RabbitMQPublisher:
    async def publish(self, message: SMSQueueMessage) -> None
    async def publish_to_dlq(self, message: SMSQueueMessage, error: str) -> None

# src/infrastructure/cache/rate_limiter.py
class RedisRateLimiter:
    async def check_limit(self, key: str, limit: int, window: int) -> bool
    async def increment(self, key: str, window: int) -> int
```

## Data Models

### PostgreSQL Schema

```sql
-- sms_requests table
CREATE TABLE sms_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key VARCHAR(64) NOT NULL UNIQUE,
    to_number VARCHAR(20) NOT NULL,  -- E.164
    message_hash VARCHAR(64) NOT NULL,  -- SHA-256
    sender_id VARCHAR(20),
    metadata JSONB,
    provider VARCHAR(50),
    status VARCHAR(20) NOT NULL DEFAULT 'accepted',
    error_code VARCHAR(50),
    error_detail TEXT,
    provider_message_id VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sms_requests_to_created ON sms_requests(to_number, created_at DESC);
CREATE INDEX idx_sms_requests_status ON sms_requests(status);
CREATE INDEX idx_sms_requests_provider_msg ON sms_requests(provider_message_id);

-- sms_events table (event sourcing)
CREATE TABLE sms_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id UUID NOT NULL REFERENCES sms_requests(id),
    event_type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sms_events_request ON sms_events(request_id, created_at);

-- otp_challenges table
CREATE TABLE otp_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    to_number VARCHAR(20) NOT NULL,
    otp_hash VARCHAR(100) NOT NULL,  -- bcrypt
    sms_request_id UUID REFERENCES sms_requests(id),
    ttl_expires_at TIMESTAMPTZ NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_otp_challenges_to_created ON otp_challenges(to_number, created_at DESC);
CREATE INDEX idx_otp_challenges_status ON otp_challenges(status);
```

### RabbitMQ Queue Structure

```yaml
exchanges:
  sms.direct:
    type: direct
    durable: true

queues:
  sms.send:
    durable: true
    arguments:
      x-dead-letter-exchange: sms.dlx
      x-dead-letter-routing-key: sms.failed
  sms.dlq:
    durable: true

bindings:
  - exchange: sms.direct
    queue: sms.send
    routing_key: sms.send
  - exchange: sms.dlx
    queue: sms.dlq
    routing_key: sms.failed
```

### Queue Message Schema

```python
class SMSQueueMessage(BaseModel):
    request_id: UUID
    to: str
    message: str  # plaintext for sending
    sender_id: str | None
    idempotency_key: str
    attempt: int = 0
    max_attempts: int = 5
    trace_id: str
    span_id: str
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Idempotency Guarantee

*For any* SMS request with a given `idempotency_key`, sending the same request multiple times SHALL return the same `request_id` and SHALL NOT create duplicate database records or duplicate sends to the provider.

**Validates: Requirements 1.2, 3.4**

### Property 2: E.164 Validation

*For any* string that does not match the E.164 format (^\+[1-9]\d{1,14}$), the SMS_Service SHALL reject the request with HTTP 400 and a validation error.

**Validates: Requirements 1.6, 6.2**

### Property 3: Message Length Validation

*For any* message string exceeding 1600 characters, the SMS_Service SHALL reject the request with HTTP 400 and a validation error.

**Validates: Requirements 1.7, 6.2**

### Property 4: OTP Hash Storage

*For any* OTP challenge created, the stored `otp_hash` value SHALL NOT equal the plaintext OTP code. Specifically, `stored_hash != plaintext_code` for all challenges.

**Validates: Requirements 2.1, 6.5, 7.6**

### Property 5: OTP Round-Trip Validation

*For any* OTP challenge, validating with the correct plaintext code (before expiration and within attempt limits) SHALL return success. Formally: `validate(challenge_id, original_code) == success` when `attempts < max_attempts` and `now < ttl_expires_at`.

**Validates: Requirements 2.3, 2.4**

### Property 6: OTP Single-Use

*For any* OTP challenge that has been successfully verified, subsequent validation attempts with the same code SHALL fail with status `already_verified`.

**Validates: Requirements 2.4**

### Property 7: OTP Attempt Exhaustion

*For any* OTP challenge where `attempts >= max_attempts`, all subsequent validation attempts SHALL fail regardless of code correctness, and status SHALL be `exhausted`.

**Validates: Requirements 2.5, 2.6**

### Property 8: OTP Expiration

*For any* OTP challenge where `now > ttl_expires_at`, validation SHALL fail with status `expired` regardless of code correctness.

**Validates: Requirements 2.7**

### Property 9: Rate Limit Enforcement

*For any* phone number that has exceeded the configured rate limit within the time window, subsequent OTP generation requests SHALL return HTTP 429 with `retry-after` header.

**Validates: Requirements 2.8, 6.3, 6.7**

### Property 10: Exponential Backoff Retry

*For any* transient error from a provider, the retry delay SHALL follow exponential backoff: `delay = min(2^attempt, 60)` seconds. After 5 failed attempts, the message SHALL be moved to DLQ.

**Validates: Requirements 3.2, 3.3**

### Property 11: Error Classification Determinism

*For any* provider error code, the classification as `transient` or `permanent` SHALL be deterministic and consistent. Transient errors (timeout, 429, 5xx) trigger retry; permanent errors (invalid number, blocked) do not.

**Validates: Requirements 4.2, 4.4, 4.5**

### Property 12: Circuit Breaker State Machine

*For any* provider, after `failure_threshold` consecutive failures, the circuit SHALL open and route traffic to fallback. After `recovery_timeout`, circuit SHALL transition to half-open and allow one test request.

**Validates: Requirements 4.3, 4.6**

### Property 13: Event Sourcing Completeness

*For any* status change on an SMS request, an event record SHALL be created in `sms_events` table with the new status, timestamp, and relevant payload.

**Validates: Requirements 5.4, 7.2**

### Property 14: Phone Number Masking

*For any* log entry containing a phone number, only the last 4 digits SHALL be visible. Format: `+*******1234`.

**Validates: Requirements 6.4**

### Property 15: JWT Authentication

*For any* request without a valid JWT token in the Authorization header, the SMS_Service SHALL return HTTP 401 Unauthorized.

**Validates: Requirements 6.1**

### Property 16: Webhook Signature Validation

*For any* webhook callback with an invalid or missing signature, the SMS_Service SHALL return HTTP 401 and SHALL NOT update message status.

**Validates: Requirements 5.6**

### Property 17: Trace Context Propagation

*For any* operation (HTTP request, queue message, database call), the `trace_id` and `request_id` SHALL be present in the operation context and logs.

**Validates: Requirements 5.5, 8.4**

## Error Handling

### Error Classification

| Error Type | HTTP Code | Retry | Example |
|------------|-----------|-------|---------|
| Validation Error | 400 | No | Invalid E.164, message too long |
| Authentication Error | 401 | No | Invalid/missing JWT |
| Rate Limit | 429 | Yes (after delay) | Too many requests |
| Provider Transient | 502 | Yes (backoff) | Timeout, provider 5xx |
| Provider Permanent | 422 | No | Invalid number, blocked |
| Internal Error | 500 | No | Database failure |
| Service Unavailable | 503 | Yes | Queue unavailable |

### Error Response Format

```python
class ErrorResponse(BaseModel):
    error_code: str  # e.g., "VALIDATION_ERROR", "RATE_LIMITED"
    message: str
    details: dict | None = None
    request_id: str
    trace_id: str
    retry_after: int | None = None  # seconds, for 429
```

### Provider Error Mapping

```python
TRANSIENT_ERRORS = {
    "timeout", "connection_error", "rate_limited",
    "service_unavailable", "internal_error"
}

PERMANENT_ERRORS = {
    "invalid_number", "number_blocked", "message_rejected",
    "insufficient_funds", "invalid_sender_id"
}
```

## Testing Strategy

### Dual Testing Approach

O serviço utiliza testes unitários e property-based tests de forma complementar:

- **Unit Tests**: Casos específicos, edge cases, integração com mocks
- **Property Tests**: Propriedades universais validadas com inputs gerados

### Property-Based Testing Configuration

- **Framework**: Hypothesis (Python)
- **Minimum iterations**: 100 per property
- **Tag format**: `@pytest.mark.property(feature="sms-service", property=N, validates="X.Y")`

### Test Structure

```
tests/
├── unit/
│   ├── test_sms_service.py
│   ├── test_otp_service.py
│   ├── test_provider_gateway.py
│   ├── test_rate_limiter.py
│   └── test_validators.py
├── property/
│   ├── test_idempotency_properties.py
│   ├── test_otp_properties.py
│   ├── test_validation_properties.py
│   └── test_retry_properties.py
├── integration/
│   ├── test_api_endpoints.py
│   ├── test_queue_processing.py
│   └── test_database_operations.py
└── conftest.py
```

### Key Test Scenarios

1. **Idempotency**: Same request twice → same response, single DB record
2. **OTP Lifecycle**: Generate → validate correct → success; validate again → fail
3. **Rate Limiting**: N+1 requests in window → 429 on N+1
4. **Circuit Breaker**: 5 failures → circuit open → fallback used
5. **Retry Backoff**: Verify delay sequence: 2s, 4s, 8s, 16s, 32s

## Project Structure

```
services/sms-service/
├── src/
│   ├── api/
│   │   ├── __init__.py
│   │   ├── main.py              # FastAPI app
│   │   ├── dependencies.py      # DI container
│   │   ├── middlewares/
│   │   │   ├── auth.py          # JWT validation
│   │   │   ├── tracing.py       # OpenTelemetry
│   │   │   └── rate_limit.py    # Rate limiting
│   │   └── routes/
│   │       ├── sms.py           # SMS endpoints
│   │       ├── otp.py           # OTP endpoints
│   │       └── health.py        # Health probes
│   ├── domain/
│   │   ├── __init__.py
│   │   ├── entities/
│   │   │   ├── sms.py
│   │   │   └── otp.py
│   │   └── interfaces/
│   │       ├── sms_repository.py
│   │       └── provider.py
│   ├── services/
│   │   ├── __init__.py
│   │   ├── sms_service.py
│   │   └── otp_service.py
│   ├── providers/
│   │   ├── __init__.py
│   │   ├── base.py              # Abstract provider
│   │   ├── twilio.py
│   │   ├── messagebird.py
│   │   └── gateway.py           # Provider gateway + circuit breaker
│   ├── worker/
│   │   ├── __init__.py
│   │   ├── consumer.py          # RabbitMQ consumer
│   │   └── retry_handler.py
│   ├── infrastructure/
│   │   ├── __init__.py
│   │   ├── database/
│   │   │   ├── connection.py
│   │   │   ├── models.py        # SQLAlchemy models
│   │   │   └── repositories.py
│   │   ├── queue/
│   │   │   ├── connection.py
│   │   │   └── publisher.py
│   │   └── cache/
│   │       ├── connection.py
│   │       └── rate_limiter.py
│   ├── shared/
│   │   ├── __init__.py
│   │   ├── validators.py        # E.164, etc.
│   │   ├── hashing.py           # SHA-256, bcrypt
│   │   ├── masking.py           # Phone masking
│   │   └── tracing.py           # OTel helpers
│   └── config/
│       ├── __init__.py
│       └── settings.py          # Pydantic Settings
├── tests/
│   ├── unit/
│   ├── property/
│   ├── integration/
│   └── conftest.py
├── migrations/
│   └── versions/
├── deploy/
│   └── kubernetes/
│       ├── deployment-api.yaml
│       ├── deployment-worker.yaml
│       ├── service.yaml
│       ├── hpa.yaml
│       ├── configmap.yaml
│       ├── networkpolicy.yaml
│       └── pdb.yaml
├── Dockerfile
├── Dockerfile.worker
├── pyproject.toml
├── alembic.ini
└── README.md
```

## Kubernetes Manifests Overview

### API Deployment
- Replicas: 2-10 (HPA based on CPU/memory)
- Resources: 256Mi-512Mi memory, 100m-500m CPU
- Probes: /health/live (liveness), /health/ready (readiness)
- Environment: ConfigMap + Secrets refs

### Worker Deployment
- Replicas: 2-5 (HPA based on queue lag metric)
- Resources: 256Mi-512Mi memory, 100m-500m CPU
- No HTTP probes (uses queue connection health)

### NetworkPolicy
- Ingress: Allow from API Gateway namespace only
- Egress: Allow to PostgreSQL, RabbitMQ, Redis, provider APIs

### HPA Metrics
- API: CPU > 70%, Memory > 80%
- Worker: Custom metric `rabbitmq_queue_messages_ready` > 100
