module github.com/authcorp/libs/go/src/events

go 1.25

require (
	github.com/authcorp/libs/go/src/functional v0.0.0
	github.com/authcorp/libs/go/src/resilience v0.0.0
)

replace github.com/authcorp/libs/go/src/functional => ../functional

replace github.com/authcorp/libs/go/src/resilience => ../resilience
