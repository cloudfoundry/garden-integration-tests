package garden_integration_tests_test

import (
	"bytes"
	"fmt"
	"io"

	"code.cloudfoundry.org/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("MaskedPaths", func() {
	Context("when the container is unprivileged", func() {
		It("masks certain files in /proc with a null character device", func() {
			files := []string{
				"/proc/kcore",
				"/proc/sched_debug",
				"/proc/timer_stats",
				"/proc/timer_list",
			}
			for _, file := range files {
				out := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					Path: "stat",
					Args: []string{file},
				}, garden.ProcessIO{
					Stdout: out,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(process.Wait()).To(Equal(0))

				Expect(out.Contents()).To(ContainSubstring("character special file"))
				Expect(out.Contents()).To(ContainSubstring("Device type: 1,3"))
			}
		})

		// None of our stemells that we test on have /proc/latency_stats, so runc
		// will not normally have to mask it.
		// Still, this has use as a regression test if run against a deployment
		// with this feature enabled.
		It("does not provide access to /proc/latency_stats", func() {
			var stdout bytes.Buffer
			process, err := container.Run(garden.ProcessSpec{
				Path: "sh",
				Args: []string{"-c", "if [ -e /proc/latency_stats ]; then stat /proc/latency_stats ; else echo notexist && exit 42; fi"},
			}, garden.ProcessIO{
				Stdout: io.MultiWriter(&stdout, GinkgoWriter),
				Stderr: GinkgoWriter,
			})
			Expect(err).ToNot(HaveOccurred())
			exitCode, err := process.Wait()
			Expect(err).NotTo(HaveOccurred())

			if exitCode == 0 {
				Expect(stdout.String()).To(ContainSubstring("character special file"))
				Expect(stdout.String()).To(ContainSubstring("Device type: 1,3"))
			} else if exitCode == 42 {
				Expect(stdout.String()).To(Equal("notexist\n"))
			} else {
				Fail(fmt.Sprintf("unexpected exit status %d", exitCode))
			}
		})

		It("masks certain dirs in /proc", func() {
			dirs := []string{
				"/proc/scsi",
			}
			for _, dir := range dirs {
				out := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					Path: "ls",
					Args: []string{"-A", dir},
				}, garden.ProcessIO{
					Stdout: out,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(process.Wait()).To(Equal(0))
				Expect(out.Contents()).To(BeEmpty(), "directory %v is not empty", dir)
			}
		})
	})
})
