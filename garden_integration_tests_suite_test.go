package garden_integration_tests_test

import (
	"os"
	"testing"
	"time"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/garden/client"
	"github.com/cloudfoundry-incubator/garden/client/connection"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	gardenHost            string
	gardenClient          garden.Client
	container             garden.Container
	containerCreateErr    error
	assertContainerCreate bool

	rootfs              string
	privilegedContainer bool
	properties          garden.Properties
	limits              garden.Limits
)

func TestGardenIntegrationTests(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(5 * time.Second)

	BeforeEach(func() {
		assertContainerCreate = true
		rootfs = "docker:///cloudfoundry/garden-busybox"
		privilegedContainer = false
		properties = garden.Properties{}
		limits = garden.Limits{}
		gardenHost = os.Getenv("GARDEN_ADDRESS")
		gardenClient = client.New(connection.New("tcp", gardenHost))
	})

	JustBeforeEach(func() {
		container, containerCreateErr = gardenClient.Create(garden.ContainerSpec{
			RootFSPath: rootfs,
			Privileged: privilegedContainer,
			Properties: properties,
			Limits:     limits,
		})

		if assertContainerCreate {
			Expect(containerCreateErr).ToNot(HaveOccurred())
		}
	})

	AfterEach(func() {
		if container != nil {
			Expect(gardenClient.Destroy(container.Handle())).To(Succeed())
		}
	})

	RunSpecs(t, "GardenIntegrationTests Suite")
}

func getContainerHandles() []string {
	containers, err := gardenClient.Containers(nil)
	Expect(err).ToNot(HaveOccurred())

	handles := make([]string, len(containers))
	for i, c := range containers {
		handles[i] = c.Handle()
	}

	return handles
}
