# Requirements Document

## Introduction

O `sms-service` é um microserviço responsável pelo envio de SMS transacionais e OTP (One-Time Password), projetado para alta confiabilidade, rastreabilidade completa, idempotência end-to-end e integração com múltiplos provedores externos. O serviço é stateless, cloud-native e otimizado para deploy em Kubernetes.

## Glossary

- **SMS_Service**: Microserviço principal responsável por receber requisições de envio de SMS
- **SMS_Worker**: Componente assíncrono que processa a fila e realiza o envio efetivo aos provedores
- **OTP_Manager**: Componente responsável por geração, armazenamento seguro e validação de códigos OTP
- **Provider_Gateway**: Camada de abstração para comunicação com provedores externos (Twilio, MessageBird, etc.)
- **Idempotency_Store**: Mecanismo de persistência para garantir idempotência de requisições
- **Message_Queue**: Sistema de filas para processamento assíncrono (RabbitMQ/Kafka/Redis Streams)
- **DLQ**: Dead Letter Queue para mensagens que falharam após todas as tentativas
- **E.164**: Formato internacional de números de telefone (+[código país][número])
- **Transient_Error**: Erro temporário que permite retry (timeout, rate limit do provedor)
- **Permanent_Error**: Erro definitivo que não permite retry (número inválido, mensagem rejeitada)

## Requirements

### Requirement 1: Envio de SMS Transacional

**User Story:** As a client service, I want to send transactional SMS messages, so that I can communicate with users via text message reliably.

#### Acceptance Criteria

1. WHEN a client sends a valid SMS request with `to` (E.164), `message`, and `idempotency_key`, THE SMS_Service SHALL accept the request and return a `request_id`, `status`, and timestamps immediately
2. WHEN a client sends an SMS request with an existing `idempotency_key`, THE SMS_Service SHALL return the original response without creating a duplicate request
3. WHEN a client sends an SMS request with optional `sender_id` and `metadata`, THE SMS_Service SHALL store and propagate these fields through the processing pipeline
4. WHEN an SMS request is accepted, THE SMS_Service SHALL persist the request to the database with status `accepted` before returning the response
5. WHEN an SMS request is accepted, THE SMS_Service SHALL enqueue the message for asynchronous processing by the SMS_Worker
6. IF a client sends an invalid E.164 phone number, THEN THE SMS_Service SHALL reject the request with a validation error and HTTP 400
7. IF a client sends a message exceeding the maximum allowed length, THEN THE SMS_Service SHALL reject the request with a validation error

### Requirement 2: Geração e Validação de OTP

**User Story:** As a client service, I want to generate and validate OTP codes via SMS, so that I can implement secure two-factor authentication.

#### Acceptance Criteria

1. WHEN a client requests OTP generation for a phone number, THE OTP_Manager SHALL generate a cryptographically secure code, hash it, and store only the hash
2. WHEN an OTP is generated, THE OTP_Manager SHALL send the plaintext OTP via SMS and return a `challenge_id` to the client
3. WHEN a client validates an OTP, THE OTP_Manager SHALL compare the provided code against the stored hash
4. WHEN an OTP validation succeeds, THE OTP_Manager SHALL mark the challenge as `verified` and prevent reuse
5. WHEN an OTP validation fails, THE OTP_Manager SHALL increment the attempt counter and return failure
6. IF the maximum validation attempts are exceeded, THEN THE OTP_Manager SHALL mark the challenge as `exhausted` and reject further attempts
7. IF the OTP TTL has expired, THEN THE OTP_Manager SHALL reject the validation with `expired` status
8. WHILE rate limits are active for a phone number, THE OTP_Manager SHALL reject new OTP generation requests with HTTP 429
9. THE OTP_Manager SHALL enforce configurable TTL (default 5 minutes) and max attempts (default 3)

### Requirement 3: Processamento Assíncrono e Filas

**User Story:** As a system operator, I want SMS sending to be processed asynchronously, so that the API remains responsive and failures can be retried.

#### Acceptance Criteria

1. WHEN an SMS request is enqueued, THE SMS_Worker SHALL consume it from the message queue and attempt delivery via Provider_Gateway
2. WHEN a provider returns a transient error, THE SMS_Worker SHALL retry with exponential backoff (base 2s, max 5 retries)
3. WHEN all retry attempts are exhausted, THE SMS_Worker SHALL move the message to the DLQ and update status to `failed`
4. WHEN processing a message, THE SMS_Worker SHALL verify idempotency to prevent duplicate sends on retry
5. WHEN a message is successfully sent, THE SMS_Worker SHALL update the database status to `sent` with provider response details
6. THE SMS_Worker SHALL process messages in order per phone number to prevent race conditions
7. IF the message queue is unavailable, THEN THE SMS_Service SHALL return HTTP 503 and not accept new requests

### Requirement 4: Abstração de Provedores SMS

**User Story:** As a system operator, I want to use multiple SMS providers with automatic fallback, so that I can ensure high availability of SMS delivery.

#### Acceptance Criteria

1. THE Provider_Gateway SHALL implement a common interface for all SMS providers (Twilio, MessageBird, etc.)
2. WHEN the primary provider fails with a transient error, THE Provider_Gateway SHALL retry on the same provider according to retry policy
3. WHEN the primary provider is unavailable (circuit open), THE Provider_Gateway SHALL route to the fallback provider
4. WHEN a provider returns a permanent error, THE Provider_Gateway SHALL NOT retry and SHALL mark the request as `failed`
5. THE Provider_Gateway SHALL classify provider errors as transient or permanent based on error codes
6. THE Provider_Gateway SHALL track provider health metrics and implement circuit breaker pattern
7. WHEN switching providers, THE Provider_Gateway SHALL log the fallback event with reason and provider details

### Requirement 5: Status e Rastreabilidade

**User Story:** As a client service, I want to query SMS delivery status and receive delivery callbacks, so that I can track message delivery.

#### Acceptance Criteria

1. WHEN a client queries status by `request_id`, THE SMS_Service SHALL return current status, timestamps, and event history
2. THE SMS_Service SHALL support the following status values: `accepted`, `queued`, `sending`, `sent`, `delivered`, `failed`, `expired`
3. WHEN a provider sends a delivery callback (webhook), THE SMS_Service SHALL validate the signature and update the message status
4. WHEN a status change occurs, THE SMS_Service SHALL create an event record in `sms_events` table
5. THE SMS_Service SHALL propagate `trace_id` and `request_id` through all operations for distributed tracing
6. IF a webhook signature is invalid, THEN THE SMS_Service SHALL reject the callback with HTTP 401

### Requirement 6: Segurança e Proteção Anti-Abuso

**User Story:** As a security engineer, I want the SMS service to be secure and protected against abuse, so that the system remains reliable and compliant.

#### Acceptance Criteria

1. THE SMS_Service SHALL authenticate all incoming requests via JWT tokens with service identity claims
2. THE SMS_Service SHALL validate all input fields (E.164 format, message length, charset)
3. THE SMS_Service SHALL implement rate limiting per client service and per destination phone number
4. THE SMS_Service SHALL mask phone numbers in all logs (show only last 4 digits)
5. THE SMS_Service SHALL store message content as hash only, never in plaintext
6. THE SMS_Service SHALL load secrets (API keys, JWT secrets) from Kubernetes Secrets or External Secrets Operator
7. IF rate limits are exceeded, THEN THE SMS_Service SHALL return HTTP 429 with retry-after header
8. THE SMS_Service SHALL support optional allowlist/denylist for phone numbers

### Requirement 7: Persistência e Modelo de Dados

**User Story:** As a system operator, I want all SMS requests and events persisted, so that I can audit and troubleshoot delivery issues.

#### Acceptance Criteria

1. THE SMS_Service SHALL persist all requests in `sms_requests` table with idempotency_key, status, provider, timestamps
2. THE SMS_Service SHALL persist all status changes in `sms_events` table with event_type and payload
3. THE SMS_Service SHALL persist OTP challenges in `otp_challenges` table with hashed OTP, TTL, and attempt tracking
4. THE SMS_Service SHALL enforce unique constraint on `idempotency_key` in `sms_requests`
5. THE SMS_Service SHALL create indexes on `(to, created_at)` and `(status)` for efficient queries
6. WHEN persisting sensitive data, THE SMS_Service SHALL store only hashes (message_hash, otp_hash)

### Requirement 8: Observabilidade

**User Story:** As a system operator, I want comprehensive observability, so that I can monitor, alert, and troubleshoot the SMS service.

#### Acceptance Criteria

1. THE SMS_Service SHALL emit OpenTelemetry traces for all HTTP requests, queue operations, and database calls
2. THE SMS_Service SHALL expose Prometheus metrics: `sms_sent_total`, `sms_failed_total`, `provider_latency_ms`, `otp_verifications_total`, `queue_lag`
3. THE SMS_Service SHALL emit structured JSON logs with fields: `request_id`, `idempotency_key`, `provider`, `to_masked`, `status`, `error_code`
4. THE SMS_Service SHALL propagate W3C Trace Context headers for distributed tracing
5. THE SMS_Service SHALL expose `/health/live` and `/health/ready` endpoints for Kubernetes probes
6. WHEN an error occurs, THE SMS_Service SHALL log with appropriate level (ERROR for failures, WARN for retries)

### Requirement 9: Deploy e Infraestrutura Kubernetes

**User Story:** As a DevOps engineer, I want production-ready Kubernetes manifests, so that I can deploy the SMS service reliably.

#### Acceptance Criteria

1. THE deployment SHALL include separate Deployments for API and Worker components
2. THE deployment SHALL include HorizontalPodAutoscaler for both API and Worker
3. THE deployment SHALL use ConfigMaps for non-sensitive configuration and Secrets for credentials
4. THE deployment SHALL include NetworkPolicy restricting ingress/egress to required services only
5. THE deployment SHALL include Service resource for API component
6. THE deployment SHALL configure resource requests and limits for all containers
7. THE deployment SHALL include PodDisruptionBudget for high availability
