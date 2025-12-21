module github.com/authcorp/libs/go/tests/server

go 1.25.5

require (
	github.com/authcorp/libs/go/src/observability v0.0.0
	github.com/authcorp/libs/go/src/server v0.0.0
	github.com/leanovate/gopter v0.2.11
)

replace (
	github.com/authcorp/libs/go/src/observability => ../../src/observability
	github.com/authcorp/libs/go/src/server => ../../src/server
)
