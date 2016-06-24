package garden_integration_tests_test

import (
	"io"

	"github.com/cloudfoundry-incubator/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Rootfses", func() {
	Context("when the rootfs path is a docker image URL", func() {
		Context("and the docker image specifies $PATH", func() {
			BeforeEach(func() {
				// Dockerfile contains:
				//   ENV PATH /usr/local/bin:/usr/bin:/bin:/from-dockerfile
				//   ENV TEST test-from-dockerfile
				//   ENV TEST second-test-from-dockerfile:$TEST
				// see diego-dockerfiles/with-volume
				rootfs = "docker:///cfgarden/with-volume"
			})

			JustBeforeEach(func() {
				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "adduser",
					Args: []string{"-D", "bob"},
				}, garden.ProcessIO{
					Stdout: GinkgoWriter,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(process.Wait()).To(Equal(0))
			})

			It("$PATH is taken from the docker image", func() {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "bob",
					Path: "/bin/sh",
					Args: []string{"-c", "echo $PATH"},
				}, garden.ProcessIO{
					Stdout: io.MultiWriter(GinkgoWriter, stdout),
					Stderr: GinkgoWriter,
				})

				Expect(err).ToNot(HaveOccurred())

				process.Wait()
				Expect(stdout).To(gbytes.Say("/usr/local/bin:/usr/bin:/bin:/from-dockerfile"))
			})

			It("$TEST is taken from the docker image", func() {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "bob",
					Path: "/bin/sh",
					Args: []string{"-c", "echo $TEST"},
				}, garden.ProcessIO{
					Stdout: io.MultiWriter(GinkgoWriter, stdout),
					Stderr: GinkgoWriter,
				})

				Expect(err).ToNot(HaveOccurred())

				process.Wait()
				Expect(stdout).To(gbytes.Say("second-test-from-dockerfile:test-from-dockerfile"))
			})
		})
	})
})
