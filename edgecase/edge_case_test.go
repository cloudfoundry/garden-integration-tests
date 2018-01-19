package edgecase_test

import (
	"time"

	"code.cloudfoundry.org/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// This test require a hacked version of runc to run.
// The hacked runc is configured to misbehave in certain circumstances,
// specifically to "hang" (i.e. sleep) before opening the exec fifo.
// This allows us to test a race condition during pea creation and container
// destruction.
// See the "gats-edgecases.yml" CI task for details on how the hacked runc is
// built.
// As a result of requiring a hacked runc, the edge case suite cannot easily
// be run locally without manual intervention.
var _ = Describe("Edge cases", func() {
	var (
		container garden.Container
		peaImage  = garden.ImageRef{URI: "docker:///alpine#3.7"}
	)

	BeforeEach(func() {
		var err error
		container, err = gardenClient.Create(garden.ContainerSpec{})
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("race condition when creating a pea and destroying the sandbox", func() {
		BeforeEach(func() {
			// download the pea image to ensure it's in the cache
			ctr, err := gardenClient.Create(garden.ContainerSpec{Image: peaImage})
			Expect(err).NotTo(HaveOccurred())
			Expect(gardenClient.Destroy(ctr.Handle())).To(Succeed())
		})

		It("doesn't hang and/or return 'container init still running' error", func() {
			doneRunningPea := make(chan struct{})
			go func() {
				// this will hang due to misbehaving runc
				container.Run(
					garden.ProcessSpec{
						Path:  "sh",
						Args:  []string{"-c", "echo i-am-pea"},
						Image: peaImage,
					},
					garden.ProcessIO{
						Stdout: GinkgoWriter,
						Stderr: GinkgoWriter,
					})
				close(doneRunningPea)
			}()

			time.Sleep(time.Second * 3) // ensure enough time for the runc process to have been started
			Expect(gardenClient.Destroy(container.Handle())).To(Succeed())

			select {
			case <-doneRunningPea:
			case <-time.After(time.Second * 1000):
				Fail("pea creation hung")
			}
		})
	})
})
