module github.com/authcorp/libs/go/tests/testing

go 1.25.5

require (
	github.com/authcorp/libs/go/src/functional v0.0.0
	github.com/authcorp/libs/go/src/testing v0.0.0
	pgregory.net/rapid v1.1.0
)

replace github.com/authcorp/libs/go/src/testing => ../../src/testing

replace github.com/authcorp/libs/go/src/functional => ../../src/functional
