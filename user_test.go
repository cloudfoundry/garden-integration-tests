package garden_integration_tests_test

import (
	"bytes"
	"io"

	"github.com/cloudfoundry-incubator/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("users", func() {

	Context("when nobody maps to 65534", func() {
		BeforeEach(func() {
			rootfs = "docker:///ubuntu"
		})

		It("should be able to su to nobody", func() {
			// Delete guff.
			proc, err := container.Run(garden.ProcessSpec{
				User: "root",
				Path: "rm",
				Args: []string{"-f", "/etc/passwd-"},
			}, garden.ProcessIO{
				Stdout: GinkgoWriter,
				Stderr: GinkgoWriter,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(proc.Wait()).To(Equal(0))

			// Ensure "nobody" has a shell.
			proc, err = container.Run(garden.ProcessSpec{
				User: "root",
				Path: "chsh",
				Args: []string{"-s", "/bin/bash", "nobody"},
			}, garden.ProcessIO{
				Stdout: GinkgoWriter,
				Stderr: GinkgoWriter,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(proc.Wait()).To(Equal(0))

			var buf bytes.Buffer
			proc, err = container.Run(garden.ProcessSpec{
				User: "root",
				Path: "su",
				Args: []string{"nobody", "-c", "whoami"},
			}, garden.ProcessIO{
				Stdout: io.MultiWriter(&buf, GinkgoWriter),
				Stderr: io.MultiWriter(&buf, GinkgoWriter),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(proc.Wait()).To(Equal(0))
			Expect(buf.String()).To(Equal("nobody\n"))
		})
	})

})
