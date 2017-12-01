package garden_integration_tests_test

import (
	"fmt"
	"io"

	"code.cloudfoundry.org/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var peaImage = garden.ImageRef{URI: "docker:///alpine#3.6"}
var noImage = garden.ImageRef{}

var _ = Describe("Partially shared containers (peas)", func() {
	Describe("sharing of namespaces", func() {
		It("runs a process that shares all of the namespaces besides the mount one", func() {

			sandboxContainerMntNs := getNS("mnt", container, noImage)
			peaContainerMntNs := getNS("mnt", container, peaImage)
			Expect(sandboxContainerMntNs).NotTo(Equal(peaContainerMntNs))

			for _, ns := range []string{"net", "ipc", "pid", "user", "uts"} {
				sandboxContainerNs := getNS(ns, container, noImage)
				peaContainerNs := getNS(ns, container, peaImage)
				Expect(sandboxContainerNs).To(Equal(peaContainerNs))
			}
		})

		Context("when the sandbox container is privileged", func() {
			BeforeEach(func() {
				setPrivileged()
			})

			It("runs a process that shares all of the namespaces besides the mount one", func() {
				sandboxContainerMntNs := getNS("mnt", container, noImage)
				peaContainerMntNs := getNS("mnt", container, peaImage)
				Expect(sandboxContainerMntNs).NotTo(Equal(peaContainerMntNs))

				for _, ns := range []string{"net", "ipc", "pid", "user", "uts"} {
					sandboxContainerNs := getNS(ns, container, noImage)
					peaContainerNs := getNS(ns, container, peaImage)
					Expect(sandboxContainerNs).To(Equal(peaContainerNs))
				}
			})
		})
	})

	It("runs a process in its own rootfs", func() {
		stdout := runForStdout(container, garden.ProcessSpec{
			Path:  "cat",
			Args:  []string{"/etc/os-release"},
			Image: peaImage,
		})
		Expect(stdout).To(gbytes.Say(`NAME="Alpine Linux"`))
	})

	Describe("pea process user and group", func() {
		It("runs the process as uid and gid 0 by default", func() {
			stdout := runForStdout(container, garden.ProcessSpec{
				Path:  "sh",
				Args:  []string{"-c", "echo -n $(id -u):$(id -g)"},
				Image: peaImage,
			})
			Expect(stdout).To(gbytes.Say("0:0"))
		})

		Context("when a uid:gid is provided", func() {
			It("runs the process as the specified uid and gid", func() {
				userGUIDs := "1001:1002"
				stdout := runForStdout(container, garden.ProcessSpec{
					Path:  "sh",
					Args:  []string{"-c", "echo -n $(id -u):$(id -g)"},
					User:  userGUIDs,
					Image: peaImage,
				})
				Expect(stdout).To(gbytes.Say(userGUIDs))
			})
		})

		Context("when a username is provided", func() {
			It("returns an error", func() {
				_, err := container.Run(garden.ProcessSpec{
					User:  "root",
					Path:  "pwd",
					Image: peaImage,
				}, garden.ProcessIO{
					Stdout: GinkgoWriter,
					Stderr: GinkgoWriter,
				})
				Expect(err).To(MatchError(ContainSubstring("'root' is not a valid uid:gid")))
			})
		})
	})

	Describe("pea process Wait and IO", func() {
		It("returns the process exit code", func() {
			processExitCode, _, _ := runProcess(container, garden.ProcessSpec{
				Path:  "sh",
				Args:  []string{"-c", "exit 123"},
				Image: peaImage,
			})

			Expect(processExitCode).To(Equal(123))
		})

		It("streams stdout and stderr back to the client", func() {
			processExitCode, stdout, stderr := runProcess(container, garden.ProcessSpec{
				Path:  "sh",
				Args:  []string{"-c", "echo stdout && echo stderr >&2"},
				Image: peaImage,
			})

			Expect(processExitCode).To(Equal(0))
			Expect(stdout).To(gbytes.Say("stdout\n"))
			Expect(stderr).To(gbytes.Say("stderr\n"))
		})
	})

	It("bind mounts the same /etc/hosts file as the container", func() {
		originalContentsInContainer := readFileInContainer(container, "/etc/hosts", noImage)
		originalContentsInPea := readFileInContainer(container, "/etc/hosts", peaImage)
		Expect(originalContentsInContainer).To(Equal(originalContentsInPea))

		appendFileInContainer(container, "/etc/hosts", "foobar", peaImage)
		contentsInPea := readFileInContainer(container, "/etc/hosts", peaImage)
		Expect(originalContentsInPea).NotTo(Equal(contentsInPea))

		contentsInContainer := readFileInContainer(container, "/etc/hosts", noImage)
		Expect(contentsInPea).To(Equal(contentsInContainer))
	})

	It("bind mounts the same /etc/resolv.conf file as the container", func() {
		originalContentsInContainer := readFileInContainer(container, "/etc/resolv.conf", noImage)
		originalContentsInPea := readFileInContainer(container, "/etc/resolv.conf", peaImage)
		Expect(originalContentsInContainer).To(Equal(originalContentsInPea))

		appendFileInContainer(container, "/etc/resolv.conf", "foobar", peaImage)
		contentsInPea := readFileInContainer(container, "/etc/resolv.conf", peaImage)
		Expect(originalContentsInPea).NotTo(Equal(contentsInPea))

		contentsInContainer := readFileInContainer(container, "/etc/resolv.conf", noImage)
		Expect(contentsInPea).To(Equal(contentsInContainer))
	})

	Context("when no working directory is specified", func() {
		It("defaults to /", func() {
			stdout := runForStdout(container, garden.ProcessSpec{
				Path:  "pwd",
				Image: peaImage,
			})
			Expect(stdout).To(gbytes.Say("/\n"))
		})
	})

	Describe("signalling", func() {
		It("sends a TERM signal to the process if requested", func() {
			stdout := gbytes.NewBuffer()

			process, err := container.Run(garden.ProcessSpec{
				Path: "sh",
				Args: []string{"-c", `
				trap 'echo termed; exit 42' SIGTERM

				while true; do
					echo waiting
					sleep 1
				done
			`},
				Image: peaImage,
			}, garden.ProcessIO{
				Stdout: io.MultiWriter(GinkgoWriter, stdout),
				Stderr: GinkgoWriter,
			})
			Expect(err).ToNot(HaveOccurred())

			Eventually(stdout).Should(gbytes.Say("waiting"))
			Expect(process.Signal(garden.SignalTerminate)).To(Succeed())
			Eventually(stdout, "2s").Should(gbytes.Say("termed"))
			Expect(process.Wait()).To(Equal(42))
		})
	})

	Describe("Limits", func() {
		BeforeEach(func() {
			limits = garden.Limits{Memory: garden.MemoryLimits{
				LimitInBytes: 64 * 1024 * 1024,
			}}
		})

		Context("when there is no memory limit on the pea", func() {
			It("shares that limit with the container", func() {
				exitCode, _, _ := runProcess(container,
					garden.ProcessSpec{
						Path:  "dd",
						Args:  []string{"if=/dev/urandom", "of=/dev/shm/too-big", "bs=1M", "count=65"},
						Image: peaImage,
					})
				Expect(exitCode).NotTo(Equal(0))
			})
		})

		Context("when there are any limits on the pea", func() {
			It("does not share memory limit with the container", func() {
				exitCode, _, _ := runProcess(container,
					garden.ProcessSpec{
						Path:  "dd",
						Args:  []string{"if=/dev/urandom", "of=/dev/shm/too-big", "bs=1M", "count=65"},
						Image: peaImage,
						OverrideContainerLimits: &garden.ProcessLimits{},
					})
				Expect(exitCode).To(Equal(0))
			})
		})
	})

	Describe("Metrics", func() {
		BeforeEach(func() {
			limits = garden.Limits{Memory: garden.MemoryLimits{
				LimitInBytes: 64 * 1024 * 1024,
			}}
		})

		Context("when there is no memory limit on the pea", func() {
			It("should return bulk metrics", func() {
				buffer := gbytes.NewBuffer()

				proc, err := container.Run(
					garden.ProcessSpec{
						Path:  "sh",
						Args:  []string{"-c", `echo hi && sleep 600`},
						Image: peaImage,
					},
					garden.ProcessIO{
						Stdout: buffer,
						Stderr: GinkgoWriter,
					},
				)
				Expect(err).NotTo(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("hi"))

				handle := proc.ID()
				metrics, err := gardenClient.BulkMetrics([]string{handle})
				Expect(err).NotTo(HaveOccurred())
				Expect(metrics).To(HaveKey(handle))
				Expect(metrics[handle].Err).NotTo(HaveOccurred())
				Expect(metrics[handle].Metrics.MemoryStat.TotalUsageTowardLimit).NotTo(BeZero())
			})
		})
	})

	Context("when the process executable doesn't exist", func() {
		It("returns an error from Run", func() {
			_, err := container.Run(
				garden.ProcessSpec{
					Path:  "does-not-exist",
					Args:  []string{},
					Image: peaImage,
				},
				garden.ProcessIO{
					Stdout: GinkgoWriter,
					Stderr: GinkgoWriter,
				},
			)
			Expect(err).To(MatchError(ContainSubstring("executable file not found in $PATH")))
		})
	})
})

func getNS(nsName string, container garden.Container, image garden.ImageRef) string {
	processSpec := garden.ProcessSpec{
		Path:  "readlink",
		Args:  []string{fmt.Sprintf("/proc/self/ns/%s", nsName)},
		Image: image,
	}

	exitCode, namespaceInode, _ := runProcess(container, processSpec)
	Expect(exitCode).To(Equal(0))

	return string(namespaceInode.Contents())
}

func readFileInContainer(container garden.Container, filePath string, image garden.ImageRef) string {
	exitCode, stdout, _ := runProcess(container, garden.ProcessSpec{
		Path:  "cat",
		Args:  []string{filePath},
		Image: image,
	})
	Expect(exitCode).To(Equal(0))

	return string(stdout.Contents())
}

func appendFileInContainer(container garden.Container, filePath, content string, image garden.ImageRef) {
	exitCode, _, _ := runProcess(container, garden.ProcessSpec{
		Path:  "sh",
		Args:  []string{"-c", fmt.Sprintf("echo %s >> %s", content, filePath)},
		Image: image,
	})
	Expect(exitCode).To(Equal(0))
}
