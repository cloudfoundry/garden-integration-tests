package garden_integration_tests_test

import (
	"bytes"
	"io"

	"code.cloudfoundry.org/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Partially shared containers (peas)", func() {
	It("runs a process in its own rootfs", func() {
		var stdout bytes.Buffer
		process, err := container.Run(garden.ProcessSpec{
			Path:  "cat",
			Args:  []string{"/etc/os-release"},
			Image: garden.ImageRef{URI: "docker:///alpine#3.6"},
		}, garden.ProcessIO{
			Stdin:  bytes.NewBuffer(nil),
			Stdout: io.MultiWriter(&stdout, GinkgoWriter),
			Stderr: GinkgoWriter,
		})
		Expect(err).NotTo(HaveOccurred())
		processExitCode, err := process.Wait()
		Expect(err).NotTo(HaveOccurred())
		Expect(processExitCode).To(Equal(0))
		Expect(stdout.String()).To(ContainSubstring(`NAME="Alpine Linux"`))
	})
})
