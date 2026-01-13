module github.com/auth-platform/iam-policy-service

go 1.25.5

require (
	// Platform libs
	github.com/authcorp/libs/go/src/cache v0.0.0
	github.com/authcorp/libs/go/src/config v0.0.0
	github.com/authcorp/libs/go/src/fault v0.0.0
	github.com/authcorp/libs/go/src/logging v0.0.0
	github.com/authcorp/libs/go/src/observability v0.0.0

	// External dependencies
	github.com/fsnotify/fsnotify v1.8.0
	github.com/google/uuid v1.6.0
	github.com/open-policy-agent/opa v1.0.0
	github.com/prometheus/client_golang v1.20.5
	go.opentelemetry.io/otel v1.35.0
	google.golang.org/grpc v1.70.0
	pgregory.net/rapid v1.2.0
)

require (
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/agnivade/levenshtein v1.2.0 // indirect
	github.com/authcorp/libs/go/src/errors v0.0.0 // indirect
	github.com/authcorp/libs/go/src/functional v0.0.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/tchap/go-patricia/v2 v2.3.1 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/yashtewari/glob-intersection v0.2.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk v1.33.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241209162323-e6fa225c2576 // indirect
	google.golang.org/protobuf v1.36.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace (
	github.com/authcorp/libs/go/src/cache => ../../libs/go/src/cache
	github.com/authcorp/libs/go/src/config => ../../libs/go/src/config
	github.com/authcorp/libs/go/src/errors => ../../libs/go/src/errors
	github.com/authcorp/libs/go/src/fault => ../../libs/go/src/fault
	github.com/authcorp/libs/go/src/functional => ../../libs/go/src/functional
	github.com/authcorp/libs/go/src/grpc => ../../libs/go/src/grpc
	github.com/authcorp/libs/go/src/logging => ../../libs/go/src/logging
	github.com/authcorp/libs/go/src/observability => ../../libs/go/src/observability
	github.com/authcorp/libs/go/src/server => ../../libs/go/src/server
	github.com/authcorp/libs/go/src/testing => ../../libs/go/src/testing
	github.com/authcorp/libs/go/src/validation => ../../libs/go/src/validation
)
