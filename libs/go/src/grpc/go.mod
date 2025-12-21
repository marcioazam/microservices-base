module github.com/authcorp/libs/go/src/grpc

go 1.25

require (
	github.com/authcorp/libs/go/src/fault v0.0.0
	google.golang.org/grpc v1.60.0
)

replace github.com/authcorp/libs/go/src/fault => ../fault

replace github.com/authcorp/libs/go/src/functional => ../functional
