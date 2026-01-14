module github.com/auth-platform/file-upload

go 1.25.5

require (
	github.com/authcorp/libs/go/src/observability v0.0.0
	// AWS SDK v2 - latest stable 2025
	github.com/aws/aws-sdk-go-v2 v1.32.8
	github.com/aws/aws-sdk-go-v2/config v1.28.10
	github.com/aws/aws-sdk-go-v2/service/s3 v1.72.2

	// JWT
	github.com/golang-jwt/jwt/v5 v5.2.1

	// UUID
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1

	// File type detection
	github.com/h2non/filetype v1.1.3

	// Database
	github.com/jmoiron/sqlx v1.4.0

	// gRPC for platform services
	google.golang.org/grpc v1.69.2
	google.golang.org/protobuf v1.36.1

	// Property-based testing
	pgregory.net/rapid v1.2.0
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.7 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.51 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.27 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.27 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.27 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.4.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.24.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.28.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.6 // indirect
	github.com/aws/smithy-go v1.22.1 // indirect

	// OpenTelemetry 1.28+ (modern observability)
	go.opentelemetry.io/otel v1.33.0 // indirect
	go.opentelemetry.io/otel/sdk v1.33.0 // indirect
	golang.org/x/net v0.32.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241209162323-e6fa225c2576 // indirect
)

replace (
	github.com/authcorp/libs/go/src/codec => ../../libs/go/src/codec
	github.com/authcorp/libs/go/src/config => ../../libs/go/src/config
	github.com/authcorp/libs/go/src/domain => ../../libs/go/src/domain
	github.com/authcorp/libs/go/src/errors => ../../libs/go/src/errors
	github.com/authcorp/libs/go/src/fault => ../../libs/go/src/fault
	github.com/authcorp/libs/go/src/functional => ../../libs/go/src/functional
	github.com/authcorp/libs/go/src/grpc => ../../libs/go/src/grpc
	github.com/authcorp/libs/go/src/http => ../../libs/go/src/http
	github.com/authcorp/libs/go/src/observability => ../../libs/go/src/observability
	github.com/authcorp/libs/go/src/pagination => ../../libs/go/src/pagination
	github.com/authcorp/libs/go/src/security => ../../libs/go/src/security
	github.com/authcorp/libs/go/src/server => ../../libs/go/src/server
	github.com/authcorp/libs/go/src/testing => ../../libs/go/src/testing
	github.com/authcorp/libs/go/src/validation => ../../libs/go/src/validation
	github.com/authcorp/libs/go/src/versioning => ../../libs/go/src/versioning
	github.com/authcorp/libs/go/src/workerpool => ../../libs/go/src/workerpool
)
