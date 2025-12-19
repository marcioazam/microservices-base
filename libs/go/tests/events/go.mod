module github.com/authcorp/libs/go/tests/events

go 1.25

require (
	github.com/authcorp/libs/go/src/events v0.0.0
	pgregory.net/rapid v1.1.0
)

replace github.com/authcorp/libs/go/src/events => ../../src/events

replace github.com/authcorp/libs/go/src/functional => ../../src/functional

replace github.com/authcorp/libs/go/src/resilience => ../../src/resilience
