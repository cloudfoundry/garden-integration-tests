package garden_integration_tests_test

import (
	"time"

	"code.cloudfoundry.org/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = FDescribe("CPU burst tests", func() {
	It("can burst", func() {
		containerMiner, err := gardenClient.Create(garden.ContainerSpec{
			Handle:  "miner",
			Limits:  garden.Limits{CPU: garden.CPULimits{LimitInShares: 100}},
			Network: networkSpec,
		})
		Expect(err).NotTo(HaveOccurred())

		_, err = containerMiner.Run(garden.ProcessSpec{
			ID:   "miner",
			Path: "sh",
			Args: []string{
				// "-c", `while true; do (yes>/dev/null&) ; sleep 5; killall yes; sleep 1; done`,
				"-c", `yes>/dev/null`,
			},
		}, garden.ProcessIO{})
		Expect(err).NotTo(HaveOccurred())

		containerExtra, err := gardenClient.Create(garden.ContainerSpec{
			Handle:  "extra",
			Limits:  garden.Limits{CPU: garden.CPULimits{LimitInShares: 200}},
			Network: networkSpec,
		})
		Expect(err).NotTo(HaveOccurred())
		_, err = containerExtra.Run(garden.ProcessSpec{
			ID:   "extra",
			Path: "sh",
			Args: []string{
				"-c", `while true; do /bin/true; sleep 1; done`,
			},
		}, garden.ProcessIO{})
		Expect(err).NotTo(HaveOccurred())

		burstControl()
	})

	AfterEach(func() {
		Expect(gardenClient.Destroy("miner")).To(Succeed())
		Expect(gardenClient.Destroy("extra")).To(Succeed())
	})
})

func burstControl() {
	sampleInterval := 5 * time.Second

	containers, err := gardenClient.Containers(nil)
	Expect(err).NotTo(HaveOccurred())
	bcdata := initBurstControlData(containers)

	for {
		time.Sleep(sampleInterval)
		bulkUpdateCpuUsage(bcdata)

		for handle, containerData := range bcdata {
			cpuUsageDuringLastInterval := float64(containerData.currentCpuUsage-containerData.previousCpuUsage) / float64(sampleInterval)

			if containerData.cappedUntil.IsZero() && cpuUsageDuringLastInterval > containerData.cpuUsageBaseLine {
				throttleContainer(handle, containerData.cpuUsageBaseLine)
				containerData.cappedUntil = time.Now().Add(2 * sampleInterval)
			}

			if !containerData.cappedUntil.IsZero() && time.Now().After(containerData.cappedUntil) {
				unthrottleContainer(handle)
				containerData.cappedUntil = time.Time{}
			}
		}
	}
}

func ptr(i uint64) *uint64 {
	return &i
}

func sumCpuShares(containers []garden.Container) uint64 {
	var totalContainersMemory uint64
	for _, c := range containers {
		totalContainersMemory += getCurrentCpuShares(c)
	}
	return totalContainersMemory
}

func getCurrentCpuShares(container garden.Container) uint64 {
	cpuLimits, err := container.CurrentCPULimits()
	Expect(err).NotTo(HaveOccurred())
	return cpuLimits.LimitInShares
}

func initBurstControlData(containers []garden.Container) map[string]*burstControlData {
	bcdMap := make(map[string]*burstControlData)
	totalContainersCpuShares := sumCpuShares(containers)
	for _, c := range containers {
		bl := float64(getCurrentCpuShares(c)) / float64(totalContainersCpuShares)
		if bl > 0 {
			bcdMap[c.Handle()] = &burstControlData{
				cpuUsageBaseLine: float64(getCurrentCpuShares(c)) / float64(totalContainersCpuShares),
			}
		}
	}

	return bcdMap
}

func getCpuUsage(handle string) uint64 {
	c, err := gardenClient.Lookup(handle)
	Expect(err).NotTo(HaveOccurred())
	metrics, err := c.Metrics()
	Expect(err).NotTo(HaveOccurred())
	return metrics.CPUStat.Usage
}

func bulkUpdateCpuUsage(bcdata map[string]*burstControlData) {
	for k, v := range bcdata {
		v.previousCpuUsage = v.currentCpuUsage
		v.currentCpuUsage = getCpuUsage(k)
	}
}

func throttleContainer(handle string, baseline float64) {
	c, err := gardenClient.Lookup(handle)
	Expect(err).NotTo(HaveOccurred())
	period := uint64(100000)
	quota := int64(float64(period) * baseline)
	err = c.UpdateLimits(garden.Limits{CPU: garden.CPULimits{Quota: quota, Period: period}})
	Expect(err).NotTo(HaveOccurred())
}

func unthrottleContainer(handle string) {
	c, err := gardenClient.Lookup(handle)
	Expect(err).NotTo(HaveOccurred())
	err = c.UpdateLimits(garden.Limits{CPU: garden.CPULimits{Quota: -1}})
	Expect(err).NotTo(HaveOccurred())
}

type burstControlData struct {
	cpuUsageBaseLine float64 // 0 - 1
	currentCpuUsage  uint64
	previousCpuUsage uint64
	cappedUntil      time.Time
}
