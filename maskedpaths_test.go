package garden_integration_tests_test

import (
	"fmt"

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
				stdout := runForStdout(container, garden.ProcessSpec{
					Path: "stat",
					Args: []string{file},
				})
				Expect(stdout).To(gbytes.Say("character special file"))
				Expect(stdout).To(gbytes.Say("Device type: 1,3"))
			}
		})

		// None of our stemells that we test on have /proc/latency_stats, so runc
		// will not normally have to mask it.
		// Still, this has use as a regression test if run against a deployment
		// with this feature enabled.
		It("does not provide access to /proc/latency_stats", func() {
			exitCode, stdout, _ := runProcess(container, garden.ProcessSpec{
				Path: "sh",
				Args: []string{"-c", "if [ -e /proc/latency_stats ]; then stat /proc/latency_stats ; else echo notexist && exit 42; fi"},
			})

			if exitCode == 0 {
				Expect(stdout).To(gbytes.Say("character special file"))
				Expect(stdout).To(gbytes.Say("Device type: 1,3"))
			} else if exitCode == 42 {
				Expect(stdout).To(gbytes.Say("notexist\n"))
			} else {
				Fail(fmt.Sprintf("unexpected exit status %d", exitCode))
			}
		})

		It("masks certain dirs in /proc", func() {
			dirs := []string{
				"/proc/scsi",
			}
			for _, dir := range dirs {
				stdout := runForStdout(container, garden.ProcessSpec{
					Path: "ls",
					Args: []string{"-A", dir},
				})
				Expect(stdout.Contents()).To(BeEmpty(), "directory %v is not empty", dir)
			}
		})
	})
})
