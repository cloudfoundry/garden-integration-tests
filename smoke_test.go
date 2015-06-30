package garden_integration_tests_test

import (
	"os"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/garden/client"
	"github.com/cloudfoundry-incubator/garden/client/connection"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Garden smoke tests", func() {

	var (
		gardenClient garden.Client
		container    garden.Container
	)

	BeforeEach(func() {
		gardenClient = client.New(connection.New("tcp", os.Getenv("GARDEN_ADDRESS")))

		var err error
		container, err = gardenClient.Create(garden.ContainerSpec{})
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(gardenClient.Destroy(container.Handle())).To(Succeed())
	})

	It("can run a process inside a container", func() {
		stdout := gbytes.NewBuffer()

		_, err := container.Run(garden.ProcessSpec{
			Path: "whoami",
			User: "root",
		}, garden.ProcessIO{
			Stdout: stdout,
			Stderr: GinkgoWriter,
		})

		Expect(err).ToNot(HaveOccurred())
		Eventually(stdout).Should(gbytes.Say("root\n"))
	})
})
