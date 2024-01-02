module code.cloudfoundry.org/garden-integration-tests

go 1.21.0

toolchain go1.21.3

require (
	code.cloudfoundry.org/archiver v0.0.0-20231206170638-1923b319e771
	code.cloudfoundry.org/garden v0.0.0-20231226182342-f6a5ba9c581c
	code.cloudfoundry.org/guardian v0.0.0-20231226182808-e49d38ea24d1
	github.com/cloudfoundry/gosigar v1.3.40
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/onsi/ginkgo/v2 v2.13.2
	github.com/onsi/gomega v1.30.0
	github.com/wavefronthq/wavefront-sdk-go v0.15.0
)

require (
	code.cloudfoundry.org/commandrunner v0.0.0-20230612151827-2b11a2b4e9b8 // indirect
	code.cloudfoundry.org/lager/v3 v3.0.3 // indirect
	github.com/bmizerany/pat v0.0.0-20210406213842-e4b6760bdd6f // indirect
	github.com/caio/go-tdigest/v4 v4.0.1 // indirect
	github.com/cloudfoundry/dropsonde v1.1.0 // indirect
	github.com/cloudfoundry/sonde-go v0.0.0-20231227232801-b682ba3cb37d // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cyphar/filepath-securejoin v0.2.4 // indirect
	github.com/docker/docker v24.0.7+incompatible // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/pprof v0.0.0-20231229205709-960ae82b1e42 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/moby/sys/mountinfo v0.7.1 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/opencontainers/runc v1.1.11 // indirect
	github.com/opencontainers/runtime-spec v1.1.0 // indirect
	github.com/openzipkin/zipkin-go v0.4.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/tedsuo/rata v1.0.0 // indirect
	github.com/vishvananda/netlink v1.2.1-beta.2 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.16.1 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gotest.tools/v3 v3.5.1 // indirect
)

replace (
	code.cloudfoundry.org/garden => ../garden
	code.cloudfoundry.org/grootfs => ../grootfs
	code.cloudfoundry.org/guardian => ../guardian
	code.cloudfoundry.org/idmapper => ../idmapper
)
