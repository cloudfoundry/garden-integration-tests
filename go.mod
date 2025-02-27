module code.cloudfoundry.org/garden-integration-tests

go 1.23.0

toolchain go1.23.6

// PIN filepath-securejoin until the https://github.com/cyphar/filepath-securejoin/pull/44/commits/0c2fbe6ccc9604cdb6870e24cf4e154cbaf16801
// is released
replace (
	github.com/cyphar/filepath-securejoin => github.com/cyphar/filepath-securejoin v0.3.6
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.2.4
)

require (
	code.cloudfoundry.org/archiver v0.26.0
	code.cloudfoundry.org/garden v0.0.0-20250227212620-c0ba6c64516b
	code.cloudfoundry.org/guardian v0.0.0-20250227214220-2f2f63d68234
	github.com/cloudfoundry/gosigar v1.3.88
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/onsi/ginkgo/v2 v2.22.2
	github.com/onsi/gomega v1.36.2
	github.com/wavefronthq/wavefront-sdk-go v0.15.0
)

require (
	code.cloudfoundry.org/commandrunner v0.29.0 // indirect
	code.cloudfoundry.org/lager/v3 v3.27.0 // indirect
	github.com/bmizerany/pat v0.0.0-20210406213842-e4b6760bdd6f // indirect
	github.com/caio/go-tdigest/v4 v4.0.1 // indirect
	github.com/cloudfoundry/dropsonde v1.1.0 // indirect
	github.com/cloudfoundry/sonde-go v0.0.0-20250217093333-7351ffed9384 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cyphar/filepath-securejoin v0.4.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20250208200701-d0013a598941 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/moby/sys/reexec v0.1.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/opencontainers/runc v1.2.5 // indirect
	github.com/opencontainers/runtime-spec v1.2.0 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/tedsuo/rata v1.0.0 // indirect
	github.com/vishvananda/netlink v1.3.0 // indirect
	github.com/vishvananda/netns v0.0.5 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	golang.org/x/tools v0.30.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	code.cloudfoundry.org/garden => ../garden
	code.cloudfoundry.org/grootfs => ../grootfs
	code.cloudfoundry.org/guardian => ../guardian
	code.cloudfoundry.org/idmapper => ../idmapper
)
