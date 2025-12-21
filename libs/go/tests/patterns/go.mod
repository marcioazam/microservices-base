module github.com/auth-platform/libs/go/tests/patterns

go 1.25.5

require (
	github.com/authcorp/libs/go/src/functional v0.0.0
	github.com/authcorp/libs/go/src/patterns v0.0.0
	github.com/leanovate/gopter v0.2.11
	pgregory.net/rapid v1.2.0
)

replace (
	github.com/authcorp/libs/go/src/functional => ../../src/functional
	github.com/authcorp/libs/go/src/patterns => ../../src/patterns
)
