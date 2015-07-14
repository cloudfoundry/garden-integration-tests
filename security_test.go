package garden_integration_tests_test

import (
	"io"
	"os"

	"github.com/cloudfoundry-incubator/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Security", func() {
	Context("by default (unprivileged)", func() {
		It("does not get root privileges on host resources", func() {
			process, err := container.Run(garden.ProcessSpec{
				Path: "sh",
				User: "root",
				Args: []string{"-c", "echo h > /proc/sysrq-trigger"},
			}, garden.ProcessIO{})
			Expect(err).ToNot(HaveOccurred())

			Expect(process.Wait()).ToNot(Equal(0))
		})

		It("can write to files in the /root directory", func() {
			process, err := container.Run(garden.ProcessSpec{
				User: "root",
				Path: "sh",
				Args: []string{"-c", `touch /root/potato`},
			}, garden.ProcessIO{})
			Expect(err).ToNot(HaveOccurred())

			Expect(process.Wait()).To(Equal(0))
		})

		Context("with a docker image", func() {
			BeforeEach(func() {
				rootfs = "docker:///cloudfoundry/preexisting_users"
			})

			It("sees root-owned files in the rootfs as owned by the container's root user", func() {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "sh",
					Args: []string{"-c", `ls -l /sbin | grep -v wsh | grep -v hook`},
				}, garden.ProcessIO{Stdout: stdout})
				Expect(err).ToNot(HaveOccurred())

				Expect(process.Wait()).To(Equal(0))
				Expect(stdout).NotTo(gbytes.Say("nobody"))
				Expect(stdout).NotTo(gbytes.Say("65534"))
				Expect(stdout).To(gbytes.Say(" root "))
			})

			It("sees the /dev/pts and /dev/ptmx as owned by the container's root user", func() {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "sh",
					Args: []string{"-c", "ls -l /dev/pts /dev/ptmx"},
				}, garden.ProcessIO{Stdout: stdout, Stderr: GinkgoWriter})
				Expect(err).ToNot(HaveOccurred())

				Expect(process.Wait()).To(Equal(0))
				Expect(stdout).NotTo(gbytes.Say("nobody"))
				Expect(stdout).NotTo(gbytes.Say("65534"))
				Expect(stdout).To(gbytes.Say(" root "))
			})

			if os.Getenv("BTRFS_SUPPORTED") != "" { // VFS driver does not support this feature`
				It("sees the root directory as owned by the container's root user", func() {
					stdout := gbytes.NewBuffer()
					process, err := container.Run(garden.ProcessSpec{
						User: "root",
						Path: "sh",
						Args: []string{"-c", "ls -al / | head -n 2"},
					}, garden.ProcessIO{Stdout: stdout, Stderr: GinkgoWriter})
					Expect(err).ToNot(HaveOccurred())

					Expect(process.Wait()).To(Equal(0))
					Expect(stdout).NotTo(gbytes.Say("nobody"))
					Expect(stdout).NotTo(gbytes.Say("65534"))
					Expect(stdout).To(gbytes.Say(" root "))
				})
			}

			It("sees alice-owned files as owned by alice", func() {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "sh",
					Args: []string{"-c", `ls -l /home/alice`},
				}, garden.ProcessIO{Stdout: stdout})
				Expect(err).ToNot(HaveOccurred())

				Expect(process.Wait()).To(Equal(0))
				Expect(stdout).To(gbytes.Say(" alice "))
				Expect(stdout).To(gbytes.Say(" alicesfile"))
			})

			It("sees devices as owned by root", func() {
				out := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "ls",
					Args: []string{"-la", "/dev/tty"},
				}, garden.ProcessIO{
					Stdout: out,
					Stderr: out,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(process.Wait()).To(Equal(0))
				Expect(string(out.Contents())).To(ContainSubstring(" root "))
				Expect(string(out.Contents())).ToNot(ContainSubstring("nobody"))
				Expect(string(out.Contents())).ToNot(ContainSubstring("65534"))
			})

			It("lets alice write in /home/alice", func() {
				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "touch",
					Args: []string{"/home/alice/newfile"},
				}, garden.ProcessIO{})
				Expect(err).ToNot(HaveOccurred())
				Expect(process.Wait()).To(Equal(0))
			})

			It("lets root write to files in the /root directory", func() {
				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "sh",
					Args: []string{"-c", `touch /root/potato`},
				}, garden.ProcessIO{})
				Expect(err).ToNot(HaveOccurred())
				Expect(process.Wait()).To(Equal(0))
			})

			It("preserves pre-existing dotfiles from base image", func() {
				out := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "cat",
					Args: []string{"/.foo"},
				}, garden.ProcessIO{
					Stdout: out,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(process.Wait()).To(Equal(0))
				Expect(out).To(gbytes.Say("this is a pre-existing dotfile"))
			})
		})
	})

	Context("when the 'privileged' flag is set on the create call", func() {
		BeforeEach(func() {
			privilegedContainer = true
		})

		It("gets real root privileges", func() {
			process, err := container.Run(garden.ProcessSpec{
				Path: "sh",
				User: "root",
				Args: []string{"-c", "echo h > /proc/sysrq-trigger"},
			}, garden.ProcessIO{})
			Expect(err).ToNot(HaveOccurred())

			Expect(process.Wait()).To(Equal(0))
		})

		It("can write to files in the /root directory", func() {
			process, err := container.Run(garden.ProcessSpec{
				User: "root",
				Path: "sh",
				Args: []string{"-c", `touch /root/potato`},
			}, garden.ProcessIO{})
			Expect(err).ToNot(HaveOccurred())

			Expect(process.Wait()).To(Equal(0))
		})

		It("sees root-owned files in the rootfs as owned by the container's root user", func() {
			stdout := gbytes.NewBuffer()
			process, err := container.Run(garden.ProcessSpec{
				User: "root",
				Path: "sh",
				Args: []string{"-c", `ls -l /sbin | grep -v wsh | grep -v hook`},
			}, garden.ProcessIO{Stdout: io.MultiWriter(GinkgoWriter, stdout)})
			Expect(err).ToNot(HaveOccurred())

			Expect(process.Wait()).To(Equal(0))
			Expect(stdout).NotTo(gbytes.Say("nobody"))
			Expect(stdout).NotTo(gbytes.Say("65534"))
			Expect(stdout).To(gbytes.Say(" root "))
		})

		Context("when the process is run as non-root user", func() {
			BeforeEach(func() {
				rootfs = "docker:///ubuntu"
			})

			Context("and the user changes to root", func() {
				JustBeforeEach(func() {
					process, err := container.Run(garden.ProcessSpec{
						User: "root",
						Path: "sh",
						Args: []string{"-c", `echo "ALL            ALL = (ALL) NOPASSWD: ALL" >> /etc/sudoers`},
					}, garden.ProcessIO{
						Stdout: GinkgoWriter,
						Stderr: GinkgoWriter,
					})

					Expect(err).ToNot(HaveOccurred())
					Expect(process.Wait()).To(Equal(0))
				})

				It("can chown files", func() {
					process, err := container.Run(garden.ProcessSpec{
						User: "vcap",
						Path: "sudo",
						Args: []string{"chown", "-R", "vcap", "/tmp"},
					}, garden.ProcessIO{
						Stdout: GinkgoWriter,
						Stderr: GinkgoWriter,
					})

					Expect(err).ToNot(HaveOccurred())
					Expect(process.Wait()).To(Equal(0))
				})

				It("does not have certain capabilities", func() {
					// This attempts to set system time which requires the CAP_SYS_TIME permission.
					process, err := container.Run(garden.ProcessSpec{
						User: "vcap",
						Path: "sudo",
						Args: []string{"date", "--set", "+2 minutes"},
					}, garden.ProcessIO{
						Stdout: GinkgoWriter,
						Stderr: GinkgoWriter,
					})

					Expect(err).ToNot(HaveOccurred())
					Expect(process.Wait()).ToNot(Equal(0))
				})
			})
		})
	})
})
