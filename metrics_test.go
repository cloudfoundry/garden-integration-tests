package garden_integration_tests_test

import (
	"time"

	"code.cloudfoundry.org/garden"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metrics", func() {
	JustBeforeEach(func() {
		skipIfWoot("Groot does not support metrics yet")
		_, err := container.Run(garden.ProcessSpec{
			Path: "sh",
			Args: []string{
				"-c", `while true; do ls -la; done`,
			},
		}, garden.ProcessIO{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns the CPU metrics", func() {
		Eventually(func() uint64 {
			return metrics(container).CPUStat.Usage
		}).ShouldNot(BeZero())
	})

	It("returns the memory metrics", func() {
		Eventually(func() uint64 {
			return metrics(container).MemoryStat.TotalUsageTowardLimit
		}).ShouldNot(BeZero())
	})

	It("returns the number of currently running pids", func() {
		Eventually(func() uint64 {
			return metrics(container).PidStat.Current
		}).ShouldNot(BeZero())
	})

	It("returns an N/A value for the max mumber of pids", func() {
		Consistently(func() uint64 {
			return metrics(container).PidStat.Max
		}).Should(BeZero())
	})

	Context("when there is a pid limit", func() {
		BeforeEach(func() {
			limits = garden.Limits{
				Pid: garden.PidLimits{Max: 128},
			}
		})

		It("returns the max number of pids", func() {
			Eventually(func() uint64 {
				return metrics(container).PidStat.Max
			}).Should(BeEquivalentTo(128))
		})
	})

	It("returns container age", func() {
		Eventually(func() time.Duration {
			return metrics(container).Age
		}).Should(Not(BeZero()))
	})

	It("has the container handle in bulk metrics", func() {
		metrics, err := gardenClient.BulkMetrics([]string{container.Handle()})
		Expect(err).NotTo(HaveOccurred())
		Expect(metrics).To(HaveKey(container.Handle()))
	})

	It("returns total memory usage for the container in bulk metrics", func() {
		Eventually(func() (uint64, error) {
			metrics, err := gardenClient.BulkMetrics([]string{container.Handle()})
			if err != nil {
				return 0, err
			}

			return metrics[container.Handle()].Metrics.MemoryStat.TotalUsageTowardLimit, nil
		}).ShouldNot(BeZero())
	})
})

func metrics(container garden.Container) garden.Metrics {
	metrics, err := container.Metrics()
	Expect(err).NotTo(HaveOccurred())
	return metrics
}
