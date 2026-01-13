module github.com/authcorp/libs/go/src/logging

go 1.24

require (
	github.com/authcorp/libs/go/src/observability v0.0.0
	google.golang.org/grpc v1.69.2
	google.golang.org/protobuf v1.36.1
)

replace github.com/authcorp/libs/go/src/observability => ../observability
