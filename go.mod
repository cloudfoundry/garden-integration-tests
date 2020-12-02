module code.cloudfoundry.org/garden-integration-tests

go 1.14

require (
	code.cloudfoundry.org/archiver v0.0.0-20180525162158-e135af3d5a2a
	code.cloudfoundry.org/garden v0.0.0-00010101000000-000000000000
	code.cloudfoundry.org/guardian v0.0.0-00010101000000-000000000000
	github.com/cloudfoundry/gosigar v1.1.0
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.3
	github.com/wavefronthq/wavefront-sdk-go v0.9.7
)

replace (
	code.cloudfoundry.org/garden => ../garden
	code.cloudfoundry.org/grootfs => ../grootfs
	code.cloudfoundry.org/guardian => ../guardian
	code.cloudfoundry.org/idmapper => ../idmapper
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc90
)
