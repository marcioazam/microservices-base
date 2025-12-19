# Go Shared Libraries

A collection of reusable Go packages organized by domain.

## Categories

| Category | Description |
|----------|-------------|
| [collections](./collections/) | Data structures (maps, sets, queues, slices) |
| [concurrency](./concurrency/) | Concurrency primitives (async, pools, channels) |
| [functional](./functional/) | Functional types (option, result, either, stream) |
| [optics](./optics/) | Functional optics (lens, prism) |
| [patterns](./patterns/) | Design patterns (registry, spec) |
| [events](./events/) | Event handling (eventbus, pubsub) |
| [resilience](./resilience/) | Fault tolerance (circuit breaker, retry, rate limit) |
| [server](./server/) | Server utilities (health, shutdown, tracing) |
| [grpc](./grpc/) | gRPC utilities |
| [utils](./utils/) | General utilities (uuid, codec, validation) |
| [testing](./testing/) | Test utilities |

## Quick Start

```go
import (
    "github.com/auth-platform/libs/go/collections/slices"
    "github.com/auth-platform/libs/go/functional/option"
    "github.com/auth-platform/libs/go/concurrency/async"
)
```

## Migration Guide

If you're migrating from the old flat structure, update your imports:

| Old Import | New Import |
|------------|------------|
| `libs/go/slices` | `github.com/auth-platform/libs/go/collections/slices` |
| `libs/go/maps` | `github.com/auth-platform/libs/go/collections/maps` |
| `libs/go/set` | `github.com/auth-platform/libs/go/collections/set` |
| `libs/go/queue` | `github.com/auth-platform/libs/go/collections/queue` |
| `libs/go/pqueue` | `github.com/auth-platform/libs/go/collections/pqueue` |
| `libs/go/lru` | `github.com/auth-platform/libs/go/collections/lru` |
| `libs/go/sort` | `github.com/auth-platform/libs/go/collections/sort` |
| `libs/go/async` | `github.com/auth-platform/libs/go/concurrency/async` |
| `libs/go/atomic` | `github.com/auth-platform/libs/go/concurrency/atomic` |
| `libs/go/channels` | `github.com/auth-platform/libs/go/concurrency/channels` |
| `libs/go/errgroup` | `github.com/auth-platform/libs/go/concurrency/errgroup` |
| `libs/go/once` | `github.com/auth-platform/libs/go/concurrency/once` |
| `libs/go/pool` | `github.com/auth-platform/libs/go/concurrency/pool` |
| `libs/go/syncmap` | `github.com/auth-platform/libs/go/concurrency/syncmap` |
| `libs/go/waitgroup` | `github.com/auth-platform/libs/go/concurrency/waitgroup` |
| `libs/go/option` | `github.com/auth-platform/libs/go/functional/option` |
| `libs/go/result` | `github.com/auth-platform/libs/go/functional/result` |
| `libs/go/either` | `github.com/auth-platform/libs/go/functional/either` |
| `libs/go/iterator` | `github.com/auth-platform/libs/go/functional/iterator` |
| `libs/go/lazy` | `github.com/auth-platform/libs/go/functional/lazy` |
| `libs/go/pipeline` | `github.com/auth-platform/libs/go/functional/pipeline` |
| `libs/go/stream` | `github.com/auth-platform/libs/go/functional/stream` |
| `libs/go/tuple` | `github.com/auth-platform/libs/go/functional/tuple` |
| `libs/go/lens` | `github.com/auth-platform/libs/go/optics/lens` |
| `libs/go/prism` | `github.com/auth-platform/libs/go/optics/prism` |
| `libs/go/registry` | `github.com/auth-platform/libs/go/patterns/registry` |
| `libs/go/spec` | `github.com/auth-platform/libs/go/patterns/spec` |
| `libs/go/events` | `github.com/auth-platform/libs/go/events/builder` |
| `libs/go/eventbus` | `github.com/auth-platform/libs/go/events/eventbus` |
| `libs/go/pubsub` | `github.com/auth-platform/libs/go/events/pubsub` |
| `libs/go/health` | `github.com/auth-platform/libs/go/server/health` |
| `libs/go/tracing` | `github.com/auth-platform/libs/go/server/tracing` |
| `libs/go/audit` | `github.com/auth-platform/libs/go/utils/audit` |
| `libs/go/cache` | `github.com/auth-platform/libs/go/utils/cache` |
| `libs/go/codec` | `github.com/auth-platform/libs/go/utils/codec` |
| `libs/go/diff` | `github.com/auth-platform/libs/go/utils/diff` |
| `libs/go/error` | `github.com/auth-platform/libs/go/utils/error` |
| `libs/go/merge` | `github.com/auth-platform/libs/go/utils/merge` |
| `libs/go/uuid` | `github.com/auth-platform/libs/go/utils/uuid` |
| `libs/go/validated` | `github.com/auth-platform/libs/go/utils/validated` |
| `libs/go/validator` | `github.com/auth-platform/libs/go/utils/validator` |
| `libs/go/testutil` | `github.com/auth-platform/libs/go/testing/testutil` |

## Development

Use Go workspace for local development:

```bash
cd libs/go
go work sync
go build ./...
go test ./...
```
