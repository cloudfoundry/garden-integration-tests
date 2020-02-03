module code.cloudfoundry.org/garden-integration-tests

go 1.12

require (
	code.cloudfoundry.org/archiver v0.0.0-20180525162158-e135af3d5a2a
	code.cloudfoundry.org/garden v0.0.0-00010101000000-000000000000
	code.cloudfoundry.org/guardian v0.0.0-00010101000000-000000000000
	github.com/cloudfoundry/gosigar v1.1.0
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.9.0
)

replace code.cloudfoundry.org/garden => ../garden

replace code.cloudfoundry.org/guardian => ../guardian

replace code.cloudfoundry.org/grootfs => ../grootfs

replace code.cloudfoundry.org/idmapper => ../idmapper
