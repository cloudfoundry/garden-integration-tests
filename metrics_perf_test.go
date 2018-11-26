package garden_integration_tests_test

import (
	"fmt"
	"sync"
	"time"

	"code.cloudfoundry.org/garden"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metrics performance", func() {
	BeforeEach(func() {
		fmt.Println("BeforeEach: creating 200 containers...")
		preheatServer(200)
		fmt.Println("BeaforeEach: done")
	})

	AfterEach(func() {
		fmt.Println("AfterEach: destroying containers...")
		cleanupContainers()
		fmt.Println("AfterEach: done")
	})

	FIt("returns bulk metrics", func() {
		allContainerIds := getAllContainerIds(gardenClient)
		Expect(len(allContainerIds)).To(Equal(201))

		cycles := 5

		var measurementDuration int64
		measurementDuration = 0
		fmt.Printf("\n\nRunning %d measurement cycles...\n", cycles)
		for i := 0; i < cycles; i++ {
			duration := bulkMetrics(allContainerIds)
			avgDuration := millis(duration.Nanoseconds() / int64(len(allContainerIds)))
			fmt.Printf("Cycle %d: calling bulk metrics took %s, average per container: %fms\n", i, duration, avgDuration)
			measurementDuration += duration.Nanoseconds()
		}

		avgDurationPerCycle := measurementDuration / int64(cycles)
		avgDurationPerContainer := avgDurationPerCycle / int64(len(allContainerIds))
		fmt.Printf("\n\nEnd results: calling bulk metrics took average %fs, average per container: %fms\n\n\n", seconds(avgDurationPerCycle), millis(avgDurationPerContainer))
	})
})

func bulkMetrics(containerIds []string) time.Duration {
	start := time.Now()
	_, err := gardenClient.BulkMetrics(containerIds)
	ExpectWithOffset(-1, err).NotTo(HaveOccurred())
	return time.Since(start)
}

func cleanupContainers() {
	allContainerIds := getAllContainerIds(gardenClient)
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(allContainerIds))

	for _, containerId := range allContainerIds {
		go func(cId string) {
			defer GinkgoRecover()
			defer waitGroup.Done()
			Expect(gardenClient.Destroy(cId)).To(Succeed())
		}(containerId)
	}
	waitGroup.Wait()
}

func preheatServer(total int) {
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(total)

	for i := 0; i < total; i++ {
		go func() {
			defer GinkgoRecover()
			defer waitGroup.Done()

			_, err := gardenClient.Create(garden.ContainerSpec{})
			Expect(err).NotTo(HaveOccurred())
		}()
	}

	waitGroup.Wait()
}

func getAllContainerIds(gardenClient garden.Client) []string {
	allContainers, err := gardenClient.Containers(nil)
	ExpectWithOffset(-1, err).NotTo(HaveOccurred())

	allContainerIds := []string{}
	for _, c := range allContainers {
		allContainerIds = append(allContainerIds, c.Handle())
	}

	return allContainerIds
}

func millis(nanos int64) float64 {
	return float64(nanos) / float64(time.Millisecond)
}

func seconds(nanos int64) float64 {
	return float64(nanos) / float64(time.Second)
}
