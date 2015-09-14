package garden_integration_tests_test

import (
	"github.com/cloudfoundry-incubator/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("smoke tests", func() {
	FIt("can run a process inside a container", func() {
		stdout := gbytes.NewBuffer()

		_, err := container.Run(garden.ProcessSpec{
			Path: "/usr/bin/whoami",
			User: "root",
		}, garden.ProcessIO{
			Stdout: stdout,
			Stderr: GinkgoWriter,
		})

		Expect(err).ToNot(HaveOccurred())
		Eventually(stdout).Should(gbytes.Say("root\n"))
	})
})
