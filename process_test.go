package garden_integration_tests_test

import (
	"fmt"
	"runtime/debug"
	"time"

	"code.cloudfoundry.org/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Process", func() {
	Describe("signalling", func() {
		It("a process can be sent SIGTERM immediately after having been started", func() {
			stdout := gbytes.NewBuffer()

			process, err := container.Run(garden.ProcessSpec{
				User: "root",
				Path: "sh",
				Args: []string{
					"-c",
					`
                sleep 10
                exit 12
                `,
				},
			}, garden.ProcessIO{
				Stdout: stdout,
			})
			Expect(err).ToNot(HaveOccurred())

			err = process.Signal(garden.SignalTerminate)
			Expect(err).ToNot(HaveOccurred())
			Expect(process.Wait()).NotTo(Equal(12))
		})
	})

	Describe("when we try to create a container process with bind mounts", func() {
		It("should return an error", func() {
			stdout := gbytes.NewBuffer()

			_, err := container.Run(garden.ProcessSpec{
				User: "root",
				Path: "whoami",
				Args: []string{},
				BindMounts: []garden.BindMount{
					garden.BindMount{
						SrcPath: "src",
						DstPath: "dst",
					},
				},
			}, garden.ProcessIO{
				Stdout: stdout,
			})
			Expect(err).To(HaveOccurred())
		})
	})
	Describe("process ID", func() {
		It("return a process containing the ID passed in the process spec", func() {
			process, err := container.Run(garden.ProcessSpec{
				ID:   "some-id",
				Path: "/bin/true",
			}, garden.ProcessIO{})
			Expect(err).ToNot(HaveOccurred())
			Expect(process.ID()).To(Equal("some-id"))
		})

		Context("when two processes with the same ID are running", func() {
			var processID string

			JustBeforeEach(func() {
				processID = "same-id"
				_, err := container.Run(garden.ProcessSpec{
					ID:   processID,
					Path: "sleep",
					Args: []string{"5"},
				}, garden.ProcessIO{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("the second process with the same id should explode", func() {
				_, err := container.Run(garden.ProcessSpec{
					ID:   processID,
					Path: "/bin/true",
				}, garden.ProcessIO{})
				Expect(err).To(MatchError(MatchRegexp(`already (in use|exists)`)))
			})
		})
	})

	Describe("environment", func() {
		It("should apply the specified environment", func() {
			exitCode, stdout, _ := runProcess(container, garden.ProcessSpec{
				Path: "env",
				Env: []string{
					"TEST=hello",
					"FRUIT=banana",
				},
			})
			Expect(exitCode).To(Equal(0))
			Expect(stdout).To(gbytes.Say("TEST=hello\nFRUIT=banana"))
		})

		Context("when the container has container spec environment specified", func() {
			BeforeEach(func() {
				env = []string{
					"CONTAINER_ENV=1",
					"TEST=hi",
				}
			})

			It("should apply the merged environment variables", func() {
				exitCode, stdout, _ := runProcess(container, garden.ProcessSpec{
					Path: "env",
					Env: []string{
						"TEST=hello",
						"FRUIT=banana",
					},
				})
				Expect(exitCode).To(Equal(0))
				Expect(stdout).To(gbytes.Say("CONTAINER_ENV=1\nTEST=hello\nFRUIT=banana"))
			})
		})
	})

	Describe("wait", func() {
		It("does not block in Wait() when all children of the process have exited", func() {
			stderr := gbytes.NewBuffer()
			process, err := container.Run(garden.ProcessSpec{
				User: "root",
				Path: "/bin/sh",
				Args: []string{"-c", `

				  cleanup ()
				  {
						ps -a >&2
						kill $child_pid
						exit 42
				  }

				  trap cleanup TERM
				  set -x
				  sleep 1000 &
				  child_pid=$!
				  # Make sure that sleep process has been forked before trapping
				  while [ ! $(ps -o comm | grep sleep) ] ;do : ; done
				  # Use stderr to avoid buffering in the shell.
				  echo trapping >&2
				  wait
				`},
			}, garden.ProcessIO{Stderr: stderr})
			Expect(err).NotTo(HaveOccurred())

			exitChan := make(chan int)
			go func(p garden.Process, exited chan<- int) {
				defer GinkgoRecover()
				status, waitErr := p.Wait()
				Expect(waitErr).NotTo(HaveOccurred())
				exited <- status
			}(process, exitChan)

			Eventually(stderr).Should(gbytes.Say("trapping"))

			Expect(process.Signal(garden.SignalTerminate)).To(Succeed())

			select {
			case status := <-exitChan:
				Expect(status).To(Equal(42))
			case <-time.After(time.Second * 20):
				debug.PrintStack()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("Process Stderr: %s", string(stderr.Contents()))
				Fail("timed out!")
			}
		})
	})

	Describe("user", func() {
		Context("when the user is specified in the form uid:gid", func() {
			It("runs the process as that user", func() {
				stdout := runForStdout(container, garden.ProcessSpec{
					User: "1001:1002",
					Path: "sh",
					Args: []string{"-c", "echo $(id -u):$(id -g)"},
				})
				Expect(stdout).To(gbytes.Say("1001:1002\n"))
			})
		})

		Context("when the user is specified ins the form username:groupname", func() {
			BeforeEach(func() {
				imageRef = garden.ImageRef{
					URI: "docker:///cfgarden/run-as-user-group:0.0.1",
				}
			})

			It("runs the process as that user", func() {
				stdout := runForStdout(container, garden.ProcessSpec{
					User: "vcap:staff",
					Path: "sh",
					Args: []string{"-c", "echo $(id -u):$(id -g)"},
				})
				Expect(stdout).To(gbytes.Say("1000:50\n"))
			})
		})

		Context("when the user is not specified", func() {
			It("runs the process as root", func() {
				stdout := runForStdout(container, garden.ProcessSpec{
					Path: "whoami",
				})
				Expect(stdout).To(gbytes.Say("root\n"))
			})
		})
	})

	Describe("working directory", func() {
		JustBeforeEach(func() {
			createUser(container, "alice")
		})

		Context("when user has access to working directory", func() {
			Context("when working directory exists", func() {
				It("spawns the process", func() {
					stdout := runForStdout(container, garden.ProcessSpec{
						User: "alice",
						Dir:  "/home/alice",
						Path: "pwd",
					})

					Expect(stdout).To(gbytes.Say("/home/alice"))
				})
			})

			Context("when working directory does not exist", func() {
				BeforeEach(func() {
					skipIfRootless()
				})

				It("spawns the process", func() {
					stdout := runForStdout(container, garden.ProcessSpec{
						User: "alice",
						Dir:  "/home/alice/nonexistent",
						Path: "pwd",
					})

					Expect(stdout).To(gbytes.Say("/home/alice/nonexistent"))
				})

				It("is created owned by the requested user", func() {
					stdout := runForStdout(container, garden.ProcessSpec{
						User: "root",
						Dir:  "/root/nonexistent",
						Path: "sh",
						Args: []string{"-c", "ls -la . | head -n 2 | tail -n 1"},
					})

					Expect(stdout).To(gbytes.Say("root"))
				})
			})
		})

		Context("when user does not have access to working directory", func() {
			JustBeforeEach(func() {
				exitCode, _, _ := runProcess(container, garden.ProcessSpec{
					User: "alice",
					Path: "sh",
					Args: []string{"-c", "mkdir -p /home/alice/nopermissions && chmod 0555 /home/alice/nopermissions"},
				})
				Expect(exitCode).To(Equal(0))
			})

			Context("when working directory does exist", func() {
				It("returns an error", func() {
					exitCode, _, stderr := runProcess(container, garden.ProcessSpec{
						User: "alice",
						Dir:  "/home/alice/nopermissions",
						Path: "touch",
						Args: []string{"test.txt"},
					})

					Expect(exitCode).ToNot(Equal(0))
					Expect(stderr).To(gbytes.Say("Permission denied"))
				})
			})

			Context("when working directory does not exist", func() {
				BeforeEach(func() {
					skipIfRootless()
				})

				It("should create the working directory, and succeed", func() {
					exitCode, _, _ := runProcess(container, garden.ProcessSpec{
						User: "alice",
						Dir:  "/home/alice/nopermissions/nonexistent",
						Path: "touch",
						Args: []string{"test.txt"},
					})

					Expect(exitCode).To(Equal(0))
				})
			})
		})

		Context("when the user does not specify the working directory", func() {
			It("should have the user home directory in the output", func() {
				stdout := runForStdout(container, garden.ProcessSpec{
					User: "alice",
					Path: "pwd",
				})

				Expect(stdout).To(gbytes.Say("/home/alice"))
			})
		})
	})
})
