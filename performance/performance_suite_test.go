package performance_test

import (
	"os"
	"time"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/garden/client"
	"github.com/cloudfoundry-incubator/garden/client/connection"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var (
	gardenHost   string
	gardenClient garden.Client
	container    garden.Container

	rootfs string
)

func TestPerformance(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(5 * time.Second)

	BeforeEach(func() {
		gardenHost = os.Getenv("GARDEN_ADDRESS")
		rootfs = "docker:///cloudfoundry/garden-busybox"
	})

	JustBeforeEach(func() {
		gardenClient = client.New(connection.New("tcp", gardenHost))

		var err error
		container, err = gardenClient.Create(garden.ContainerSpec{
			RootFSPath: rootfs,
		})
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(gardenClient.Destroy(container.Handle())).To(Succeed())
	})

	RunSpecs(t, "Performance Suite")
}
