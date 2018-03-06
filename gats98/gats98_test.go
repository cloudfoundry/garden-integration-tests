package gats98_test

import (
	"code.cloudfoundry.org/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Windows", func() {
	var (
		container garden.Container
	)

	AfterEach(func() {
		if container != nil {
			// ignoring the error since it can return unknown handle error
			theContainer, _ := gardenClient.Lookup(container.Handle())

			if theContainer != nil {
				Expect(gardenClient.Destroy(container.Handle())).To(Succeed())
			}
		}
	})

	Context("creating a container", func() {
		It("succeeds and can run a process", func() {
			var err error
			container, err = gardenClient.Create(garden.ContainerSpec{
				Handle:     "",
				Image:      testImage,
				Privileged: false,
				Properties: garden.Properties{},
				Env:        []string{},
				Limits:     garden.Limits{},
				Network:    "",
			})
			Expect(err).NotTo(HaveOccurred())

			exitCode, stdout, _ := runProcess(container, garden.ProcessSpec{
				User: "",
				Path: "cmd.exe",
				Args: []string{"/C", `echo hello`},
			})

			Expect(exitCode).To(Equal(0))
			Expect(stdout).To(gbytes.Say("hello"))
		})
	})
})
