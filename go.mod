module code.cloudfoundry.org/garden-integration-tests

go 1.12

require (
	code.cloudfoundry.org/archiver v0.0.0-20180525162158-e135af3d5a2a
	code.cloudfoundry.org/garden v0.0.0-00010101000000-000000000000
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.6.0
	golang.org/x/net v0.0.0-20190827160401-ba9fcec4b297 // indirect
	golang.org/x/sys v0.0.0-20190826190057-c7b8b68b1456 // indirect
	golang.org/x/text v0.3.2 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
)

replace code.cloudfoundry.org/garden => ../garden
