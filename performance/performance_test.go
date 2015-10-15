package performance_test

import (
	"io"
	"fmt"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("performance", func() {
	FDescribe("creating", func() {
		Measure("multiple concurrent creates", func(b Benchmarker) {
		 gardenClient.Create(garden.ContainerSpec{}) // make sure we're hitting cache

			handles := []string{}
			b.Time("concurrent creations", func() {
				chans := []chan string{}
				for i := 0; i < 50; i++ {
					ch := make(chan string, 1)
					go func(c chan string, index int) {
						defer GinkgoRecover()
						b.Time(fmt.Sprintf("create-%d",index), func() {
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
		}, 2)
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
