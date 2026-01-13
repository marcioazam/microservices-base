---
inclusion: always
---

# Microservices Infrastructure Pattern

## Database Strategy
- **Persistent DBs (PostgreSQL, MongoDB)**: Separate database per microservice, same instance in dev
- **Cache (Redis)**: Shared instance, namespace keys with service prefix (`auth:*`, `orders:*`)
- **Message Broker (RabbitMQ/Kafka)**: Shared instance, separate queues/exchanges per service
- **Observability (Prometheus, Elasticsearch, Jaeger)**: Always shared, centralized

## Docker Structure
```
/deploy/
  docker-compose.yml           # All services + shared infra
  docker-compose.override.yml  # Dev overrides
  /kubernetes/                 # K8s manifests
/platform|services/
  /{service-name}/
    Dockerfile                 # Service build only (no infra)
```

## Rules
- Never duplicate infrastructure (Redis, Prometheus, etc.) per service
- Centralize all docker-compose in `/deploy/`
- Service Dockerfiles contain only application build
- Shared infra in docker-compose: networks, volumes, health checks
- Use environment variables for service discovery
