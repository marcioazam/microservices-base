# Implementation Plan: Logging Microservice

## Overview

Este plano implementa um microserviço de logging centralizado em C# (.NET 8) seguindo a arquitetura definida no design. A implementação segue uma abordagem incremental, começando pela estrutura do projeto e modelos de domínio, seguido pelos componentes de ingestão, processamento assíncrono, armazenamento e APIs de consulta.

## Tasks

- [x] 1. Setup do Projeto e Estrutura Base
  - [x] 1.1 Criar estrutura de diretórios do projeto em `platform/logging-service/`
    - Criar solution file e projetos: Api, Core, Infrastructure, Worker
    - Configurar referências entre projetos
    - _Requirements: 10.5_

  - [x] 1.2 Configurar dependências e packages NuGet
    - Adicionar: NEST (ElasticSearch), RabbitMQ.Client, Grpc.AspNetCore, prometheus-net
    - Adicionar: OpenTelemetry, FluentValidation, Serilog
    - _Requirements: 7.5_

  - [x] 1.3 Criar arquivos de configuração base
    - appsettings.json com estrutura de configuração
    - Dockerfile e docker-compose.yml
    - _Requirements: 10.1_

- [x] 2. Implementar Modelos de Domínio (Core)
  - [x] 2.1 Criar LogEntry record e tipos relacionados
    - Implementar LogEntry, LogLevel enum, ExceptionInfo
    - Implementar ValidationResult, FieldError
    - _Requirements: 2.1, 2.3_

  - [x] 2.2 Escrever property test para estrutura de LogEntry
    - **Property 3: Log Entry Structure Completeness**
    - **Validates: Requirements 2.1, 2.3**

  - [x] 2.3 Criar interfaces de serviço (ILogIngestionService, ILogRepository, ILogQueue, etc.)
    - Definir contratos para todos os componentes
    - _Requirements: 1.1, 3.1, 4.1_

  - [x] 2.4 Implementar LogQuery e PagedResult models
    - Criar modelos de consulta e paginação
    - _Requirements: 8.1, 8.2, 8.3_

- [x] 3. Implementar Validação e Enriquecimento
  - [x] 3.1 Implementar LogEntryValidator
    - Validar campos obrigatórios: timestamp, service_id, level, message
    - Retornar erros específicos por campo
    - _Requirements: 1.3, 1.4_

  - [x] 3.2 Escrever property test para validação de entrada
    - **Property 1: Input Validation Rejects Invalid Entries**
    - **Validates: Requirements 1.3, 1.4**

  - [x] 3.3 Implementar LogEntryEnricher
    - Gerar correlation_id quando ausente (UUID v4)
    - Normalizar timestamp para UTC
    - _Requirements: 2.2_

  - [x] 3.4 Escrever property test para geração de correlation_id
    - **Property 4: Correlation ID Generation**
    - **Validates: Requirements 2.2**

- [x] 4. Checkpoint - Validar Modelos e Validação
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Implementar Serialização JSON
  - [x] 5.1 Configurar JsonSerializerOptions para LogEntry
    - Configurar naming policy, converters para DateTimeOffset e enums
    - _Requirements: 2.4_

  - [x] 5.2 Escrever property test para round-trip de serialização
    - **Property 5: Serialization Round-Trip**
    - **Validates: Requirements 2.4, 2.5**

- [x] 6. Implementar Segurança e PII Masking
  - [x] 6.1 Implementar PiiMasker
    - Detectar padrões: email, phone, IP
    - Aplicar masking/redaction conforme configuração
    - _Requirements: 6.3_

  - [x] 6.2 Escrever property test para masking de PII
    - **Property 10: PII Masking Completeness**
    - **Validates: Requirements 6.3**

  - [x] 6.3 Implementar SecurityOptions e configuração
    - Configurar TLS, encryption at rest, PII patterns
    - _Requirements: 6.1, 6.2_

- [x] 7. Implementar Queue (RabbitMQ)
  - [x] 7.1 Implementar RabbitMqLogQueue
    - Enqueue/Dequeue de LogEntry
    - Batch operations
    - _Requirements: 3.1_

  - [x] 7.2 Implementar monitoramento de queue depth
    - Emitir warning quando queue atinge 80% capacidade
    - Aplicar backpressure quando queue cheia
    - _Requirements: 3.3, 3.4_

- [x] 8. Implementar ElasticSearch Repository
  - [x] 8.1 Implementar ElasticSearchLogRepository
    - SaveAsync e SaveBatchAsync
    - Index naming: logs-{service_id}-{yyyy.MM.dd}
    - _Requirements: 4.1, 4.2_

  - [x] 8.2 Escrever property test para naming de índices
    - **Property 7: Index Naming Convention**
    - **Validates: Requirements 4.1, 4.2**

  - [x] 8.3 Implementar QueryAsync com filtros
    - Filtros: time range, service_id, log_level, correlation_id, search text
    - Paginação e sorting
    - _Requirements: 4.3, 4.4, 4.5_

  - [x] 8.4 Escrever property test para filtros de query
    - **Property 8: Query Filter Correctness**
    - **Validates: Requirements 4.4, 4.5**

- [x] 9. Checkpoint - Validar Infraestrutura
  - Ensure all tests pass, ask the user if questions arise.

- [x] 10. Implementar Serviço de Ingestão
  - [x] 10.1 Implementar LogIngestionService
    - Validar, enriquecer, mascarar PII, enfileirar
    - Suportar single e batch ingestion
    - _Requirements: 1.1, 1.2, 1.5_

  - [x] 10.2 Escrever property test para batch size
    - **Property 2: Batch Size Enforcement**
    - **Validates: Requirements 1.5**

- [x] 11. Implementar Worker de Processamento
  - [x] 11.1 Implementar LogProcessorWorker
    - Background service para processar queue
    - Batch processing com retry
    - _Requirements: 3.2, 3.5_

  - [x] 11.2 Escrever property test para garantia de entrega
    - **Property 6: Queue Delivery Guarantee**
    - **Validates: Requirements 3.1, 3.5**

- [x] 12. Implementar APIs REST
  - [x] 12.1 Implementar LogsController
    - POST /api/v1/logs - single ingestion
    - POST /api/v1/logs/batch - batch ingestion
    - GET /api/v1/logs - query com filtros
    - GET /api/v1/logs/{id} - get by id
    - _Requirements: 1.1, 1.2, 8.1_

  - [x] 12.2 Implementar export endpoints
    - GET /api/v1/logs/export - JSON e CSV
    - _Requirements: 8.5_

  - [x] 12.3 Escrever property test para paginação
    - **Property 15: Query Pagination Correctness**
    - **Validates: Requirements 8.1, 8.2, 8.3**

  - [x] 12.4 Escrever property test para formato de export
    - **Property 16: Export Format Validity**
    - **Validates: Requirements 8.5**

- [x] 13. Implementar gRPC Service
  - [x] 13.1 Criar proto files para LoggingService
    - IngestLog, IngestLogBatch, QueryLogs, StreamLogs
    - _Requirements: 1.1_

  - [x] 13.2 Implementar LoggingGrpcService
    - Implementar todos os métodos gRPC
    - _Requirements: 1.1_

- [x] 14. Checkpoint - Validar APIs
  - Ensure all tests pass, ask the user if questions arise.

- [x] 15. Implementar Observabilidade
  - [x] 15.1 Implementar Prometheus Metrics
    - logs_received_total, logs_processed_total, logs_failed_total
    - queue_depth, processing_latency_seconds
    - _Requirements: 7.2_

  - [x] 15.2 Escrever property test para accuracy de métricas
    - **Property 13: Metrics Accuracy**
    - **Validates: Requirements 7.2**

  - [x] 15.3 Implementar Health Checks
    - /health/live e /health/ready
    - Verificar ElasticSearch e Queue
    - _Requirements: 7.3_

  - [x] 15.4 Configurar OpenTelemetry
    - Distributed tracing com W3C Trace Context
    - _Requirements: 7.5_

  - [x] 15.5 Escrever property test para propagação de trace context
    - **Property 14: Trace Context Propagation**
    - **Validates: Requirements 7.5**

- [x] 16. Implementar Políticas de Retenção
  - [x] 16.1 Implementar RetentionService
    - Aplicar políticas por log level
    - Archive to cold storage antes de delete
    - _Requirements: 5.1, 5.2, 5.3_

  - [x] 16.2 Escrever property test para enforcement de retenção
    - **Property 9: Retention Policy Enforcement**
    - **Validates: Requirements 5.2, 5.3**

- [x] 17. Implementar Auditoria
  - [x] 17.1 Implementar AuditLogService
    - Registrar todas as operações de query
    - _Requirements: 6.5_

  - [x] 17.2 Escrever property test para completude de audit trail
    - **Property 12: Audit Trail Completeness**
    - **Validates: Requirements 6.5**

- [x] 18. Configurar Kubernetes e Deployment
  - [x] 18.1 Criar manifests Kubernetes
    - Deployment, Service, ConfigMap, Secrets
    - HorizontalPodAutoscaler
    - _Requirements: 10.1, 10.2_

  - [x] 18.2 Configurar ILM Policy no ElasticSearch
    - Hot, warm, cold, delete phases
    - _Requirements: 5.1, 5.4_

- [x] 19. Checkpoint Final - Validação Completa
  - All implementation tasks completed.

## Notes

- Todas as tasks são obrigatórias para garantir cobertura completa desde o início
- Cada task referencia requisitos específicos para rastreabilidade
- Checkpoints garantem validação incremental
- Property tests validam propriedades universais de corretude
- Unit tests validam exemplos específicos e edge cases
- O serviço será posicionado em `platform/logging-service/` seguindo ADR-001
