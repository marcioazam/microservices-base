module github.com/authcorp/libs/go/src/grpc

go 1.25

require (
	github.com/authcorp/libs/go/src/resilience v0.0.0
	google.golang.org/grpc v1.60.0
)

replace github.com/authcorp/libs/go/src/resilience => ../resilience

replace github.com/authcorp/libs/go/src/functional => ../functional
