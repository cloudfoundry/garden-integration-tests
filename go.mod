module code.cloudfoundry.org/garden-integration-tests

go 1.16

require (
	code.cloudfoundry.org/archiver v0.0.0-20210609160716-67523bd33dbf
	code.cloudfoundry.org/garden v0.0.0-20210608104724-fa3a10d59c82
	code.cloudfoundry.org/guardian v0.0.0-20220607160814-bbdc1696f4d2
	github.com/cloudfoundry/gosigar v1.3.3
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/moby/sys/mountinfo v0.6.0 // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.19.0
	github.com/wavefronthq/wavefront-sdk-go v0.9.10
)

replace (
	code.cloudfoundry.org/garden => ../garden
	code.cloudfoundry.org/grootfs => ../grootfs
	code.cloudfoundry.org/guardian => ../guardian
	code.cloudfoundry.org/idmapper => ../idmapper
	github.com/docker/docker => github.com/docker/docker v20.10.13+incompatible
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.1.0
	golang.org/x/text => golang.org/x/text v0.3.7
)
