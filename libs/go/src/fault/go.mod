module github.com/authcorp/libs/go/src/fault

go 1.25

require (
	github.com/authcorp/libs/go/src/errors v0.0.0
	github.com/authcorp/libs/go/src/functional v0.0.0
)

replace (
	github.com/authcorp/libs/go/src/errors => ../errors
	github.com/authcorp/libs/go/src/functional => ../functional
)
