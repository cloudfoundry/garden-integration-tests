package garden_integration_tests_test

import (
	"code.cloudfoundry.org/garden"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("smoke tests", func() {
	It("can run a process inside a container", func() {
		stdout := runForStdout(container, garden.ProcessSpec{
			Path: "whoami",
			User: "root",
		})

		Expect(stdout).To(gbytes.Say("root\n"))
	})
})
