# Monorepo Architecture - State of the Art 2025

## Pesquisa e Referências

Baseado em pesquisa de melhores práticas de monorepo para microserviços em 2025:
- NX Monorepo (nx.dev)
- AWS PDK Monorepo
- Graphite Engineering Blog
- Luca Pette - How to Structure a Monorepo
- golang-standards/project-layout

## Estrutura Recomendada

```
auth-platform/
├── .github/                    # GitHub Actions workflows
│   └── workflows/
├── .kiro/                      # Kiro IDE specs
│   └── specs/
├── api/                        # API contracts (protos, OpenAPI)
│   └── proto/
│       ├── auth/               # Auth domain protos
│       └── infra/              # Infrastructure protos
├── deploy/                     # Deployment configurations
│   ├── docker/                 # Dockerfiles por serviço
│   └── kubernetes/
│       ├── gateway/            # API Gateway configs
│       ├── helm/               # Helm charts
│       └── vault-bootstrap/    # Vault setup scripts
├── docs/                       # Documentation
│   ├── adr/                    # Architecture Decision Records
│   ├── api/                    # API documentation
│   └── runbooks/               # Operational runbooks
├── libs/                       # Shared libraries (NEW)
│   ├── go/                     # Go shared libs
│   │   ├── audit/              # Audit logging
│   │   ├── caep/               # CAEP events
│   │   ├── errors/             # Error handling
│   │   ├── tracing/            # OpenTelemetry
│   │   └── vault/              # Vault client
│   ├── rust/                   # Rust shared libs
│   └── elixir/                 # Elixir shared libs
├── platform/                   # Platform/Infrastructure services (NEW)
│   └── resilience-service/     # Resilience microservice
│       ├── cmd/
│       ├── internal/
│       ├── go.mod
│       └── Dockerfile
├── services/                   # Domain microservices (NEW)
│   ├── auth-edge/              # Auth Edge Service (Rust)
│   │   ├── src/
│   │   ├── Cargo.toml
│   │   └── Dockerfile
│   ├── iam-policy/             # IAM Policy Service (Go)
│   │   ├── cmd/
│   │   ├── internal/
│   │   └── go.mod
│   ├── mfa/                    # MFA Service (Elixir)
│   │   ├── lib/
│   │   └── mix.exs
│   ├── session-identity/       # Session Identity (Elixir)
│   │   ├── lib/
│   │   └── mix.exs
│   └── token/                  # Token Service (Rust)
│       ├── src/
│       └── Cargo.toml
├── sdk/                        # Client SDKs
│   ├── go/
│   ├── python/
│   └── typescript/
├── tools/                      # Build tools and scripts (NEW)
│   ├── scripts/
│   └── generators/
├── Makefile                    # Root Makefile
└── README.md
```

## Mudanças Propostas

### Atual → Proposto

| Atual | Proposto | Razão |
|-------|----------|-------|
| `auth/` | `services/` | Separação clara entre serviços de domínio |
| `auth/shared/` | `libs/` | Bibliotecas compartilhadas em pasta dedicada |
| `infra/` | `platform/` | Nomenclatura padrão para serviços de plataforma |
| `proto/` (root) | `api/proto/` | Centralização de contratos de API |

### Benefícios

1. **Separação Clara de Responsabilidades**
   - `services/` - Microserviços de domínio (negócio)
   - `platform/` - Serviços de infraestrutura (resilience, observability)
   - `libs/` - Código compartilhado entre serviços

2. **Escalabilidade**
   - Fácil adicionar novos serviços
   - Bibliotecas reutilizáveis
   - Contratos centralizados

3. **Developer Experience**
   - Estrutura intuitiva
   - Fácil navegação
   - Padrão da indústria (NX, Bazel, Turborepo)

## Estrutura Detalhada por Serviço

### Go Service (Standard Layout)
```
services/iam-policy/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── domain/           # Domain models
│   ├── handler/          # HTTP/gRPC handlers
│   ├── repository/       # Data access
│   └── service/          # Business logic
├── pkg/                  # Public packages (if any)
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### Rust Service
```
services/auth-edge/
├── src/
│   ├── main.rs
│   ├── lib.rs
│   ├── handlers/
│   ├── domain/
│   └── infra/
├── tests/
├── Cargo.toml
├── Cargo.lock
└── README.md
```

### Elixir Service
```
services/mfa/
├── lib/
│   └── mfa/
│       ├── application.ex
│       ├── domain/
│       └── handlers/
├── priv/
├── test/
├── mix.exs
└── README.md
```

## Plano de Migração

### Fase 1: Criar nova estrutura
```bash
mkdir -p services platform libs tools
```

### Fase 2: Mover serviços de domínio
```bash
# auth/* → services/*
mv auth/auth-edge-service services/auth-edge
mv auth/iam-policy-service services/iam-policy
mv auth/mfa-service services/mfa
mv auth/session-identity-core services/session-identity
mv auth/token-service services/token
```

### Fase 3: Mover bibliotecas compartilhadas
```bash
# auth/shared/* → libs/go/*
mv auth/shared/* libs/go/
```

### Fase 4: Mover serviços de plataforma
```bash
# infra/* → platform/*
mv infra/resilience-service platform/resilience-service
```

### Fase 5: Consolidar protos
```bash
# proto/* → api/proto/auth/*
mv proto/* api/proto/auth/
```

### Fase 6: Atualizar imports e referências
- Atualizar go.mod paths
- Atualizar Cargo.toml paths
- Atualizar mix.exs paths
- Atualizar Dockerfiles
- Atualizar CI/CD workflows

## Referências

- [NX Folder Structure](https://nx.dev/docs/concepts/decisions/folder-structure)
- [golang-standards/project-layout](https://github.com/golang-standards/project-layout)
- [How to Structure a Monorepo - Luca Pette](https://lucapette.me/writing/how-to-structure-a-monorepo/)
- [AWS PDK Monorepo](https://aws.github.io/aws-pdk/developer_guides/monorepo/index.html)
- [Graphite - How we organize our monorepo](https://graphite.com/blog/how-we-organize-our-monorepo-to-ship-fast)
