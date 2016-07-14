package garden_integration_tests_test

import (
	"io"

	"code.cloudfoundry.org/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Rootfses", func() {
	BeforeEach(func() {
		rootfs = "docker:///cfgarden/with-volume"
	})

	JustBeforeEach(func() {
		createUser(container, "bob")
	})

	Context("when the rootfs path is a docker image URL", func() {
		Context("and the image specifies $PATH", func() {
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

		Context("and the image specifies a VOLUME", func() {
			It("creates the volume directory, if it does not already exist", func() {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "bob",
					Path: "ls",
					Args: []string{"-l", "/"},
				}, garden.ProcessIO{
					Stdout: io.MultiWriter(GinkgoWriter, stdout),
					Stderr: GinkgoWriter,
				})

				Expect(err).ToNot(HaveOccurred())

				process.Wait()
				Expect(stdout).To(gbytes.Say("foo"))
			})
		})
	})
})
