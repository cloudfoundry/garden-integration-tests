module code.cloudfoundry.org/garden-integration-tests

go 1.22.7

toolchain go1.23.2

require (
	code.cloudfoundry.org/archiver v0.23.0
	code.cloudfoundry.org/garden v0.0.0-20241218021336-76cb7fa71f17
	code.cloudfoundry.org/guardian v0.0.0-20241218022048-e6383bd9744d
	github.com/cloudfoundry/gosigar v1.3.80
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/onsi/ginkgo/v2 v2.22.1
	github.com/onsi/gomega v1.36.1
	github.com/wavefronthq/wavefront-sdk-go v0.15.0
)

require (
	code.cloudfoundry.org/commandrunner v0.21.0 // indirect
	code.cloudfoundry.org/lager/v3 v3.19.0 // indirect
	github.com/bmizerany/pat v0.0.0-20210406213842-e4b6760bdd6f // indirect
	github.com/caio/go-tdigest/v4 v4.0.1 // indirect
	github.com/cloudfoundry/dropsonde v1.1.0 // indirect
	github.com/cloudfoundry/sonde-go v0.0.0-20241016180203-3c0e1c24e908 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cyphar/filepath-securejoin v0.3.6 // indirect
	github.com/docker/docker v27.4.1+incompatible // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/pprof v0.0.0-20241210010833-40e02aabc2ad // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/opencontainers/runc v1.2.3 // indirect
	github.com/opencontainers/runtime-spec v1.2.0 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/tedsuo/rata v1.0.0 // indirect
	github.com/vishvananda/netlink v1.3.0 // indirect
	github.com/vishvananda/netns v0.0.5 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/tools v0.28.0 // indirect
	google.golang.org/protobuf v1.36.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	code.cloudfoundry.org/garden => ../garden
	code.cloudfoundry.org/grootfs => ../grootfs
	code.cloudfoundry.org/guardian => ../guardian
	code.cloudfoundry.org/idmapper => ../idmapper
)
