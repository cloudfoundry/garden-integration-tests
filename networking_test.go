package garden_integration_tests_test

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

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
		Expect(err).ToNot(HaveOccurred())

		// Allow nc time to start running.
		time.Sleep(2 * time.Second)

		nc, err := gexec.Start(exec.Command("nc", gardenHostname, fmt.Sprintf("%d", hostPort)), GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(nc).Should(gbytes.Say("hallo"))
		Eventually(nc).Should(gexec.Exit(0))
	})

	It("can access a remote address after a NetOut", func() {
		ips, err := net.LookupIP("www.example.com")
		Expect(err).ToNot(HaveOccurred())
		Expect(ips).ToNot(BeEmpty())
		externalIP := ips[0]

		err = container.NetOut(garden.NetOutRule{
			Networks: []garden.IPRange{
				garden.IPRangeFromIP(externalIP),
			},
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(checkInternet(container, externalIP)).To(Succeed())
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
