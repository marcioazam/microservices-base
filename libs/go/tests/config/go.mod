module github.com/auth-platform/libs/go/tests/config

go 1.25.5

require (
	github.com/auth-platform/libs/go/config v0.0.0
	pgregory.net/rapid v1.2.0
)

require (
	github.com/kr/pretty v0.1.0 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/auth-platform/libs/go/config => ../../src/config
