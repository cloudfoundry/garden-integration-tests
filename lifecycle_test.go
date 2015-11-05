package garden_integration_tests_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	archiver "github.com/pivotal-golang/archiver/extractor/test_helper"
)

var _ = Describe("Lifecycle", func() {
	Context("Creating a container with limits", func() {
		BeforeEach(func() {
			limits = garden.Limits{
				Memory: garden.MemoryLimits{
					LimitInBytes: 1024 * 1024 * 128,
				},
				CPU: garden.CPULimits{
					LimitInShares: 50,
				},
			}
		})

		It("it applies limits if set in the container spec", func() {
			memoryLimit, err := container.CurrentMemoryLimits()
			Expect(err).ToNot(HaveOccurred())
			Expect(memoryLimit).To(Equal(limits.Memory))

			cpuLimit, err := container.CurrentCPULimits()
			Expect(err).ToNot(HaveOccurred())
			Expect(cpuLimit).To(Equal(limits.CPU))
		})

		It("does not apply limits if not set in container spec", func() {
			diskLimit, err := container.CurrentDiskLimits()
			Expect(err).ToNot(HaveOccurred())
			Expect(diskLimit).To(Equal(garden.DiskLimits{}))

			bandwidthLimit, err := container.CurrentBandwidthLimits()
			Expect(err).ToNot(HaveOccurred())
			Expect(bandwidthLimit).To(Equal(garden.BandwidthLimits{}))
		})
	})

	It("provides /dev/shm as tmpfs in the container", func() {
		process, err := container.Run(garden.ProcessSpec{
			User: "alice",
			Path: "dd",
			Args: []string{"if=/dev/urandom", "of=/dev/shm/some-data", "count=64", "bs=1k"},
		}, garden.ProcessIO{})
		Expect(err).ToNot(HaveOccurred())

		Expect(process.Wait()).To(Equal(0))

		outBuf := gbytes.NewBuffer()

		process, err = container.Run(garden.ProcessSpec{
			User: "alice",
			Path: "cat",
			Args: []string{"/proc/mounts"},
		}, garden.ProcessIO{
			Stdout: outBuf,
			Stderr: GinkgoWriter,
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(process.Wait()).To(Equal(0))

		Expect(outBuf).To(gbytes.Say("tmpfs /dev/shm tmpfs"))
		Expect(outBuf).To(gbytes.Say("rw,nodev,relatime"))
	})

	It("gives the container a hostname based on its id", func() {
		stdout := gbytes.NewBuffer()

		_, err := container.Run(garden.ProcessSpec{
			User: "alice",
			Path: "hostname",
		}, garden.ProcessIO{
			Stdout: stdout,
		})
		Expect(err).ToNot(HaveOccurred())

		Eventually(stdout).Should(gbytes.Say(fmt.Sprintf("%s\n", container.Handle())))
	})

	Context("and sending a List request", func() {
		It("includes the created container", func() {
			Expect(getContainerHandles()).To(ContainElement(container.Handle()))
		})
	})

	Context("and sending an Info request", func() {
		It("returns the container's info", func() {
			info, err := container.Info()
			Expect(err).ToNot(HaveOccurred())

			Expect(info.State).To(Equal("active"))
		})
	})

	Context("Using a docker image", func() {
		Context("when there is a VOLUME associated with the docker image", func() {
			BeforeEach(func() {
				// dockerfile contains `VOLUME /foo`, see diego-dockerfiles/with-volume
				rootfs = "docker:///cloudfoundry/with-volume"
			})

			JustBeforeEach(func() {
				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "adduser",
					Args: []string{"-D", "bob"},
				}, garden.ProcessIO{
					Stdout: GinkgoWriter,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(process.Wait()).To(Equal(0))
			})

			It("creates the volume directory, if it does not already exist", func() {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "bob",
					Path: "ls",
					Args: []string{"-l", "/"},
				}, garden.ProcessIO{
					Stdout: io.MultiWriter(GinkgoWriter, stdout),
					Stderr: GinkgoWriter,
				})

				Expect(err).ToNot(HaveOccurred())

				process.Wait()
				Expect(stdout).To(gbytes.Say("foo"))
			})
		})
	})

	Context("running a process", func() {
		Context("when root is requested", func() {
			It("runs as root inside the container", func() {
				stdout := gbytes.NewBuffer()

				_, err := container.Run(garden.ProcessSpec{
					Path: "whoami",
					User: "root",
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: GinkgoWriter,
				})

				Expect(err).ToNot(HaveOccurred())
				Eventually(stdout).Should(gbytes.Say("root\n"))
			})

			Context("and there is no /root directory in the image", func() {
				BeforeEach(func() {
					rootfs = "docker:///onsi/grace-busybox"
				})

				It("still allows running as root", func() {
					_, err := container.Run(garden.ProcessSpec{
						Path: "ls",
						User: "root",
					}, garden.ProcessIO{})

					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		It("streams output back and reports the exit status", func() {
			stdout := gbytes.NewBuffer()
			stderr := gbytes.NewBuffer()

			process, err := container.Run(garden.ProcessSpec{
				User: "alice",
				Path: "sh",
				Args: []string{"-c", "sleep 0.5; echo $FIRST; sleep 0.5; echo $SECOND >&2; sleep 0.5; exit 42"},
				Env:  []string{"FIRST=hello", "SECOND=goodbye"},
			}, garden.ProcessIO{
				Stdout: stdout,
				Stderr: stderr,
			})
			Expect(err).ToNot(HaveOccurred())

			Eventually(stdout).Should(gbytes.Say("hello\n"))
			Eventually(stderr).Should(gbytes.Say("goodbye\n"))
			Expect(process.Wait()).To(Equal(42))
		})

		It("sends a TERM signal to the process if requested", func() {

			stdout := gbytes.NewBuffer()

			process, err := container.Run(garden.ProcessSpec{
				User: "alice",
				Path: "sh",
				Args: []string{"-c", `
				trap 'echo termed; exit 42' SIGTERM

				while true; do
					echo waiting
					sleep 1
				done
			`},
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

		It("sends a TERM signal to the process run by root if requested", func() {

			stdout := gbytes.NewBuffer()

			process, err := container.Run(garden.ProcessSpec{
				User: "root",
				Path: "sh",
				Args: []string{"-c", `
				trap 'echo termed; exit 42' SIGTERM

				while true; do
					echo waiting
					sleep 1
				done
			`},
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

		Context("even when /bin/kill does not exist", func() {
			JustBeforeEach(func() {
				process, err := container.Run(garden.ProcessSpec{
					User: "root",
					Path: "rm",
					Args: []string{"/bin/kill"},
				}, garden.ProcessIO{
					Stdout: GinkgoWriter,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(process.Wait()).To(Equal(0))
			})

			checkProcessIsGone := func(container garden.Container, argsPrefix string) {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf(`
						 ps ax -o args= | grep -q '^%s'
					 `, argsPrefix)},
				}, garden.ProcessIO{
					Stdout: io.MultiWriter(stdout, GinkgoWriter),
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(process.Wait()).To(Equal(1))
				Eventually(stdout).ShouldNot(gbytes.Say("waiting"))
			}

			It("sends a KILL signal to the process if requested", func(done Done) {
				stdout := gbytes.NewBuffer()
				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "sh",
					Args: []string{"-c", `
							while true; do
							  echo waiting
								sleep 1
							done
						`},
				}, garden.ProcessIO{
					Stdout: io.MultiWriter(GinkgoWriter, stdout),
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())
				Eventually(stdout).Should(gbytes.Say("waiting"))

				Expect(process.Signal(garden.SignalKill)).To(Succeed())
				Expect(process.Wait()).To(Equal(255))

				checkProcessIsGone(container, "sh -c while")

				close(done)
			}, 10.0)

			It("sends a TERMINATE signal to the process if requested", func(done Done) {
				stdout := gbytes.NewBuffer()

				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "sh",
					Args: []string{"-c", `
							while true; do
							  echo waiting
								sleep 1
							done
						`},
				}, garden.ProcessIO{
					Stdout: io.MultiWriter(GinkgoWriter, stdout),
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())
				Eventually(stdout).Should(gbytes.Say("waiting"))

				Expect(process.Signal(garden.SignalTerminate)).To(Succeed())
				Expect(process.Wait()).To(Equal(255))

				checkProcessIsGone(container, "sh -c while")

				close(done)
			}, 10.0)

			Context("when killing a process that does not use streaming", func() {
				var process garden.Process

				JustBeforeEach(func() {
					var err error

					process, err = container.Run(garden.ProcessSpec{
						User: "alice",
						Path: "sleep",
						Args: []string{"1000"},
					}, garden.ProcessIO{})
					Expect(err).ToNot(HaveOccurred())

					Expect(process.Signal(garden.SignalKill)).To(Succeed())
				})

				It("goes away", func(done Done) {
					Expect(process.Wait()).NotTo(Equal(0))

					checkProcessIsGone(container, "sleep")

					close(done)
				}, 30.0)
			})
		})

		It("avoids a race condition when sending a kill signal", func(done Done) {
			for i := 0; i < 100; i++ {
				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "sh",
					Args: []string{"-c", `while true; do echo -n "x"; sleep 1; done`},
				}, garden.ProcessIO{
					Stdout: GinkgoWriter,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(process.Signal(garden.SignalKill)).To(Succeed())
				Expect(process.Wait()).NotTo(Equal(0))
			}

			close(done)
		}, 480.0)

		It("collects the process's full output, even if it exits quickly after", func() {
			for i := 0; i < 1000; i++ {
				stdout := gbytes.NewBuffer()

				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "sh",
					Args: []string{"-c", "cat <&0"},
				}, garden.ProcessIO{
					Stdin:  bytes.NewBuffer([]byte("hi stdout")),
					Stderr: os.Stderr,
					Stdout: stdout,
				})

				if err != nil {
					println("ERROR: " + err.Error())
					select {}
				}

				Expect(err).ToNot(HaveOccurred())
				Expect(process.Wait()).To(Equal(0))

				Expect(stdout).To(gbytes.Say("hi stdout"))
			}
		})

		It("streams input to the process's stdin", func() {
			stdout := gbytes.NewBuffer()

			process, err := container.Run(garden.ProcessSpec{
				User: "alice",
				Path: "sh",
				Args: []string{"-c", "cat <&0"},
			}, garden.ProcessIO{
				Stdin:  bytes.NewBufferString("hello\nworld"),
				Stdout: stdout,
			})
			Expect(err).ToNot(HaveOccurred())

			Eventually(stdout).Should(gbytes.Say("hello\nworld"))
			Expect(process.Wait()).To(Equal(0))
		})

		It("forwards the exit status even if stdin is still being written", func() {
			// this covers the case of intermediaries shuffling i/o around (e.g. wsh)
			// receiving SIGPIPE on write() due to the backing process exiting without
			// flushing stdin
			//
			// in practice it's flaky; sometimes write() finishes just before the
			// process exits, so run it ~10 times (observed it fail often in this range)

			for i := 0; i < 10; i++ {
				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "ls",
				}, garden.ProcessIO{
					Stdin: bytes.NewBufferString(strings.Repeat("x", 1024)),
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(process.Wait()).To(Equal(0))
			}
		})

		Context("when no user is specified", func() {

			It("returns an error", func() {
				_, err := container.Run(garden.ProcessSpec{
					Path: "pwd",
				}, garden.ProcessIO{})
				Expect(err).To(MatchError(ContainSubstring("A User for the process to run as must be specified")))
			})
		})

		Context("with a memory limit", func() {
			JustBeforeEach(func() {
				err := container.LimitMemory(garden.MemoryLimits{
					LimitInBytes: 64 * 1024 * 1024,
				})
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when the process writes too much to /dev/shm", func() {
				It("is killed", func() {
					process, err := container.Run(garden.ProcessSpec{
						User: "alice",
						Path: "dd",
						Args: []string{"if=/dev/urandom", "of=/dev/shm/too-big", "bs=1M", "count=65"},
					}, garden.ProcessIO{})
					Expect(err).ToNot(HaveOccurred())

					Expect(process.Wait()).ToNot(Equal(0))
				})
			})
		})

		Context("with a tty", func() {
			It("executes the process with a raw tty with the given window size", func() {
				stdout := gbytes.NewBuffer()

				inR, inW := io.Pipe()

				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "sh",
					Args: []string{"-c", "read foo; stty -a"},
					TTY: &garden.TTYSpec{
						WindowSize: &garden.WindowSize{
							Columns: 123,
							Rows:    456,
						},
					},
				}, garden.ProcessIO{
					Stdin:  inR,
					Stdout: stdout,
				})
				Expect(err).ToNot(HaveOccurred())

				_, err = inW.Write([]byte("hello"))
				Expect(err).ToNot(HaveOccurred())

				Eventually(stdout).Should(gbytes.Say("hello"))

				_, err = inW.Write([]byte("\n"))
				Expect(err).ToNot(HaveOccurred())

				Eventually(stdout, "3s").Should(gbytes.Say("rows 456; columns 123;"))

				Expect(process.Wait()).To(Equal(0))
			})

			It("can have its terminal resized", func() {
				stdout := gbytes.NewBuffer()

				inR, inW := io.Pipe()

				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "sh",
					Args: []string{
						"-c",
						`
						trap "stty -a" SIGWINCH

						# continuously read so that the trap can keep firing
						while true; do
							echo waiting
							if read; then
								exit 0
							fi
						done
					`,
					},
					TTY: &garden.TTYSpec{},
				}, garden.ProcessIO{
					Stdin:  inR,
					Stdout: stdout,
				})
				Expect(err).ToNot(HaveOccurred())

				Eventually(stdout).Should(gbytes.Say("waiting"))

				err = process.SetTTY(garden.TTYSpec{
					WindowSize: &garden.WindowSize{
						Columns: 123,
						Rows:    456,
					},
				})
				Expect(err).ToNot(HaveOccurred())

				Eventually(stdout).Should(gbytes.Say("rows 456; columns 123;"))

				_, err = fmt.Fprintf(inW, "ok\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(process.Wait()).To(Equal(0))
			})
		})

		Context("with a working directory", func() {
			It("executes with the working directory as the dir", func() {
				stdout := gbytes.NewBuffer()

				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "pwd",
					Dir:  "/usr",
				}, garden.ProcessIO{
					Stdout: stdout,
				})
				Expect(err).ToNot(HaveOccurred())

				Eventually(stdout).Should(gbytes.Say("/usr\n"))
				Expect(process.Wait()).To(Equal(0))
			})
		})

		Context("and then attaching to it", func() {
			It("streams output and the exit status to the attached request", func(done Done) {
				stdout1 := gbytes.NewBuffer()
				stdout2 := gbytes.NewBuffer()

				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "sh",
					Args: []string{"-c", "sleep 2; echo hello; sleep 0.5; echo goodbye; sleep 0.5; exit 42"},
				}, garden.ProcessIO{
					Stdout: stdout1,
				})
				Expect(err).ToNot(HaveOccurred())

				attached, err := container.Attach(process.ID(), garden.ProcessIO{
					Stdout: stdout2,
				})
				Expect(err).ToNot(HaveOccurred())

				time.Sleep(2 * time.Second)

				Eventually(stdout1).Should(gbytes.Say("hello\n"))
				Eventually(stdout1).Should(gbytes.Say("goodbye\n"))

				Eventually(stdout2).Should(gbytes.Say("hello\n"))
				Eventually(stdout2).Should(gbytes.Say("goodbye\n"))

				Expect(process.Wait()).To(Equal(42))
				Expect(attached.Wait()).To(Equal(42))

				close(done)
			}, 10.0)
		})

		Context("and then sending a stop request", func() {
			It("terminates all running processes", func() {
				stdout := gbytes.NewBuffer()

				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "sh",
					Args: []string{
						"-c",
						`
					trap 'exit 42' SIGTERM

					# sync with test, and allow trap to fire when not sleeping
					while true; do
						echo waiting
						sleep 1
					done
					`,
					},
				}, garden.ProcessIO{
					Stdout: stdout,
					Stderr: GinkgoWriter,
				})
				Expect(err).ToNot(HaveOccurred())

				Eventually(stdout, 30).Should(gbytes.Say("waiting"))

				err = container.Stop(false)
				Expect(err).ToNot(HaveOccurred())

				Expect(process.Wait()).To(Equal(42))
			})

			It("recursively terminates all child processes", func(done Done) {
				defer close(done)

				stderr := gbytes.NewBuffer()

				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "sh",
					Args: []string{
						"-c",
						`
					# don't die until child processes die
					trap wait SIGTERM

					# spawn child that exits when it receives TERM
					sh -c 'trap wait SIGTERM; sleep 100 & wait' &

					# sync with test. Use stderr to avoid buffering in the shell.
					echo waiting >&2

					# wait on children
					wait
					`,
					},
				}, garden.ProcessIO{
					Stderr: stderr,
				})

				Expect(err).ToNot(HaveOccurred())

				Eventually(stderr, 5).Should(gbytes.Say("waiting\n"))

				stoppedAt := time.Now()

				err = container.Stop(false)
				Expect(err).ToNot(HaveOccurred())

				Expect(process.Wait()).To(Equal(143)) // 143 = 128 + SIGTERM

				Expect(time.Since(stoppedAt)).To(BeNumerically("<=", 9*time.Second))
			}, 15)

			It("changes the container's state to 'stopped'", func() {
				err := container.Stop(false)
				Expect(err).ToNot(HaveOccurred())

				info, err := container.Info()
				Expect(err).ToNot(HaveOccurred())

				Expect(info.State).To(Equal("stopped"))
			})

			Context("when a process does not die 10 seconds after receiving SIGTERM", func() {
				It("is forcibly killed", func() {
					stdout := gbytes.NewBuffer()
					process, err := container.Run(garden.ProcessSpec{
						User: "alice",
						Path: "sh",
						Args: []string{
							"-c",
							`
							trap "echo cannot touch this" SIGTERM

							echo waiting
							while true
							do
								sleep 1000
							done
						`,
						},
					}, garden.ProcessIO{Stdout: stdout})

					Eventually(stdout).Should(gbytes.Say("waiting"))

					Expect(err).ToNot(HaveOccurred())

					stoppedAt := time.Now()

					err = container.Stop(false)
					Expect(err).ToNot(HaveOccurred())

					exitStatus, err := process.Wait()
					Expect(err).ToNot(HaveOccurred())
					if exitStatus != 137 && exitStatus != 255 {
						Fail(fmt.Sprintf("Unexpected exitStatus: %d", exitStatus))
					}

					Expect(time.Since(stoppedAt)).To(BeNumerically(">=", 10*time.Second))
				})
			})
		})

		Context("and streaming files in", func() {
			var tarStream io.Reader

			JustBeforeEach(func() {
				tmpdir, err := ioutil.TempDir("", "some-temp-dir-parent")
				Expect(err).ToNot(HaveOccurred())

				tgzPath := filepath.Join(tmpdir, "some.tgz")

				archiver.CreateTarGZArchive(
					tgzPath,
					[]archiver.ArchiveFile{
						{
							Name: "./some-temp-dir",
							Dir:  true,
						},
						{
							Name: "./some-temp-dir/some-temp-file",
							Body: "some-body",
						},
					},
				)

				tgz, err := os.Open(tgzPath)
				Expect(err).ToNot(HaveOccurred())

				tarStream, err = gzip.NewReader(tgz)
				Expect(err).ToNot(HaveOccurred())
			})

			It("creates the files in the container, as the specified user", func() {
				err := container.StreamIn(garden.StreamInSpec{
					User:      "alice",
					Path:      "/home/alice",
					TarStream: tarStream,
				})
				Expect(err).ToNot(HaveOccurred())

				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "test",
					Args: []string{"-f", "/home/alice/some-temp-dir/some-temp-file"},
				}, garden.ProcessIO{})
				Expect(err).ToNot(HaveOccurred())

				Expect(process.Wait()).To(Equal(0))

				output := gbytes.NewBuffer()
				process, err = container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "ls",
					Args: []string{"-al", "/home/alice/some-temp-dir/some-temp-file"},
				}, garden.ProcessIO{
					Stdout: output,
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(process.Wait()).To(Equal(0))

				// output should look like -rwxrwxrwx 1 alice alice 9 Jan  1  1970 /tmp/some-container-dir/some-temp-dir/some-temp-file
				Expect(output).To(gbytes.Say("alice"))
				Expect(output).To(gbytes.Say("alice"))
			})

			Context("when no user specified", func() {
				It("streams the files in as root", func() {
					err := container.StreamIn(garden.StreamInSpec{
						Path:      "/home/alice",
						TarStream: tarStream,
					})
					Expect(err).ToNot(HaveOccurred())

					out := gbytes.NewBuffer()
					process, err := container.Run(garden.ProcessSpec{
						User: "root",
						Path: "ls",
						Args: []string{"-la", "/home/alice/some-temp-dir/some-temp-file"},
					}, garden.ProcessIO{
						Stdout: out,
						Stderr: out,
					})
					Expect(err).ToNot(HaveOccurred())

					Expect(process.Wait()).To(Equal(0))
					Expect(string(out.Contents())).To(ContainSubstring("root"))
				})
			})

			Context("when a non-existent user specified", func() {
				It("returns error", func() {
					err := container.StreamIn(garden.StreamInSpec{
						User:      "batman",
						Path:      "/home/alice",
						TarStream: tarStream,
					})
					Expect(err).To(MatchError(ContainSubstring("error streaming in")))
				})
			})

			Context("when the specified user does not have permission to stream in", func() {
				JustBeforeEach(func() {
					process, err := container.Run(garden.ProcessSpec{
						User: "root",
						Path: "adduser",
						Args: []string{"-D", "bob"},
					}, garden.ProcessIO{
						Stdout: GinkgoWriter,
						Stderr: GinkgoWriter,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(process.Wait()).To(Equal(0))
				})

				It("returns error", func() {
					err := container.StreamIn(garden.StreamInSpec{
						User:      "bob",
						Path:      "/home/alice",
						TarStream: tarStream,
					})
					Expect(err).To(MatchError(ContainSubstring("Permission denied")))
				})
			})

			Context("in a privileged container", func() {
				BeforeEach(func() {
					privilegedContainer = true
				})

				It("streams in relative to the default run directory", func() {
					err := container.StreamIn(garden.StreamInSpec{
						User:      "alice",
						Path:      ".",
						TarStream: tarStream,
					})
					Expect(err).ToNot(HaveOccurred())

					process, err := container.Run(garden.ProcessSpec{
						User: "alice",
						Path: "test",
						Args: []string{"-f", "some-temp-dir/some-temp-file"},
					}, garden.ProcessIO{})
					Expect(err).ToNot(HaveOccurred())

					Expect(process.Wait()).To(Equal(0))
				})
			})

			It("streams in relative to the default run directory", func() {
				err := container.StreamIn(garden.StreamInSpec{
					User:      "alice",
					Path:      ".",
					TarStream: tarStream,
				})
				Expect(err).ToNot(HaveOccurred())

				process, err := container.Run(garden.ProcessSpec{
					User: "alice",
					Path: "test",
					Args: []string{"-f", "some-temp-dir/some-temp-file"},
				}, garden.ProcessIO{})
				Expect(err).ToNot(HaveOccurred())

				Expect(process.Wait()).To(Equal(0))
			})

			It("returns an error when the tar process dies", func() {
				err := container.StreamIn(garden.StreamInSpec{
					User: "alice",
					Path: "/tmp/some-container-dir",
					TarStream: &io.LimitedReader{
						R: tarStream,
						N: 10,
					},
				})
				Expect(err).To(HaveOccurred())
			})

			Context("and then copying them out", func() {
				itStreamsTheDirectory := func(user string) {
					It("streams the directory", func() {
						process, err := container.Run(garden.ProcessSpec{
							User: "alice",
							Path: "sh",
							Args: []string{"-c", `mkdir -p some-outer-dir/some-inner-dir && touch some-outer-dir/some-inner-dir/some-file`},
						}, garden.ProcessIO{})
						Expect(err).ToNot(HaveOccurred())

						Expect(process.Wait()).To(Equal(0))

						tarOutput, err := container.StreamOut(garden.StreamOutSpec{
							User: user,
							Path: "/home/alice/some-outer-dir/some-inner-dir",
						})
						Expect(err).ToNot(HaveOccurred())

						tarReader := tar.NewReader(tarOutput)

						header, err := tarReader.Next()
						Expect(err).ToNot(HaveOccurred())
						Expect(header.Name).To(Equal("some-inner-dir/"))

						header, err = tarReader.Next()
						Expect(err).ToNot(HaveOccurred())
						Expect(header.Name).To(Equal("some-inner-dir/some-file"))
					})

				}

				itStreamsTheDirectory("alice")

				Context("when no user specified", func() {
					// Any user's files can be streamed out as root
					itStreamsTheDirectory("")
				})

				Context("with a trailing slash", func() {
					It("streams the contents of the directory", func() {
						process, err := container.Run(garden.ProcessSpec{
							User: "alice",
							Path: "sh",
							Args: []string{"-c", `mkdir -p some-container-dir && touch some-container-dir/some-file`},
						}, garden.ProcessIO{})
						Expect(err).ToNot(HaveOccurred())

						Expect(process.Wait()).To(Equal(0))

						tarOutput, err := container.StreamOut(garden.StreamOutSpec{
							User: "alice",
							Path: "some-container-dir/",
						})
						Expect(err).ToNot(HaveOccurred())

						tarReader := tar.NewReader(tarOutput)

						header, err := tarReader.Next()
						Expect(err).ToNot(HaveOccurred())
						Expect(header.Name).To(Equal("./"))

						header, err = tarReader.Next()
						Expect(err).ToNot(HaveOccurred())
						Expect(header.Name).To(Equal("./some-file"))
					})
				})
			})
		})
	})

	Context("when the container GraceTime is modified", func() {
		It("should disappear after grace time and before timeout", func() {
			_, err := gardenClient.Lookup(container.Handle())
			Expect(err).NotTo(HaveOccurred())

			container.SetGraceTime(500 * time.Millisecond)

			Eventually(func() error {
				_, err := gardenClient.Lookup(container.Handle())
				return err
			}, "10s").Should(HaveOccurred())

			container = nil // avoid double-destroying in AfterEach
		})
	})
})
