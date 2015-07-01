package garden_integration_tests_test

import (
	"os"
	"testing"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/garden/client"
	"github.com/cloudfoundry-incubator/garden/client/connection"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	gardenClient garden.Client
	container    garden.Container

	gardenHost string
	rootfs     string
)

func TestGardenIntegrationTests(t *testing.T) {
	RegisterFailHandler(Fail)

	BeforeEach(func() {
		rootfs = ""
		gardenHost = os.Getenv("GARDEN_ADDRESS")
	})

	JustBeforeEach(func() {
		gardenClient = client.New(connection.New("tcp", gardenHost))

		var err error
		container, err = gardenClient.Create(garden.ContainerSpec{RootFSPath: rootfs})
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(gardenClient.Destroy(container.Handle())).To(Succeed())
	})

	RunSpecs(t, "GardenIntegrationTests Suite")
}
