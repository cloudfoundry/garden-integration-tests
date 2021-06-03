module code.cloudfoundry.org/garden-integration-tests

go 1.16

require (
	code.cloudfoundry.org/archiver v0.0.0-20210513174825-6979f8d756e2
	code.cloudfoundry.org/garden v0.0.0-20210208153517-580cadd489d2
	code.cloudfoundry.org/guardian v0.0.0-20210527102652-2f945c09a983
	github.com/cloudfoundry/gosigar v1.2.0
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/wavefronthq/wavefront-sdk-go v0.9.8
)

replace (
	code.cloudfoundry.org/garden => ../garden
	code.cloudfoundry.org/grootfs => ../grootfs
	code.cloudfoundry.org/guardian => ../guardian
	code.cloudfoundry.org/idmapper => ../idmapper
)
