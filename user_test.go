package garden_integration_tests_test

import (
	"strconv"

	"code.cloudfoundry.org/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("users", func() {
	It("has a sufficiently large UID/GID range", func() {
		stdout := runForStdout(container, garden.ProcessSpec{
			User: "1000000000:1000000000",
			Path: "id",
		})
		Expect(stdout).To(gbytes.Say("uid=1000000000 gid=1000000000\n"))
	})

	Context("when creating users", func() {
		BeforeEach(func() {
			imageRef.URI = "docker:///cfgarden/garden-busybox"
		})

		It("creates a user with a large uid and gid", func() {
			uid := 700000
			gid := 700000

			exitCode, _, _ := runProcess(container, garden.ProcessSpec{
				User: "root",
				Path: "addgroup",
				Args: []string{"-g", strconv.Itoa(gid), "bob"},
			})
			Expect(exitCode).To(Equal(0))

			exitCode, _, _ = runProcess(container, garden.ProcessSpec{
				User: "root",
				Path: "adduser",
				Args: []string{"-u", strconv.Itoa(uid), "-G", "bob", "-D", "bob"},
			})
			Expect(exitCode).To(Equal(0))

			exitCode, _, _ = runProcess(container, garden.ProcessSpec{
				User: "bob",
				Path: "echo",
				Args: []string{"Hello Baldrick"},
			})
			Expect(exitCode).To(Equal(0))
		})
	})

	Context("when rootfs defines user/groups", func() {
		BeforeEach(func() {
			imageRef.URI = "docker:///cfgarden/with-user-with-groups"
		})

		It("ignores additional groups", func() {
			stdout := runForStdout(container, garden.ProcessSpec{
				User: "alice",
				Path: "cat",
				Args: []string{"/proc/self/status"},
			})

			Expect(stdout).To(gbytes.Say("Groups:\t\n"))
			Expect(stdout).NotTo(gbytes.Say("1010"))
			Expect(stdout).NotTo(gbytes.Say("1011"))
		})
	})
})
