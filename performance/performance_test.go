package performance_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

const containerPrefix = "concurrent-create-handle"

var dogURL = "https://app.datadoghq.com/api/v1/series?api_key=" + os.Getenv("DATADOG_API_KEY")

var _ = Describe("performance", func() {
	JustBeforeEach(func() {
		warmUp(gardenClient)
	})

	Measure("multiple concurrent creates", func(b Benchmarker) {
		concurrencyLevel := 5
		handles := []string{}

		b.Time("concurrent creations", func() {
			wg := sync.WaitGroup{}

			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				h := fmt.Sprintf("%s-%d", containerPrefix, i)
				handles = append(handles, h)

				go func(index int, handle string) {
					defer wg.Done()
					defer GinkgoRecover()

					b.Time(fmt.Sprintf("create-%d", index), func() {
						_, err := gardenClient.Create(garden.ContainerSpec{Handle: handle})
						Expect(err).ToNot(HaveOccurred())
					})
				}(i, h)
			}

			wg.Wait()
		})

		b.Time("destroy", func() {
			for _, handle := range handles {
				b.Time(fmt.Sprintf("destroy-%s", handle), func() {
					Expect(gardenClient.Destroy(handle)).To(Succeed())
				})
			}

			// ensure all containers are actually destroyed
			Eventually(func() error {
				for _, handle := range handles {
					_, err := gardenClient.Lookup(handle)
					if err == nil {
						return errors.New(fmt.Sprintf("container '%s' exists but it should've been destroyed", handle))
					}
				}
				return nil
			}).ShouldNot(HaveOccurred())
		})
	}, 50)

	Measure("stream bytes in", func(b Benchmarker) {
		concurrenyLevel := 5
		By("starting")

		b.Time("concurrent streamings", func() {
			wg := sync.WaitGroup{}

			for i := 0; i < concurrenyLevel; i++ {
				wg.Add(1)

				go func(index int) {
					defer wg.Done()
					defer GinkgoRecover()

					// do it twice in a row to increase likelihood of overlaps
					createAndStream(index, b)
					createAndStream(index, b)
				}(i)
			}

			wg.Wait()
		})
	}, 10)

	Describe("streaming", func() {
		BeforeEach(func() {
			rootfs = "docker:///cfgarden/garden-busybox"
		})

		Measure("it should stream stdout and stderr efficiently", func(b Benchmarker) {
			b.Time("(baseline) streaming 50M of stdout to /dev/null", func() {
				stdout := gbytes.NewBuffer()
				stderr := gbytes.NewBuffer()

				_, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "sh",
					Args: []string{"-c", "tr '\\0' 'a' < /dev/zero | dd count=50 bs=1M of=/dev/null; echo done"},
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: stderr,
				})
				Expect(err).ToNot(HaveOccurred())

				Eventually(stdout, "2s").Should(gbytes.Say("done\n"))
			})

			time := b.Time("streaming 50M of data via garden", func() {
				stdout := gbytes.NewBuffer()
				stderr := gbytes.NewBuffer()

				_, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "sh",
					Args: []string{"-c", "tr '\\0' 'a' < /dev/zero | dd count=50 bs=1M; echo done"},
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: stderr,
				})
				Expect(err).ToNot(HaveOccurred())

				Eventually(stdout, "10s").Should(gbytes.Say("done\n"))
			})

			Expect(time.Seconds()).To(BeNumerically("<", 3))
		}, 10)
	})

	Describe("a process inside a container", func() {
		BeforeEach(func() {
			rootfs = "docker:///cfgarden/ubuntu-bc"
		})

		Measure("starting lots of processes", func(b Benchmarker) {
			b.Time("end to end time", func() {
				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "bash",
					Args: []string{"-c", `
					for i in {1..1000}
					do
						/bin/echo hi > /dev/null
					done
				`},
				}, garden.ProcessIO{
					Stdout: GinkgoWriter,
					Stderr: GinkgoWriter,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(process.Wait()).To(Equal(0))
			})

			// TODO add expectations to avoid regression
		}, 20)

		Measure("running a calculation", func(b Benchmarker) {
			stderr := gbytes.NewBuffer()
			b.Time("end to end time", func() {
				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "bash",
					Args: []string{
						"-c",
						`time echo "scale=1000; a(1)*4" | bc -l`,
					},
				}, garden.ProcessIO{
					Stdout: GinkgoWriter,
					Stderr: io.MultiWriter(stderr, GinkgoWriter),
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(process.Wait()).To(Equal(0))
			})

			timeTaken := func(lines string) string {
				for _, line := range strings.Split(lines, "\n") {
					cols := strings.Fields(line)
					if len(cols) < 2 {
						continue
					}
					if cols[0] == "user" {
						return cols[1]
					}
				}
				return "error!"
			}

			dur, err := time.ParseDuration(timeTaken(string(stderr.Contents())))
			Expect(err).NotTo(HaveOccurred())

			b.RecordValue("time in calculation", dur.Seconds())

			// Once we have a good baseline...
			//Expect(timed).To(BeNumerically(",", ???))
			//Expect(b.Seconds()).To(BeNumerically(",", ???))
		}, 20)
	})
})

func emitMetric(req interface{}) {
	if os.Getenv("DATADOG_API_KEY") == "" {
		Fail("DATADOG_API_KEY not set!")
	}
	buf, err := json.Marshal(req)
	if err != nil {
		Fail("cannot-marshal-metric: " + err.Error())
		return
	}
	response, err := http.Post(dogURL, "application/json", bytes.NewReader(buf))
	if err != nil {
		Fail("cannot-emit-metric: " + err.Error())
		return
	}
	if response.StatusCode != http.StatusAccepted {
		Fail(fmt.Sprintf("cannot-emit-metric: error code not 202: %d %s", response.StatusCode, response.Status))
		return
	}
}

func warmUp(gardenClient garden.Client) {
	ctr, err := gardenClient.Create(garden.ContainerSpec{})
	Expect(err).ToNot(HaveOccurred())
	Expect(gardenClient.Destroy(ctr.Handle())).To(Succeed())
}

func streaminDora(ctr garden.Container) {
	for i := 0; i < 20; i++ {
		By(fmt.Sprintf("preparing stream %d for handle %s", i, ctr.Handle()))
		// Stream in a tar file to ctr
		var tarStream io.Reader

		pwd, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		tgzPath := path.Join(pwd, "../resources/dora.tgz")
		tgz, err := os.Open(tgzPath)
		Expect(err).ToNot(HaveOccurred())
		tarStream, err = gzip.NewReader(tgz)
		Expect(err).ToNot(HaveOccurred())

		By(fmt.Sprintf("starting stream %d for handle: %s", i, ctr.Handle()))
		Expect(ctr.StreamIn(garden.StreamInSpec{
			User:      "root",
			Path:      fmt.Sprintf("/root/stream-file-%d", i),
			TarStream: tarStream,
		})).To(Succeed())
		By(fmt.Sprintf("stream %d done for handle: %s", i, ctr.Handle()))

		tgz.Close()
	}
}

func createAndStream(index int, b Benchmarker) {
	var handle string
	var ctr garden.Container
	var err error

	b.Time(fmt.Sprintf("stream-%d", index), func() {
		creationTime := b.Time(fmt.Sprintf("create-%d", index), func() {
			By("creating container " + strconv.Itoa(index))
			ctr, err = gardenClient.Create(garden.ContainerSpec{
				Limits: garden.Limits{
					Disk: garden.DiskLimits{ByteHard: 2 * 1024 * 1024 * 1024},
				},
				Privileged: true,
			})
			Expect(err).ToNot(HaveOccurred())
			handle = ctr.Handle()
			By("done creating container " + strconv.Itoa(index))
		})
		now := time.Now()
		emitMetric(map[string]interface{}{
			"series": []map[string]interface{}{
				{
					"metric": "garden.container-creation-time",
					"points": [][]int64{
						{now.Unix(), int64(creationTime)},
					},
					"tags": []string{"deployment:" + os.Getenv("ENVIRONMENT") + "-garden"},
				},
			},
		})

		By("starting stream in to container " + handle)

		streaminDora(ctr)

		By("succefully streamed in to container " + handle)

		b.Time(fmt.Sprintf("delete-%d", index), func() {
			By("destroying container " + handle)
			Expect(gardenClient.Destroy(handle)).To(Succeed())
			By("successfully destroyed container " + handle)
		})
	})
}
