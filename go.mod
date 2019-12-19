module code.cloudfoundry.org/garden-integration-tests

go 1.12

require (
	code.cloudfoundry.org/archiver v0.0.0-20180525162158-e135af3d5a2a
	code.cloudfoundry.org/garden v0.0.0-00010101000000-000000000000
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.7.1
	golang.org/x/net v0.0.0-20191028085509-fe3aa8a45271 // indirect
	golang.org/x/sys v0.0.0-20191029155521-f43be2a4598c // indirect
)

replace code.cloudfoundry.org/garden => ../garden
