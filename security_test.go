package garden_integration_tests_test

import (
	"io"
	"os"
	"strings"

	"github.com/cloudfoundry-incubator/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Security", func() {
	Describe("PID namespace", func() {
		FIt("isolates processes so that only processes from inside the container are visible", func() {
			_, err := container.Run(garden.ProcessSpec{
				User: "alice",
				Path: "sleep",
				Args: []string{"989898"},
			}, garden.ProcessIO{
				Stdout: GinkgoWriter,
				Stderr: GinkgoWriter,
			})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() []string {
				psout := gbytes.NewBuffer()
				ps, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "sh",
					Args: []string{"-c", "ps -a"},
				}, garden.ProcessIO{
					Stdout: psout,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(ps.Wait()).To(Equal(0))
				return strings.Split(string(psout.Contents()), "\n")
			}).Should(HaveLen(6)) // header, wshd, sleep, sh, ps, \n
		})

		It("does not leak fds in to spawned processes", func() {
			stdout := gbytes.NewBuffer()
			process, err := container.Run(garden.ProcessSpec{
				User: "root",
				Path: "ls",
				Args: []string{"/proc/self/fd"},
			}, garden.ProcessIO{
				Stdout: stdout,
				Stderr: GinkgoWriter,
			})
			Expect(err).ToNot(HaveOccurred())

			exitStatus, err := process.Wait()
			Expect(err).ToNot(HaveOccurred())
			Expect(exitStatus).To(Equal(0))

			Expect(stdout).To(gbytes.Say("0\n1\n2\n3\n")) // stdin, stdout, stderr, /proc/self/fd
		})

		It("has the correct initial process", func() {
			stdout := gbytes.NewBuffer()
			process, err := container.Run(garden.ProcessSpec{
				User: "root",
				Path: "/bin/ps",
				Args: []string{"-o", "pid,args"},
			}, garden.ProcessIO{
				Stdout: stdout,
				Stderr: GinkgoWriter,
			})
			Expect(err).ToNot(HaveOccurred())

			exitStatus, err := process.Wait()
			Expect(err).ToNot(HaveOccurred())
			Expect(exitStatus).To(Equal(0))

			Expect(stdout).To(gbytes.Say(`\s+1\s+{exe}\s+initd.*-title="wshd: %s"`, container.Handle()))
		})
	})

	Describe("Mount namespace", func() {
		It("unmounts /tmp/garden-host* in the container", func() {
			stdout := gbytes.NewBuffer()
			process, err := container.Run(garden.ProcessSpec{
				User: "root",
				Path: "/bin/cat",
				Args: []string{"/proc/mounts"},
			}, garden.ProcessIO{
				Stdout: stdout,
				Stderr: GinkgoWriter,
			})
			Expect(err).ToNot(HaveOccurred())

			exitStatus, err := process.Wait()
			Expect(err).ToNot(HaveOccurred())
			Expect(exitStatus).To(Equal(0))
			Expect(stdout).ToNot(gbytes.Say(` /tmp/garden-host`))
		})
	})

	Describe("File system", func() {
		It("/tmp is world-writable in the container", func() {
			stdout := gbytes.NewBuffer()
			process, err := container.Run(garden.ProcessSpec{
				User: "root",
				Path: "ls",
				Args: []string{"-al", "/tmp"},
			}, garden.ProcessIO{
				Stdout: stdout,
				Stderr: GinkgoWriter,
			})
			Expect(err).ToNot(HaveOccurred())

			exitStatus, err := process.Wait()
			Expect(err).ToNot(HaveOccurred())
			Expect(exitStatus).To(Equal(0))
			Expect(stdout).To(gbytes.Say(`drwxrwxrwt`))
		})

		Context("in an unprivileged container", func() {
			BeforeEach(func() {
				privilegedContainer = false
			})

			It("/proc IS mounted as Read-Only", func() {
				stdout := gbytes.NewBuffer()

				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "cat",
					Args: []string{"/proc/mounts"},
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: GinkgoWriter,
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(process.Wait()).To(Equal(0))
				Expect(stdout).To(gbytes.Say("proc /proc proc ro"))
			})

			It("/sys IS mounted as Read-Only", func() {
				stdout := gbytes.NewBuffer()

				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "cat",
					Args: []string{"/proc/mounts"},
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: GinkgoWriter,
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(process.Wait()).To(Equal(0))
				Expect(stdout).To(gbytes.Say("sysfs /sys sysfs ro"))
			})
		})

		Context("in a privileged container", func() {
			BeforeEach(func() {
				privilegedContainer = true
			})

			It("/proc IS mounted as Read-Write", func() {
				stdout := gbytes.NewBuffer()

				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "cat",
					Args: []string{"/proc/mounts"},
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: GinkgoWriter,
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(process.Wait()).To(Equal(0))
				Expect(stdout).To(gbytes.Say("proc /proc proc rw"))
			})

			It("/sys IS mounted as Read-Only", func() {
				stdout := gbytes.NewBuffer()

				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "cat",
					Args: []string{"/proc/mounts"},
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: GinkgoWriter,
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(process.Wait()).To(Equal(0))
				Expect(stdout).To(gbytes.Say("sysfs /sys sysfs ro"))
			})
		})
	})

	Describe("Control groups", func() {
		It("places the container in the required cgroup subsystems", func() {
			stdout := gbytes.NewBuffer()
			process, err := container.Run(garden.ProcessSpec{
				User: "root",
				Path: "/bin/sh",
				Args: []string{"-c", "cat /proc/$$/cgroup"},
			}, garden.ProcessIO{
				Stdout: stdout,
				Stderr: GinkgoWriter,
			})
			Expect(err).ToNot(HaveOccurred())

			exitStatus, err := process.Wait()
			Expect(err).ToNot(HaveOccurred())
			Expect(exitStatus).To(Equal(0))

			op := stdout.Contents()
			Expect(op).To(MatchRegexp(`\bcpu\b`))
			Expect(op).To(MatchRegexp(`\bcpuacct\b`))
			Expect(op).To(MatchRegexp(`\bcpuset\b`))
			Expect(op).To(MatchRegexp(`\bdevices\b`))
			Expect(op).To(MatchRegexp(`\bmemory\b`))
		})
	})

	Describe("Users and groups", func() {
		Context("when running a command in a working dir", func() {
			It("executes with setuid and setgid", func() {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Dir:  "/usr",
					Path: "pwd",
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())

				exitStatus, err := process.Wait()
				Expect(err).ToNot(HaveOccurred())
				Expect(exitStatus).To(Equal(0))
				Expect(stdout).To(gbytes.Say("^/usr\n"))
			})
		})

		Context("when running a command as a non-root user", func() {
			It("executes with setuid and setgid", func() {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "/bin/sh",
					Args: []string{"-c", "id -u; id -g"},
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())

				exitStatus, err := process.Wait()
				Expect(err).ToNot(HaveOccurred())
				Expect(exitStatus).To(Equal(0))
				Expect(stdout).To(gbytes.Say("1001\n1001\n"))
			})

			It("sets $HOME, $USER, and $PATH", func() {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "/bin/sh",
					Args: []string{"-c", "env | sort"},
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())

				exitStatus, err := process.Wait()
				Expect(err).ToNot(HaveOccurred())
				Expect(exitStatus).To(Equal(0))
				Expect(stdout).To(gbytes.Say("HOME=/home/alice\nPATH=/usr/local/bin:/usr/bin:/bin\nPWD=/home/alice\nSHLVL=1\nUSER=alice\n"))
			})

			Context("when $HOME is set in the spec", func() {
				It("sets $HOME from the spec", func() {
					stdout := gbytes.NewBuffer()
					process, err := container.Run(garden.ProcessSpec{
						User: "alice",
						Path: "/bin/sh",
						Args: []string{"-c", "echo $HOME"},
						Env: []string{
							"HOME=/nowhere",
						},
					}, garden.ProcessIO{
						Stdout: stdout,
						Stderr: GinkgoWriter,
					})
					Expect(err).ToNot(HaveOccurred())

					exitStatus, err := process.Wait()
					Expect(err).ToNot(HaveOccurred())
					Expect(exitStatus).To(Equal(0))
					Expect(stdout).To(gbytes.Say("/nowhere"))
				})
			})

			It("executes in the user's home directory", func() {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "/bin/pwd",
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())

				exitStatus, err := process.Wait()
				Expect(err).ToNot(HaveOccurred())
				Expect(exitStatus).To(Equal(0))
				Expect(stdout).To(gbytes.Say("/home/alice\n"))
			})

			It("sets the specified environment variables", func() {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Env:  []string{"VAR1=VALUE1", "VAR2=VALUE2"},
					Path: "/bin/sh",
					Args: []string{"-c", "env | sort"},
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())

				exitStatus, err := process.Wait()
				Expect(err).ToNot(HaveOccurred())
				Expect(exitStatus).To(Equal(0))
				Expect(stdout).To(gbytes.Say("VAR1=VALUE1\nVAR2=VALUE2\n"))
			})

			It("searches a sanitized path not including /sbin for the executable", func() {
				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "ls",
				}, garden.ProcessIO{
					Stdout: GinkgoWriter,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())
				exitStatus, err := process.Wait()
				Expect(err).ToNot(HaveOccurred())
				Expect(exitStatus).To(Equal(0))

				process, err = container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "ifconfig", // ifconfig is only available in /sbin
				}, garden.ProcessIO{
					Stdout: GinkgoWriter,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())
				exitStatus, err = process.Wait()
				Expect(err).ToNot(HaveOccurred())
				Expect(exitStatus).To(Equal(255))
			})

		})

		Context("when running a command as root", func() {
			It("executes with setuid and setgid", func() {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "/bin/sh",
					Args: []string{"-c", "id -u; id -g"},
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())

				exitStatus, err := process.Wait()
				Expect(err).ToNot(HaveOccurred())
				Expect(exitStatus).To(Equal(0))
				Expect(stdout).To(gbytes.Say("0\n0\n"))
			})

			It("sets $HOME, $USER, and $PATH", func() {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "/bin/sh",
					Args: []string{"-c", "env | sort"},
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())

				exitStatus, err := process.Wait()
				Expect(err).ToNot(HaveOccurred())
				Expect(exitStatus).To(Equal(0))
				Expect(stdout).To(gbytes.Say("HOME=/root\nPATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\nPWD=/root\nSHLVL=1\nUSER=root\n"))
			})

			It("executes in root's home directory", func() {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "/bin/pwd",
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())

				exitStatus, err := process.Wait()
				Expect(err).ToNot(HaveOccurred())
				Expect(exitStatus).To(Equal(0))
				Expect(stdout).To(gbytes.Say("/root\n"))
			})

			It("sets the specified environment variables", func() {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Env:  []string{"VAR1=VALUE1", "VAR2=VALUE2"},
					Path: "/bin/sh",
					Args: []string{"-c", "env | sort"},
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())

				exitStatus, err := process.Wait()
				Expect(err).ToNot(HaveOccurred())
				Expect(exitStatus).To(Equal(0))
				Expect(stdout).To(gbytes.Say("VAR1=VALUE1\nVAR2=VALUE2\n"))
			})

			It("searches a sanitized path not including /sbin for the executable", func() {
				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "ifconfig", // ifconfig is only available in /sbin
				}, garden.ProcessIO{
					Stdout: GinkgoWriter,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())
				exitStatus, err := process.Wait()
				Expect(err).ToNot(HaveOccurred())
				Expect(exitStatus).To(Equal(0))
			})
		})
	})

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
					Args: []string{"-c", `ls -l /sbin | grep -v wsh | grep -v hook | grep -v proc_starter | grep -v initd`},
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

			It("sees the root directory as owned by the container's root user", func() {
				if os.Getenv("BTRFS_SUPPORTED") == "" {
					Skip("Non-BTRFS drivers do not support making the container's root directory owned by container root")
				}
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

					process, err = container.Run(garden.ProcessSpec{
						User: "root",
						Path: "useradd",
						Args: []string{"-U", "-m", "bob"},
					}, garden.ProcessIO{
						Stdout: GinkgoWriter,
						Stderr: GinkgoWriter,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(process.Wait()).To(Equal(0))
				})

				It("can chown files", func() {
					process, err := container.Run(garden.ProcessSpec{
						User: "bob",
						Path: "sudo",
						Args: []string{"chown", "-R", "bob", "/tmp"},
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
						User: "bob",
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
