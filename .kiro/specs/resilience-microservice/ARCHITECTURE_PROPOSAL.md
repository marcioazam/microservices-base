# Proposta de Reestruturação da Arquitetura - Monorepo 2025

## Visão Geral

Baseado nas melhores práticas de monorepo para 2025, proponho uma reestruturação que separa claramente os domínios de negócio dos serviços de infraestrutura, seguindo o padrão de "Domain-Driven Monorepo".

## Estrutura Atual vs Proposta

### Estrutura Atual
```
├── auth/                          # Tudo misturado em auth
│   ├── auth-edge-service/
│   ├── token-service/
│   ├── session-identity-core/
│   ├── iam-policy-service/
│   ├── mfa-service/
│   └── shared/
├── deployment/
├── docs/
├── proto/
├── sdk/
└── postman/
```

### Estrutura Proposta (2025 State-of-the-Art)
```
├── services/                      # 🔹 Microserviços de Domínio
│   ├── auth/                      # Domínio: Autenticação
│   │   ├── edge-service/          # Rust - JWT validation, mTLS
│   │   ├── token-service/         # Rust - JWT signing, DPoP
│   │   └── mfa-service/           # Elixir - TOTP, WebAuthn
│   │
│   ├── identity/                  # Domínio: Identidade
│   │   ├── session-service/       # Elixir - Sessions, OAuth 2.1
│   │   └── iam-service/           # Go - RBAC/ABAC with OPA
│   │
│   └── README.md
│
├── infra/                         # 🔹 Serviços de Infraestrutura
│   ├── resilience-service/        # Go - Circuit breaker, retry, rate limit
│   ├── observability/             # Configurações de observabilidade
│   │   ├── prometheus/
│   │   ├── grafana/
│   │   └── jaeger/
│   └── README.md
│
├── libs/                          # 🔹 Bibliotecas Compartilhadas
│   ├── go/                        # Libs Go compartilhadas
│   │   ├── audit/
│   │   ├── errors/
│   │   └── tracing/
│   ├── rust/                      # Libs Rust compartilhadas
│   │   ├── crypto/
│   │   └── grpc-common/
│   └── elixir/                    # Libs Elixir compartilhadas
│       └── event-sourcing/
│
├── api/                           # 🔹 Contratos de API
│   ├── proto/                     # Protocol Buffers
│   │   ├── auth/
│   │   ├── identity/
│   │   └── infra/
│   ├── openapi/                   # OpenAPI specs (se houver REST)
│   └── graphql/                   # GraphQL schemas (se houver)
│
├── deploy/                        # 🔹 Deployment & IaC
│   ├── kubernetes/
│   │   ├── base/                  # Kustomize base
│   │   ├── overlays/              # Kustomize overlays (dev, staging, prod)
│   │   └── helm/                  # Helm charts
│   ├── docker/
│   └── terraform/                 # Infrastructure as Code
│
├── tools/                         # 🔹 Ferramentas de Desenvolvimento
│   ├── scripts/
│   ├── generators/
│   └── linters/
│
├── docs/                          # 🔹 Documentação
│   ├── architecture/
│   │   ├── decisions/             # ADRs
│   │   └── diagrams/
│   ├── runbooks/
│   └── api/
│
├── tests/                         # 🔹 Testes E2E e de Integração
│   ├── e2e/
│   ├── integration/
│   ├── load/
│   └── contract/                  # Pact tests
│
├── sdk/                           # 🔹 SDKs para Clientes
│   ├── go/
│   ├── python/
│   └── typescript/
│
├── .github/                       # CI/CD workflows
├── .kiro/                         # Kiro specs
├── Makefile                       # Build commands
├── docker-compose.yml             # Local development
└── README.md
```

## Benefícios da Nova Estrutura

### 1. Separação Clara de Domínios
- **services/**: Microserviços de negócio organizados por domínio (auth, identity)
- **infra/**: Serviços de infraestrutura cross-cutting (resilience, observability)
- **libs/**: Código compartilhado por linguagem

### 2. Escalabilidade de Times
- Times podem ter ownership claro por domínio
- CODEOWNERS pode ser configurado por pasta
- Builds incrementais por domínio

### 3. Consistência de Contratos
- Todos os protos em `api/proto/` organizados por domínio
- Facilita geração de código para múltiplas linguagens
- Versionamento centralizado de APIs

### 4. Deploy Independente
- Cada serviço em `services/` ou `infra/` pode ser deployado independentemente
- Helm charts organizados por serviço
- Kustomize overlays para diferentes ambientes

### 5. Developer Experience
- `tools/` centraliza scripts e geradores
- `tests/` separa testes E2E dos testes unitários (que ficam com cada serviço)
- `docs/` com ADRs e runbooks organizados

## Mapeamento de Migração

| Atual | Proposto |
|-------|----------|
| `auth/auth-edge-service/` | `services/auth/edge-service/` |
| `auth/token-service/` | `services/auth/token-service/` |
| `auth/mfa-service/` | `services/auth/mfa-service/` |
| `auth/session-identity-core/` | `services/identity/session-service/` |
| `auth/iam-policy-service/` | `services/identity/iam-service/` |
| `auth/shared/` | `libs/go/` (dividir por funcionalidade) |
| `proto/` | `api/proto/` |
| `deployment/` | `deploy/` |
| (novo) | `infra/resilience-service/` |

## Estrutura Interna de um Microserviço

Cada microserviço segue a estrutura padrão da linguagem:

### Go Service (resilience-service)
```
infra/resilience-service/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── circuitbreaker/
│   ├── retry/
│   ├── ratelimit/
│   ├── bulkhead/
│   ├── policy/
│   ├── health/
│   ├── grpc/
│   └── config/
├── pkg/                           # Código exportável
├── api/                           # Protos locais (link para api/proto)
├── Dockerfile
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### Rust Service (edge-service)
```
services/auth/edge-service/
├── src/
│   ├── main.rs
│   ├── lib.rs
│   ├── circuit_breaker/
│   ├── jwt/
│   ├── mtls/
│   └── rate_limiter/
├── tests/
├── build.rs
├── Cargo.toml
├── Dockerfile
└── README.md
```

### Elixir Service (session-service)
```
services/identity/session-service/
├── lib/
│   └── session_service/
├── priv/
├── test/
├── config/
├── mix.exs
├── Dockerfile
└── README.md
```

## Próximos Passos

1. **Aprovar esta proposta** de reestruturação
2. **Atualizar o design.md** do resilience-service com o novo path
3. **Atualizar o tasks.md** com as tarefas de criação na nova estrutura
4. **Criar ADR** documentando a decisão de reestruturação

## Referências

- [Nx Monorepo Folder Structure](https://nx.dev/docs/concepts/decisions/folder-structure)
- [Aviator Monorepo Guide 2024](https://www.aviator.co/blog/monorepo-a-hands-on-guide-for-managing-repositories-and-microservices/)
- [GoReplay Microservices Best Practices 2025](https://goreplay.org/blog/best-practices-for-microservices/)
