module github.com/authcorp/libs/go/tests/fault

go 1.25.5

require (
	github.com/authcorp/libs/go/src/fault v0.0.0
	pgregory.net/rapid v1.2.0
)

require (
	github.com/authcorp/libs/go/src/errors v0.0.0 // indirect
	github.com/authcorp/libs/go/src/functional v0.0.0 // indirect
)

replace github.com/authcorp/libs/go/src/fault => ../../src/fault

replace github.com/authcorp/libs/go/src/functional => ../../src/functional

replace github.com/authcorp/libs/go/src/errors => ../../src/errors
