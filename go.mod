module code.cloudfoundry.org/garden-integration-tests

go 1.12

require (
	code.cloudfoundry.org/archiver v0.0.0-20180525162158-e135af3d5a2a
	code.cloudfoundry.org/garden v0.0.0-00010101000000-000000000000
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
)

replace code.cloudfoundry.org/garden => ../garden
