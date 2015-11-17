package performance_test

import (
	"time"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/garden/client"
	"github.com/cloudfoundry-incubator/garden/client/connection"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

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
		stdout := gbytes.NewBuffer()
		stderr := gbytes.NewBuffer()

		process, err := container.Run(garden.ProcessSpec{
			User: "alice",
			Path: "sh",
			Args: []string{"-c", "while true; do sleep 1; done"},
		}, garden.ProcessIO{
			Stdout: stdout,
			Stderr: stderr,
		})
		Expect(err).ToNot(HaveOccurred())
		go process.Wait()
	})

	AfterEach(func() {
		Expect(gardenClient.Destroy(container.Handle())).To(Succeed())
	})

	RunSpecs(t, "Performance Suite")
}
