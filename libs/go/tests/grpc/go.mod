module github.com/auth-platform/libs/go/tests/grpc

go 1.25

require (
	github.com/auth-platform/libs/go/src/grpc v0.0.0
	github.com/leanovate/gopter v0.2.11
)

replace github.com/auth-platform/libs/go/src/grpc => ../../src/grpc
