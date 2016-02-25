package garden_integration_tests_test

import (
	"fmt"
	"net"
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

		hostPort, _, err := container.NetIn(0, 8080)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() string {
			out := gbytes.NewBuffer()
			process, err := container.Run(garden.ProcessSpec{
				Path: "netstat",
				Args: []string{"-a"},
				User: "root",
			}, garden.ProcessIO{
				Stdout: out,
			})
			Expect(err).ToNot(HaveOccurred())

			exitCode, err := process.Wait()
			Expect(err).NotTo(HaveOccurred())
			Expect(exitCode).To(Equal(0))

			return string(out.Contents())
		}).Should(ContainSubstring("LISTEN"))

		gardenHostname := strings.Split(gardenHost, ":")[0]
		nc, err := gexec.Start(exec.Command("nc", gardenHostname, fmt.Sprintf("%d", hostPort)), GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(nc).Should(gbytes.Say("hallo"))
		Eventually(nc).Should(gexec.Exit(0))
	})
})

func checkInternet(container garden.Container, externalIP net.IP) error {
	return checkConnection(container, externalIP.String(), 80)
}

func checkConnection(container garden.Container, ip string, port int) error {
	process, err := container.Run(garden.ProcessSpec{
		User: "alice",
		Path: "sh",
		Args: []string{"-c", fmt.Sprintf("echo hello | nc -w1 %s %d", ip, port)},
	}, garden.ProcessIO{Stdout: GinkgoWriter, Stderr: GinkgoWriter})
	if err != nil {
		return err
	}

	exitCode, err := process.Wait()
	if err != nil {
		return err
	}

	if exitCode == 0 {
		return nil
	} else {
		return fmt.Errorf("Request failed. Process exited with code %d", exitCode)
	}
}
