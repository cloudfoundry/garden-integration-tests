package garden_integration_tests_test

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/cloudfoundry-incubator/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Networking", func() {
	It("can be contacted after a NetIn", func() {
		process, err := container.Run(garden.ProcessSpec{
			Path: "sh",
			Args: []string{"-c", "echo hallo | nc -l -p 8080"},
			User: "root",
		}, garden.ProcessIO{
			Stdout: GinkgoWriter,
			Stderr: GinkgoWriter,
		})
		Expect(err).ToNot(HaveOccurred())

		defer func() {
			process.Signal(garden.SignalTerminate)
			_, err := process.Wait()
			Expect(err).NotTo(HaveOccurred())
		}()

		gardenHostname := strings.Split(gardenHost, ":")[0]

		hostPort, _, err := container.NetIn(0, 8080)
		nc, err := gexec.Start(exec.Command("nc", gardenHostname, fmt.Sprintf("%d", hostPort)), GinkgoWriter, GinkgoWriter)
		Eventually(nc).Should(gbytes.Say("hallo"))
		Eventually(nc).Should(gexec.Exit(0))
	})
})
