package garden_integration_tests_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/guardian/gardener"
	"code.cloudfoundry.org/guardian/rundmc/cgroups"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = FDescribe("CPU Throttling", func() {
	var (
		containerOnePort uint32

		containerTwo     garden.Container
		containerTwoPort uint32
	)

	BeforeEach(func() {
		skipIfCpuThrottlingNotEnabled()

		imageRef = garden.ImageRef{URI: "docker:///cfgarden/throttled-or-not"}
		limits = garden.Limits{CPU: garden.CPULimits{Weight: 100}}
	})

	JustBeforeEach(func() {
		var err error
		containerOnePort, _, err = container.NetIn(0, 8080)
		Expect(err).NotTo(HaveOccurred())
		startSpinnerApp(container, containerOnePort)

		containerTwo, err = gardenClient.Create(garden.ContainerSpec{
			Image:  imageRef,
			Limits: limits,
		})
		Expect(err).NotTo(HaveOccurred())

		containerTwoPort, _, err = containerTwo.NetIn(0, 8080)
		Expect(err).NotTo(HaveOccurred())
		startSpinnerApp(containerTwo, containerTwoPort)

		writeToGinkgo(fmt.Sprintf("ContainerOne handle: %s\n", container.Handle()))
		writeToGinkgo(fmt.Sprintf("ContainerTwo handle: %s\n", containerTwo.Handle()))
	})

	AfterEach(func() {
		Expect(destroyContainer(containerTwo)).To(Succeed())
	})

	Context("CPU-intensive application is punished to the bad cgroup (because it is way over its entitlement)", func() {
		var containerOneInitialUsage float64

		JustBeforeEach(func() {
			var err error
			Expect(spin(container, containerOnePort)).To(Succeed())
			Eventually(punished(container, containerOnePort), "1m").Should(BeTrue())
			containerOneInitialUsage, err = currentUsage(container)()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("and another application wants to spike", func() {
			JustBeforeEach(func() {
				Expect(spin(containerTwo, containerTwoPort)).To(Succeed())
			})

			It("throttles the application that has been spiking so far", func() {
				Eventually(currentUsage(container), "1m").Should(BeNumerically("<", containerOneInitialUsage/2))
			})

			It("allows the previously idle application to spike", func() {
				containerOneCurrentUsage, err := currentUsage(container)()
				Expect(err).NotTo(HaveOccurred())
				Eventually(currentUsage(containerTwo), "1m").Should(BeNumerically(">", containerOneCurrentUsage*2))
			})
		})
	})
})

func skipIfCpuThrottlingNotEnabled() {
	if os.Getenv("CPU_THROTTLING_ENABLED") == "true" {
		return
	}

	Skip("CPU throttling is not enabled")
}

func spin(container garden.Container, containerPort uint32) error {
	if _, err := httpGet(fmt.Sprintf("http://%s:%d/spin", externalIP(container), containerPort)); err != nil {
		return fmt.Errorf("spin failed: %+v", err)
	}

	return nil
}

func externalIP(container garden.Container) string {
	properties, err := container.Properties()
	Expect(err).NotTo(HaveOccurred())
	return properties[gardener.ExternalIPKey]
}

func startSpinnerApp(container garden.Container, containerPort uint32) {
	_, err := container.Run(garden.ProcessSpec{Path: "/go/src/app/main"}, garden.ProcessIO{})
	Expect(err).NotTo(HaveOccurred())

	Eventually(func() (string, error) {
		return httpGet(fmt.Sprintf("http://%s:%d/ping", externalIP(container), containerPort))
	}).Should(Equal("pong"))

	// Wait for the initial spike to be over and make sure the container is in the good cgroup
	Eventually(currentUsage(container), "1m", "1s").Should(BeNumerically("<", 0.01))
	Eventually(punished(container, containerPort)).Should(BeFalse())
}

func getCPUUsageAndEntitlement(container garden.Container) (uint64, uint64, error) {
	metrics, err := container.Metrics()
	if err != nil {
		return 0, 0, err
	}

	return metrics.CPUStat.Usage, metrics.CPUEntitlement, nil
}

func currentUsage(container garden.Container) func() (float64, error) {
	return func() (float64, error) {
		firstUsage, firstEntitlement, err := getCPUUsageAndEntitlement(container)
		if err != nil {
			return 0, nil
		}

		time.Sleep(time.Second)

		secondUsage, secondEntitlement, err := getCPUUsageAndEntitlement(container)
		if err != nil {
			return 0, nil
		}

		deltaUsage := secondUsage - firstUsage
		deltaEntitlement := secondEntitlement - firstEntitlement

		result := float64(deltaUsage) / float64(deltaEntitlement)
		writeToGinkgo(fmt.Sprintf("[%s] usage: %f\n", container.Handle(), result))
		return result, nil
	}
}

func isPunished(container garden.Container, containerPort uint32) (bool, error) {
	cgroup, err := httpGet(fmt.Sprintf("http://%s:%d/cpucgroup", externalIP(container), containerPort))
	if err != nil {
		return false, err
	}

	writeToGinkgo(fmt.Sprintf("[%s] cgroup: %s\n", container.Handle(), cgroup))
	return strings.HasSuffix(cgroup, filepath.Join(cgroups.BadCgroupName, container.Handle())), nil
}

func punished(container garden.Container, containerPort uint32) func() (bool, error) {
	return func() (bool, error) {
		return isPunished(container, containerPort)
	}
}

func writeToGinkgo(message string) {
	_, err := io.WriteString(GinkgoWriter, message)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
}
