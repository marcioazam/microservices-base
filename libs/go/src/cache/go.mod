module github.com/authcorp/libs/go/src/cache

go 1.24

require (
	github.com/authcorp/libs/go/src/fault v0.0.0
	github.com/authcorp/libs/go/src/functional v0.0.0
	github.com/authcorp/libs/go/src/observability v0.0.0
	google.golang.org/grpc v1.69.2
	google.golang.org/protobuf v1.36.1
)

replace (
	github.com/authcorp/libs/go/src/fault => ../fault
	github.com/authcorp/libs/go/src/functional => ../functional
	github.com/authcorp/libs/go/src/observability => ../observability
)
