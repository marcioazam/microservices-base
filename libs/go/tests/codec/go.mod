module github.com/authcorp/libs/go/tests/codec

go 1.25.5

require (
	github.com/authcorp/libs/go/src/codec v0.0.0
	pgregory.net/rapid v1.2.0
)

require (
	github.com/authcorp/libs/go/src/functional v0.0.0 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/authcorp/libs/go/src/codec => ../../src/codec

replace github.com/authcorp/libs/go/src/functional => ../../src/functional
