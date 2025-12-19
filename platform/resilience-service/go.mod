module github.com/auth-platform/platform/resilience-service

go 1.25.5

require (
	github.com/auth-platform/libs/go/server/health v0.0.0-00010101000000-000000000000
	github.com/auth-platform/libs/go/resilience/domain v0.0.0-00010101000000-000000000000
	github.com/auth-platform/libs/go/resilience/errors v0.0.0-00010101000000-000000000000
	github.com/auth-platform/libs/go/utils/uuid v0.0.0-00010101000000-000000000000
	github.com/leanovate/gopter v0.2.11
	github.com/redis/go-redis/v9 v9.17.0
	go.opentelemetry.io/otel v1.39.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.39.0
	go.opentelemetry.io/otel/sdk v1.39.0
	go.opentelemetry.io/otel/trace v1.39.0
	google.golang.org/grpc v1.77.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.3 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.opentelemetry.io/proto/otlp v1.9.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

// Local replace directives for libs/go packages (new category-based structure)

// Collections
replace github.com/auth-platform/libs/go/collections/lru => ../../libs/go/collections/lru
replace github.com/auth-platform/libs/go/collections/maps => ../../libs/go/collections/maps
replace github.com/auth-platform/libs/go/collections/pqueue => ../../libs/go/collections/pqueue
replace github.com/auth-platform/libs/go/collections/queue => ../../libs/go/collections/queue
replace github.com/auth-platform/libs/go/collections/set => ../../libs/go/collections/set
replace github.com/auth-platform/libs/go/collections/slices => ../../libs/go/collections/slices
replace github.com/auth-platform/libs/go/collections/sort => ../../libs/go/collections/sort

// Concurrency
replace github.com/auth-platform/libs/go/concurrency/async => ../../libs/go/concurrency/async
replace github.com/auth-platform/libs/go/concurrency/atomic => ../../libs/go/concurrency/atomic
replace github.com/auth-platform/libs/go/concurrency/channels => ../../libs/go/concurrency/channels
replace github.com/auth-platform/libs/go/concurrency/errgroup => ../../libs/go/concurrency/errgroup
replace github.com/auth-platform/libs/go/concurrency/once => ../../libs/go/concurrency/once
replace github.com/auth-platform/libs/go/concurrency/pool => ../../libs/go/concurrency/pool
replace github.com/auth-platform/libs/go/concurrency/syncmap => ../../libs/go/concurrency/syncmap
replace github.com/auth-platform/libs/go/concurrency/waitgroup => ../../libs/go/concurrency/waitgroup

// Functional
replace github.com/auth-platform/libs/go/functional/either => ../../libs/go/functional/either
replace github.com/auth-platform/libs/go/functional/iterator => ../../libs/go/functional/iterator
replace github.com/auth-platform/libs/go/functional/lazy => ../../libs/go/functional/lazy
replace github.com/auth-platform/libs/go/functional/option => ../../libs/go/functional/option
replace github.com/auth-platform/libs/go/functional/pipeline => ../../libs/go/functional/pipeline
replace github.com/auth-platform/libs/go/functional/result => ../../libs/go/functional/result
replace github.com/auth-platform/libs/go/functional/stream => ../../libs/go/functional/stream
replace github.com/auth-platform/libs/go/functional/tuple => ../../libs/go/functional/tuple

// Optics
replace github.com/auth-platform/libs/go/optics/lens => ../../libs/go/optics/lens
replace github.com/auth-platform/libs/go/optics/prism => ../../libs/go/optics/prism

// Patterns
replace github.com/auth-platform/libs/go/patterns/registry => ../../libs/go/patterns/registry
replace github.com/auth-platform/libs/go/patterns/spec => ../../libs/go/patterns/spec

// Events
replace github.com/auth-platform/libs/go/events/builder => ../../libs/go/events/builder
replace github.com/auth-platform/libs/go/events/eventbus => ../../libs/go/events/eventbus
replace github.com/auth-platform/libs/go/events/pubsub => ../../libs/go/events/pubsub

// Resilience
replace github.com/auth-platform/libs/go/resilience/bulkhead => ../../libs/go/resilience/bulkhead
replace github.com/auth-platform/libs/go/resilience/circuitbreaker => ../../libs/go/resilience/circuitbreaker
replace github.com/auth-platform/libs/go/resilience/domain => ../../libs/go/resilience/domain
replace github.com/auth-platform/libs/go/resilience/errors => ../../libs/go/resilience/errors
replace github.com/auth-platform/libs/go/resilience/ratelimit => ../../libs/go/resilience/ratelimit
replace github.com/auth-platform/libs/go/resilience/retry => ../../libs/go/resilience/retry
replace github.com/auth-platform/libs/go/resilience/timeout => ../../libs/go/resilience/timeout

// Server
replace github.com/auth-platform/libs/go/server/health => ../../libs/go/server/health
replace github.com/auth-platform/libs/go/server/shutdown => ../../libs/go/server/shutdown
replace github.com/auth-platform/libs/go/server/tracing => ../../libs/go/server/tracing

// gRPC
replace github.com/auth-platform/libs/go/grpc/errors => ../../libs/go/grpc/errors

// Utils
replace github.com/auth-platform/libs/go/utils/audit => ../../libs/go/utils/audit
replace github.com/auth-platform/libs/go/utils/cache => ../../libs/go/utils/cache
replace github.com/auth-platform/libs/go/utils/codec => ../../libs/go/utils/codec
replace github.com/auth-platform/libs/go/utils/diff => ../../libs/go/utils/diff
replace github.com/auth-platform/libs/go/utils/error => ../../libs/go/utils/error
replace github.com/auth-platform/libs/go/utils/merge => ../../libs/go/utils/merge
replace github.com/auth-platform/libs/go/utils/uuid => ../../libs/go/utils/uuid
replace github.com/auth-platform/libs/go/utils/validated => ../../libs/go/utils/validated
replace github.com/auth-platform/libs/go/utils/validator => ../../libs/go/utils/validator

// Testing
replace github.com/auth-platform/libs/go/testing/testutil => ../../libs/go/testing/testutil
