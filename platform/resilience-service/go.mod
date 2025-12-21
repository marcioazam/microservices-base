module github.com/auth-platform/platform/resilience-service

go 1.25.5

require (
	github.com/authcorp/libs/go/src/collections v0.0.0
	github.com/authcorp/libs/go/src/functional v0.0.0
	github.com/authcorp/libs/go/src/fault v0.0.0
	github.com/authcorp/libs/go/src/validation v0.0.0
	github.com/failsafe-go/failsafe-go v0.6.9
	github.com/go-playground/validator/v10 v10.29.0
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.3
	github.com/leanovate/gopter v0.2.11
	github.com/redis/go-redis/v9 v9.17.0
	github.com/spf13/viper v1.8.1
	go.opentelemetry.io/otel v1.39.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.39.0
	go.opentelemetry.io/otel/metric v1.39.0
	go.opentelemetry.io/otel/sdk v1.39.0
	go.opentelemetry.io/otel/trace v1.39.0
	go.uber.org/fx v1.23.0
	google.golang.org/grpc v1.77.0
	google.golang.org/protobuf v1.36.10
	pgregory.net/rapid v1.2.0
)

require (
	github.com/authcorp/libs/go/src/errors v0.0.0 // indirect
	github.com/bits-and-blooms/bitset v1.14.3 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/gabriel-vasile/mimetype v1.4.11 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.3 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/pelletier/go-toml v1.9.3 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.39.0 // indirect
	go.opentelemetry.io/proto/otlp v1.9.0 // indirect
	go.uber.org/dig v1.18.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

// Local replace directives for libs/go/src packages
replace (
	github.com/authcorp/libs/go/src/collections => ../../libs/go/src/collections
	github.com/authcorp/libs/go/src/errors => ../../libs/go/src/errors
	github.com/authcorp/libs/go/src/functional => ../../libs/go/src/functional
	github.com/authcorp/libs/go/src/patterns => ../../libs/go/src/patterns
	github.com/authcorp/libs/go/src/fault => ../../libs/go/src/fault
	github.com/authcorp/libs/go/src/validation => ../../libs/go/src/validation
)
