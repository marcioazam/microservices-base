module github.com/auth-platform/cache-service

go 1.25.5

require (
	github.com/authcorp/libs/go/src/fault v0.0.0
	github.com/authcorp/libs/go/src/grpc v0.0.0
	github.com/authcorp/libs/go/src/http v0.0.0
	github.com/authcorp/libs/go/src/observability v0.0.0
	github.com/authcorp/libs/go/src/server v0.0.0
	github.com/go-chi/chi/v5 v5.2.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/leanovate/gopter v0.2.11
	github.com/prometheus/client_golang v1.20.5
	github.com/rabbitmq/amqp091-go v1.10.0
	github.com/redis/go-redis/v9 v9.7.0
	github.com/segmentio/kafka-go v0.4.47
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/otel v1.33.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.33.0
	go.opentelemetry.io/otel/sdk v1.33.0
	go.opentelemetry.io/otel/trace v1.33.0
	google.golang.org/grpc v1.69.2
	google.golang.org/protobuf v1.36.1
	pgregory.net/rapid v1.2.0
)

replace (
	github.com/authcorp/libs/go/src/errors => ../../libs/go/src/errors
	github.com/authcorp/libs/go/src/fault => ../../libs/go/src/fault
	github.com/authcorp/libs/go/src/functional => ../../libs/go/src/functional
	github.com/authcorp/libs/go/src/grpc => ../../libs/go/src/grpc
	github.com/authcorp/libs/go/src/http => ../../libs/go/src/http
	github.com/authcorp/libs/go/src/observability => ../../libs/go/src/observability
	github.com/authcorp/libs/go/src/server => ../../libs/go/src/server
)

require (
	github.com/authcorp/libs/go/src/errors v0.0.0 // indirect
	github.com/authcorp/libs/go/src/functional v0.0.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.24.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.33.0 // indirect
	go.opentelemetry.io/otel/metric v1.33.0 // indirect
	go.opentelemetry.io/proto/otlp v1.4.0 // indirect
	golang.org/x/net v0.32.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241209162323-e6fa225c2576 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241209162323-e6fa225c2576 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
