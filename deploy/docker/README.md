# Centralized Docker Infrastructure

This directory contains the centralized Docker Compose configuration for all microservices.

## Architecture

```
deploy/docker/
├── docker-compose.yml          # Main compose file with all services
├── docker-compose.override.yml # Development overrides
├── init-scripts/
│   └── postgres/               # PostgreSQL initialization (creates DBs per service)
└── observability/
    ├── prometheus.yml          # Centralized metrics collection
    ├── otel-collector-config.yaml
    └── grafana/provisioning/   # Grafana dashboards
```

## Shared Infrastructure

| Component      | Container Name        | Port(s)       | Purpose                    |
|----------------|----------------------|---------------|----------------------------|
| PostgreSQL     | platform-postgres    | 5432          | Persistent data (1 DB/service) |
| Redis          | platform-redis       | 6379          | Cache, sessions (namespaced) |
| Elasticsearch  | platform-elasticsearch| 9200, 9300   | Logs storage               |
| RabbitMQ       | platform-rabbitmq    | 5672, 15672   | Message broker             |
| Prometheus     | platform-prometheus  | 9099          | Metrics collection         |
| Grafana        | platform-grafana     | 3000          | Dashboards                 |
| Jaeger         | platform-jaeger      | 16686         | Distributed tracing        |
| OTel Collector | platform-otel-collector| 4317, 4318  | Telemetry aggregation      |

## Database Strategy

- **PostgreSQL**: One instance, separate database per service
  - `auth_db`, `session_db`, `mfa_db`, `iam_db`
  - `logging_db`, `resilience_db`

- **Redis**: Shared instance with namespace prefixes
  - `auth:*`, `session:*`, `logging:*`, `resilience:*`

## Usage

```bash
# Start all services
docker-compose up -d

# Start with development overrides
docker-compose -f docker-compose.yml -f docker-compose.override.yml up -d

# Start specific services
docker-compose up -d logging-api resilience-service

# View logs
docker-compose logs -f logging-api

# Stop all
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

## Service Ports

| Service              | HTTP/gRPC | Metrics |
|---------------------|-----------|---------|
| logging-api         | 5000/5001 | 5000    |
| resilience-service  | 50056     | -       |
| auth-edge-service   | 8080      | 9090    |
| token-service       | 8081      | 9091    |
| session-identity-core| 4000/8082| 9092    |
| iam-policy-service  | 8083      | 9093    |
| mfa-service         | 8084      | 9094    |

## Environment Variables

Services use environment variables for configuration. Key patterns:

- `*_REDIS_URL`: Redis connection (shared instance)
- `*_DATABASE_URL`: PostgreSQL connection (service-specific DB)
- `OTEL_EXPORTER_OTLP_ENDPOINT`: OpenTelemetry collector
