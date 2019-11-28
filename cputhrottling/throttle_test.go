package cputhrottling_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"code.cloudfoundry.org/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("throttle tests", func() {
	var (
		badContainer      garden.Container
		goodContainer     garden.Container
		badContainerPort  uint32
		goodContainerPort uint32
	)

	JustBeforeEach(func() {
		var err error

		badContainer, err = gardenClient.Create(garden.ContainerSpec{
			Image: garden.ImageRef{URI: "docker:///cfgarden/throttled-or-not"},
		})
		Expect(err).NotTo(HaveOccurred())

		badContainerPort, _, err = badContainer.NetIn(0, 8080)
		Expect(err).NotTo(HaveOccurred())

		_, err = badContainer.Run(garden.ProcessSpec{Path: "/go/src/app/main"}, garden.ProcessIO{})
		Expect(err).NotTo(HaveOccurred())

		goodContainer, err = gardenClient.Create(garden.ContainerSpec{
			Image: garden.ImageRef{URI: "docker:///cfgarden/throttled-or-not"},
		})
		Expect(err).NotTo(HaveOccurred())

		goodContainerPort, _, err = goodContainer.NetIn(0, 8080)
		Expect(err).NotTo(HaveOccurred())

		_, err = goodContainer.Run(garden.ProcessSpec{Path: "/go/src/app/main"}, garden.ProcessIO{})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(gardenClient.Destroy(badContainer.Handle())).To(Succeed())
		Expect(gardenClient.Destroy(goodContainer.Handle())).To(Succeed())
	})

	It("will eventually throttle the 'good' app", func() {
		Expect(spin(badContainerPort)).To(Succeed())
		// Wait for the bad container to enter the "bad" cgroup
		time.Sleep(9 * time.Second)

		Expect(spin(goodContainerPort)).To(Succeed())
		time.Sleep(1 * time.Second)
		initalAvg, err := getLastAverage(goodContainerPort)
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() (float64, error) {
			currAvg, err := getLastAverage(goodContainerPort)
			if err != nil {
				return 0, err
			}
			return currAvg / initalAvg, nil
		}, "2m").Should(BeNumerically("<", 0.5))

		Expect(goodContainerPort).To(Equal(badContainerPort))
	})
})

func spin(containerPort uint32) error {
	if _, err := httpGet(fmt.Sprintf("http://%s:%d/spin", gardenHost, containerPort)); err != nil {
		return fmt.Errorf("spin failed: %+v", err)
	}

	return nil
}

func getLastAverage(containerPort uint32) (float64, error) {
	resp, err := httpGet(fmt.Sprintf("http://%s:%d/lastavg", gardenHost, containerPort))
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(resp, 64)
}

func httpGet(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}
