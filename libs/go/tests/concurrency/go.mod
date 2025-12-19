module github.com/authcorp/libs/go/tests/concurrency

go 1.25.5

require (
	github.com/authcorp/libs/go/src/concurrency v0.0.0
	github.com/authcorp/libs/go/src/functional v0.0.0
	pgregory.net/rapid v1.1.0
)

replace github.com/authcorp/libs/go/src/concurrency => ../../src/concurrency

replace github.com/authcorp/libs/go/src/functional => ../../src/functional
