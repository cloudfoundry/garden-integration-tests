package performance_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
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

var dogURL = "https://app.datadoghq.com/api/v1/series?api_key=" + os.Getenv("DATADOG_API_KEY")

func emitMetric(req interface{}) {
	buf, err := json.Marshal(req)
	if err != nil {
		Fail("cannot-marshal-metric: " + err.Error())
		return
	}
	_, err = http.Post(dogURL, "application/json", bytes.NewReader(buf))
	if err != nil {
		Fail("cannot-emit-metric: " + err.Error())
		return
	}
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
					"tags": []string{"deployment:garden-garden"},
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

var _ = Describe("performance", func() {
	Describe("creating", func() {
		Measure("multiple concurrent creates", func(b Benchmarker) {
			// make sure we're warmed up and hitting the cache
			for i := 0; i < 5; i++ {
				ctr, err := gardenClient.Create(garden.ContainerSpec{})
				Expect(err).ToNot(HaveOccurred())
				Expect(gardenClient.Destroy(ctr.Handle())).To(Succeed())
			}

			handles := []string{}
			b.Time("concurrent creations", func() {
				chans := []chan string{}
				for i := 0; i < 5; i++ {
					ch := make(chan string, 1)
					go func(c chan string, index int) {
						defer GinkgoRecover()
						b.Time(fmt.Sprintf("create-%d", index), func() {
							ctr, err := gardenClient.Create(garden.ContainerSpec{})
							Expect(err).ToNot(HaveOccurred())
							c <- ctr.Handle()
						})
					}(ch, i)
					chans = append(chans, ch)
				}

				for _, ch := range chans {
					handle := <-ch
					if handle != "" {
						handles = append(handles, handle)
					}
				}
			})

			for _, handle := range handles {
				Expect(gardenClient.Destroy(handle)).To(Succeed())
			}
		}, 50)

		Measure("stream bytes in", func(b Benchmarker) {
			// make sure we're warmed up and hitting the cache
			for i := 0; i < 5; i++ {
				ctr, err := gardenClient.Create(garden.ContainerSpec{})
				Expect(err).ToNot(HaveOccurred())
				Expect(gardenClient.Destroy(ctr.Handle())).To(Succeed())
			}
			By("starting")

			b.Time("concurrent streamings", func() {
				chans := []chan struct{}{}
				for i := 0; i < 3; i++ {
					ch := make(chan struct{}, 1)
					chans = append(chans, ch)

					go func(c chan struct{}, index int) {
						defer GinkgoRecover()

						createAndStream(index, b)
						createAndStream(index, b)

						c <- struct{}{}
					}(ch, i)
				}

				for _, ch := range chans {
					<-ch
				}
			})
		}, 10)
	})

	Describe("streaming", func() {
		BeforeEach(func() {
			rootfs = "docker:///cloudfoundry/garden-busybox"
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

		Measure("concurrent streaming to multiple containers", func(b Benchmarker) {
			fmt.Fprintf(GinkgoWriter, "about to create containers\n")
			// create some containers, in parallel for speed
			chans := []chan string{}
			for i := 0; i < 50; i++ {
				ch := make(chan string, 1)
				go func(c chan string, index int) {
					defer GinkgoRecover()
					ctr, err := gardenClient.Create(garden.ContainerSpec{
						Privileged: true,
					})
					Expect(err).ToNot(HaveOccurred())
					c <- ctr.Handle()
				}(ch, i)
				chans = append(chans, ch)
			}

			// collect the container handles
			handles := []string{}
			for _, ch := range chans {
				handle := <-ch
				if handle != "" {
					handles = append(handles, handle)
				}
			}
			fmt.Fprintf(GinkgoWriter, "containers created\n")

			// measure streaming data to the containers
			b.Time("concurrent streaming", func() {
				startingGate := &sync.WaitGroup{}
				startingGate.Add(1)

				streamChans := []chan struct{}{}
				for _, handle := range handles {
					ch := make(chan struct{}, 1)
					streamChans = append(streamChans, ch)
					go func(ch chan struct{}, handle string) {
						defer GinkgoRecover()
						ctr, err := gardenClient.Lookup(handle)
						Expect(err).ToNot(HaveOccurred())

						fmt.Fprintf(GinkgoWriter, "at the starting gate for container %s\n", handle)
						startingGate.Wait()

						fmt.Fprintf(GinkgoWriter, "about to stream in to container %s\n", handle)

						// Stream in a tar file to ctr
						var tarStream io.Reader

						pwd, err := os.Getwd()
						Expect(err).ToNot(HaveOccurred())
						tgzPath := path.Join(pwd, "../resources/dora.tgz")
						tgz, err := os.Open(tgzPath)
						Expect(err).ToNot(HaveOccurred())
						defer tgz.Close()

						tarStream, err = gzip.NewReader(tgz)
						Expect(err).ToNot(HaveOccurred())

						Expect(ctr.StreamIn(garden.StreamInSpec{
							User:      "root",
							Path:      "/root/xxx",
							TarStream: tarStream,
						})).To(Succeed())

						fmt.Fprintf(GinkgoWriter, "successfully streamed in to container %s\n", handle)
						ch <- struct{}{}
					}(ch, handle)
				}

				startingGate.Done()

				fmt.Fprintf(GinkgoWriter, "about to wait for streaming to end\n")
				for _, ch := range streamChans {
					<-ch
				}
			})

			fmt.Fprintf(GinkgoWriter, "about to destroy containers\n")

			for _, handle := range handles {
				Expect(gardenClient.Destroy(handle)).To(Succeed())
			}
		}, 10)
	})

	Describe("a process inside a container", func() {
		BeforeEach(func() {
			rootfs = "docker:///cloudfoundry/ubuntu-bc"
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
