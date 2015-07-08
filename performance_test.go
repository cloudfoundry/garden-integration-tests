package garden_integration_tests_test

import (
	"io"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("performance", func() {

	BeforeEach(func() {
		rootfs = "docker:///cloudfoundry/ubuntu-bc"
	})

	Describe("a process inside a container", func() {
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
