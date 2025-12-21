module github.com/authcorp/libs/go/tests/collections

go 1.25.5

require (
	github.com/authcorp/libs/go/src/collections v0.0.0
	pgregory.net/rapid v1.2.0
)

require github.com/authcorp/libs/go/src/functional v0.0.0

replace github.com/authcorp/libs/go/src/collections => ../../src/collections

replace github.com/authcorp/libs/go/src/functional => ../../src/functional
