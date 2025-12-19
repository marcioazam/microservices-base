module github.com/authcorp/libs/go/tests/utils

go 1.25.5

require (
	github.com/authcorp/libs/go/src/utils v0.0.0
	pgregory.net/rapid v1.1.0
)

require github.com/authcorp/libs/go/src/functional v0.0.0 // indirect

replace github.com/authcorp/libs/go/src/utils => ../../src/utils

replace github.com/authcorp/libs/go/src/functional => ../../src/functional
