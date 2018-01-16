package edgecase_test

import (
	"sync"
	"time"

	"code.cloudfoundry.org/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// These tests are slow, and prone to false positives if we were to regress on
// the functionality under test.
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

	AfterEach(func() {
		Expect(gardenClient.Destroy(container.Handle())).To(Succeed())
	})

	Context("when creating a pea and destroying the sandbox at the same time", func() {
		test := func(wg *sync.WaitGroup, peaCreationDuration time.Duration) {
			defer wg.Done()
			ctr, err := gardenClient.Create(garden.ContainerSpec{
				Image: peaImage,
			})
			Expect(err).NotTo(HaveOccurred())

			go ctr.Run(
				garden.ProcessSpec{
					Path:  "sleep",
					Args:  []string{"30"},
					Image: peaImage,
				},
				garden.ProcessIO{
					Stdout: GinkgoWriter,
					Stderr: GinkgoWriter,
				},
			)

			doneDeleting := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				defer close(doneDeleting)
				time.Sleep(peaCreationDuration)
				Expect(gardenClient.Destroy(ctr.Handle())).To(Succeed())
			}()

			<-doneDeleting
		}

		// https://www.pivotaltracker.com/story/show/154242239
		It("is able to destroy", func() {
			start := time.Now()
			_, err := container.Run(
				garden.ProcessSpec{
					Path:  "sleep",
					Args:  []string{"30"},
					Image: peaImage,
				},
				garden.ProcessIO{
					Stdout: GinkgoWriter,
					Stderr: GinkgoWriter,
				},
			)
			peaCreationDuration := time.Since(start)
			Expect(err).NotTo(HaveOccurred())

			for i := 0; i < 5; i++ {
				wg := &sync.WaitGroup{}
				for i := 0; i < 50; i++ {
					wg.Add(1)
					go test(wg, peaCreationDuration)
				}
				wg.Wait()
			}
		})
	})
})
