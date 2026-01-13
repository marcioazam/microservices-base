# Logging Microservice

Microserviço de logging centralizado em C# (.NET 9) para a plataforma de autenticação.

## Tecnologias (Dezembro 2025)

- **.NET 9.0** com C# 13
- **Elastic.Clients.Elasticsearch 8.17.0** (substituiu NEST)
- **RabbitMQ.Client 7.1.0** (async-first API)
- **OpenTelemetry 1.10.0** (distributed tracing)
- **xUnit 3.x** + **FsCheck 3.x** (property-based testing)
- **Testcontainers 4.x** (integration tests)

## Arquitetura

- **API Layer**: gRPC + REST para ingestão e consulta de logs
- **Processing Layer**: Processamento assíncrono via RabbitMQ
- **Storage Layer**: ElasticSearch para armazenamento indexado
- **Observability**: Prometheus metrics, OpenTelemetry tracing

## Resiliência

Este serviço é protegido pelo Service Mesh (Linkerd) via `ResiliencePolicy`:

- **gRPC (porta 5001)**: Circuit breaker com threshold alto (10), sem retry (fire-and-forget), timeout 2s
- **HTTP (porta 5000)**: Circuit breaker, retry para 503/504, timeout 5s

Ver `deploy/kubernetes/service-mesh/logging-service/resilience-policy.yaml` para configuração.

## Estrutura do Projeto

```
platform/logging-service/
├── src/
│   ├── LoggingService.Api/           # API Layer (gRPC + REST)
│   ├── LoggingService.Core/          # Domain models e interfaces
│   ├── LoggingService.Infrastructure/ # ElasticSearch, Queue implementations
│   └── LoggingService.Worker/        # Background processing worker
├── tests/
│   ├── LoggingService.Unit.Tests/
│   ├── LoggingService.Integration.Tests/
│   └── LoggingService.Property.Tests/
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## Executando Localmente

```bash
# Subir infraestrutura
docker-compose up -d elasticsearch rabbitmq

# Executar API
cd src/LoggingService.Api
dotnet run

# Executar Worker
cd src/LoggingService.Worker
dotnet run
```

## Executando com Docker

```bash
docker-compose up -d
```

## Endpoints

- **REST API**: http://localhost:5000/api/v1/logs
- **gRPC**: localhost:5001
- **Metrics**: http://localhost:5000/metrics
- **Health**: http://localhost:5000/health/live
- **Kibana**: http://localhost:5601
- **Grafana**: http://localhost:3000
- **RabbitMQ Management**: http://localhost:15672

## API Reference

### REST API

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/logs | Ingest single log entry |
| POST | /api/v1/logs/batch | Ingest batch of logs (max 1000) |
| GET | /api/v1/logs | Query logs with filters |
| GET | /api/v1/logs/{id} | Get specific log entry |
| GET | /api/v1/logs/export | Export logs as JSON/CSV |

### gRPC API

Package: `logging.v1`

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| IngestLog | IngestLogRequest | IngestLogResponse | Ingest single log entry |
| IngestLogBatch | IngestLogBatchRequest | IngestLogBatchResponse | Ingest batch of log entries |
| QueryLogs | QueryLogsRequest | QueryLogsResponse | Query logs with filters |
| StreamLogs | StreamLogsRequest | stream LogEntryMessage | Stream logs in real-time |

#### Log Levels

| Level | Value | Description |
|-------|-------|-------------|
| DEBUG | 1 | Debug information |
| INFO | 2 | Informational messages |
| WARN | 3 | Warning conditions |
| ERROR | 4 | Error conditions |
| FATAL | 5 | Fatal/critical errors |

Proto file location: `src/LoggingService.Api/Protos/logging.proto`

## Testes

```bash
# Unit tests
dotnet test tests/LoggingService.Unit.Tests

# Property tests
dotnet test tests/LoggingService.Property.Tests

# Integration tests (requer Docker)
dotnet test tests/LoggingService.Integration.Tests
```
