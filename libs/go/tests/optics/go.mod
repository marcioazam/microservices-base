module github.com/authcorp/libs/go/tests/optics

go 1.25.5

require (
	github.com/authcorp/libs/go/src/functional v0.0.0
	github.com/authcorp/libs/go/src/optics v0.0.0
	github.com/leanovate/gopter v0.2.11
)

replace (
	github.com/authcorp/libs/go/src/functional => ../../src/functional
	github.com/authcorp/libs/go/src/optics => ../../src/optics
)
