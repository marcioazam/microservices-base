# Design Document: Resilience Service Test Reorganization

## Overview

Este documento descreve o design para reorganização dos testes do `platform/resilience-service`. O objetivo é mover todos os arquivos de teste dispersos em `internal/` para uma estrutura centralizada em `tests/`, seguindo boas práticas de organização de projetos Go.

### Current State Analysis

**Arquivos de teste identificados (22 total):**

| Tipo | Quantidade | Localização Atual |
|------|------------|-------------------|
| Property Tests | 15 | `internal/*/` |
| Benchmark Tests | 4 | `internal/*/` |
| Unit Tests | 2 | `internal/infra/metrics/` |
| Integration Tests | 1 | `internal/infra/redis/` |
| Test Utilities | 2 | `internal/testutil/` |

**Problemas identificados:**
1. Testes dispersos junto ao código fonte
2. Erros de compilação em `bulkhead_bench_test.go` (campo `Timeout` inexistente, tipo `PartitionedConfig` inexistente)
3. Falta de estrutura centralizada

## Architecture

### Target Directory Structure

```
platform/resilience-service/
├── cmd/
├── internal/
│   ├── bulkhead/
│   ├── circuitbreaker/
│   ├── config/
│   ├── domain/
│   ├── grpc/
│   ├── health/
│   ├── infra/
│   ├── policy/
│   ├── ratelimit/
│   ├── retry/
│   ├── server/
│   └── timeout/
├── pkg/
└── tests/
    ├── benchmark/
    │   ├── bulkhead_bench_test.go
    │   ├── circuitbreaker_bench_test.go
    │   ├── ratelimit_bench_test.go
    │   └── retry_bench_test.go
    ├── integration/
    │   └── redis_client_test.go
    ├── property/
    │   ├── bulkhead_prop_test.go
    │   ├── circuitbreaker_prop_test.go
    │   ├── emitter_prop_test.go
    │   ├── errors_prop_test.go
    │   ├── health_prop_test.go
    │   ├── logger_prop_test.go
    │   ├── policy_prop_test.go
    │   ├── ratelimit_prop_test.go
    │   ├── retry_handler_prop_test.go
    │   ├── retry_policy_prop_test.go
    │   ├── serialization_prop_test.go
    │   ├── shutdown_prop_test.go
    │   └── timeout_prop_test.go
    ├── unit/
    │   └── histogram_test.go
    └── testutil/
        ├── generators.go
        └── helpers.go
```

## Components and Interfaces

### Test Package Structure

Cada subpasta em `tests/` será um pacote Go separado que importa os pacotes internos:

```go
// tests/property/bulkhead_prop_test.go
package property

import (
    "github.com/auth-platform/platform/resilience-service/internal/bulkhead"
    "github.com/auth-platform/platform/resilience-service/tests/testutil"
)
```

### Test Utility Interface

```go
// tests/testutil/generators.go
package testutil

import "github.com/leanovate/gopter"

// DefaultTestParameters returns standard gopter parameters with 100+ iterations
func DefaultTestParameters() *gopter.TestParameters

// Generator functions for domain types
func GenCircuitState() gopter.Gen
func GenCircuitBreakerConfig() gopter.Gen
func GenRetryConfig() gopter.Gen
// ... etc
```

## Data Models

### File Mapping

| Original Path | New Path |
|--------------|----------|
| `internal/bulkhead/bulkhead_prop_test.go` | `tests/property/bulkhead_prop_test.go` |
| `internal/bulkhead/bulkhead_bench_test.go` | `tests/benchmark/bulkhead_bench_test.go` |
| `internal/circuitbreaker/breaker_prop_test.go` | `tests/property/circuitbreaker_prop_test.go` |
| `internal/circuitbreaker/breaker_bench_test.go` | `tests/benchmark/circuitbreaker_bench_test.go` |
| `internal/circuitbreaker/emitter_prop_test.go` | `tests/property/emitter_prop_test.go` |
| `internal/circuitbreaker/serialization_prop_test.go` | `tests/property/serialization_prop_test.go` |
| `internal/grpc/errors_prop_test.go` | `tests/property/errors_prop_test.go` |
| `internal/health/aggregator_prop_test.go` | `tests/property/health_prop_test.go` |
| `internal/infra/audit/logger_prop_test.go` | `tests/property/logger_prop_test.go` |
| `internal/infra/metrics/histogram_test.go` | `tests/unit/histogram_test.go` |
| `internal/infra/redis/client_integration_test.go` | `tests/integration/redis_client_test.go` |
| `internal/policy/engine_prop_test.go` | `tests/property/policy_prop_test.go` |
| `internal/ratelimit/ratelimit_prop_test.go` | `tests/property/ratelimit_prop_test.go` |
| `internal/ratelimit/ratelimit_bench_test.go` | `tests/benchmark/ratelimit_bench_test.go` |
| `internal/retry/handler_prop_test.go` | `tests/property/retry_handler_prop_test.go` |
| `internal/retry/policy_prop_test.go` | `tests/property/retry_policy_prop_test.go` |
| `internal/retry/handler_bench_test.go` | `tests/benchmark/retry_bench_test.go` |
| `internal/server/shutdown_prop_test.go` | `tests/property/shutdown_prop_test.go` |
| `internal/timeout/manager_prop_test.go` | `tests/property/timeout_prop_test.go` |
| `internal/testutil/generators.go` | `tests/testutil/generators.go` |
| `internal/testutil/helpers.go` | `tests/testutil/helpers.go` |

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Test File Location Correctness
*For any* test file in the project, if it belongs to the resilience-service, then it should be located under `platform/resilience-service/tests/` directory and not under `internal/`.
**Validates: Requirements 1.1**

### Property 2: Test Type Directory Mapping
*For any* test file, its location should match its type: property tests in `property/`, benchmarks in `benchmark/`, unit tests in `unit/`, integration tests in `integration/`.
**Validates: Requirements 1.2, 2.3**

### Property 3: File Line Count Constraint
*For any* test file, the total number of lines should not exceed 400 (500 with documented justification).
**Validates: Requirements 2.1**

### Property 4: Function Line Count Constraint
*For any* test function, the total number of lines should not exceed 50 (75 with documented justification).
**Validates: Requirements 2.2**

### Property 5: Property Test Documentation
*For any* property test function, it should contain a comment with feature name, property number, and requirements reference.
**Validates: Requirements 4.1, 4.2**

### Property 6: Property Test Iteration Count
*For any* property test, the test parameters should specify a minimum of 100 iterations.
**Validates: Requirements 4.3**

### Property 7: Integration Test Isolation
*For any* integration test file, it should have a `//go:build integration` tag and use `t.Skip` for unavailable services.
**Validates: Requirements 6.2, 6.3**

### Property 8: Helper Function Pattern
*For any* test helper function that accepts `*testing.T`, it should call `t.Helper()` as the first statement.
**Validates: Requirements 3.3**

## Error Handling

### Compilation Errors to Fix

**bulkhead_bench_test.go:**
1. Replace `Timeout` field with `QueueTimeout` in Config struct
2. Remove references to `NewPartitioned` and `PartitionedConfig` (não existem)
3. Change `bh.Acquire(ctx)` return type handling (returns `error`, not `bool`)

### Migration Errors

- Se um arquivo não puder ser movido, documentar o motivo
- Se imports quebrarem, atualizar para novos caminhos
- Se testes falharem após migração, investigar e corrigir

## Testing Strategy

### Dual Testing Approach

**Unit Tests:**
- Verificar comportamento específico de funções
- Testar edge cases e condições de erro
- Localização: `tests/unit/`

**Property-Based Tests:**
- Usar `gopter` como biblioteca PBT
- Mínimo de 100 iterações por propriedade
- Cada teste deve referenciar a propriedade do design
- Formato de anotação: `**Feature: resilience-test-reorganization, Property N: Description**`
- Localização: `tests/property/`

**Benchmark Tests:**
- Medir performance de operações críticas
- Localização: `tests/benchmark/`

**Integration Tests:**
- Build tag: `//go:build integration`
- Skip graceful quando dependências indisponíveis
- Localização: `tests/integration/`

### Validation Steps

1. `go build ./...` - Verificar compilação
2. `go test ./tests/...` - Executar todos os testes
3. `go test -bench=. ./tests/benchmark/...` - Executar benchmarks
4. `go test -tags=integration ./tests/integration/...` - Executar integração (com Redis)

