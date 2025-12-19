module github.com/auth-platform/libs/go/tests/server

go 1.25

require (
	github.com/auth-platform/libs/go/src/server v0.0.0
	github.com/leanovate/gopter v0.2.11
)

replace github.com/auth-platform/libs/go/src/server => ../../src/server
