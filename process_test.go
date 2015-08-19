package garden_integration_tests_test

import (
	"io"
	"runtime/debug"
	"time"

	"github.com/cloudfoundry-incubator/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Process", func() {
	BeforeEach(func() {
		rootfs = "docker:///ubuntu"
	})

	Describe("signalling", func() {
		JustBeforeEach(func() {
			process, err := container.Run(garden.ProcessSpec{
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

		It("a process can be sent SIGTERM immediately after having been started", func() {
			stdout := gbytes.NewBuffer()

			process, err := container.Run(garden.ProcessSpec{
				User: "bob",
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

	Describe("wait", func() {
		JustBeforeEach(func() {
			process, err := container.Run(garden.ProcessSpec{
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

		It("does not block in Wait() when all children of the process have exited", func() {
			buffer := gbytes.NewBuffer()
			process, err := container.Run(garden.ProcessSpec{
				User: "bob",
				Path: "/bin/bash",
				Args: []string{"-c", `

				  cleanup ()
				  {
				  	kill $child_pid
				  	exit 42
				  }

				  trap cleanup TERM
				  echo trapping

				  sleep 1000 &
				  child_pid=$!
				  wait
				`},
			}, garden.ProcessIO{Stdout: buffer})
			Expect(err).NotTo(HaveOccurred())

			exitChan := make(chan int)
			go func(p garden.Process, exited chan<- int) {
				GinkgoRecover()
				status, waitErr := p.Wait()
				Expect(waitErr).NotTo(HaveOccurred())
				exited <- status
			}(process, exitChan)

			Eventually(buffer).Should(gbytes.Say("trapping"))

			Expect(process.Signal(garden.SignalTerminate)).To(Succeed())

			select {
			case status := <-exitChan:
				Expect(status).To(Equal(42))
			case <-time.After(time.Second * 10):
				debug.PrintStack()
				Fail("timed out!")
			}
		})

		It("does not block in Wait() when a child of the process has not exited", func() {
			buffer := gbytes.NewBuffer()
			process, err := container.Run(garden.ProcessSpec{
				User: "bob",
				Path: "/bin/bash",
				Args: []string{"-c", `
					cleanup ()
					{
						exit 42
					}

					trap cleanup TERM
					sleep 100000 &
					disown -h
 -
					echo trapping
					wait
				`},
			}, garden.ProcessIO{Stdout: buffer})
			Expect(err).NotTo(HaveOccurred())

			exitChan := make(chan int)
			go func(p garden.Process, exited chan<- int) {
				GinkgoRecover()
				status, waitErr := p.Wait()
				Expect(waitErr).NotTo(HaveOccurred())
				exited <- status
			}(process, exitChan)

			Eventually(buffer).Should(gbytes.Say("trapping"))

			Expect(process.Signal(garden.SignalTerminate)).To(Succeed())
			select {
			case status := <-exitChan:
				Expect(status).To(Equal(42))
			case <-time.After(time.Second * 10):
				Fail("Wait should not block when a child has not exited")
			}
		})
	})

	Describe("working directory", func() {
		BeforeEach(func() {
			rootfs = "docker:///cloudfoundry/preexisting_users"
		})

		Context("when user has access to working directory", func() {
			Context("when working directory exists", func() {
				It("a process is spawned", func() {
					out := gbytes.NewBuffer()
					process, err := container.Run(garden.ProcessSpec{
						User: "alice",
						Dir:  "/home/alice",
						Path: "pwd",
					}, garden.ProcessIO{
						Stdout: out,
						Stderr: out,
					})

					Expect(err).ToNot(HaveOccurred())
					Expect(process.Wait()).To(Equal(0))
					Eventually(out).Should(gbytes.Say("/home/alice"))
				})
			})
		})

		Context("when user has access to create working directory", func() {
			Context("when working directory does not exist", func() {
				It("a process is spawned", func() {
					out := gbytes.NewBuffer()
					process, err := container.Run(garden.ProcessSpec{
						User: "alice",
						Dir:  "/home/alice/nonexistent",
						Path: "pwd",
					}, garden.ProcessIO{
						Stdout: out,
						Stderr: GinkgoWriter,
					})

					Expect(err).ToNot(HaveOccurred())
					Expect(process.Wait()).To(Equal(0))
					Eventually(out).Should(gbytes.Say("/home/alice/nonexistent"))
				})
			})
		})

		Context("when user does not have access to working directory", func() {
			Context("when working directory does exist", func() {
				It("returns an error", func() {
					out := gbytes.NewBuffer()
					process, err := container.Run(garden.ProcessSpec{
						User: "alice",
						Dir:  "/root",
						Path: "ls",
					}, garden.ProcessIO{
						Stdout: GinkgoWriter,
						Stderr: io.MultiWriter(GinkgoWriter, out),
					})

					Expect(err).ToNot(HaveOccurred())

					exitStatus, err := process.Wait()
					Expect(exitStatus).ToNot(Equal(0))
					Expect(out).To(gbytes.Say("proc_starter: ExecAsUser: system: invalid working directory: /root"))
				})
			})

			Context("when working directory does not exist", func() {
				It("returns an error", func() {
					out := gbytes.NewBuffer()
					process, err := container.Run(garden.ProcessSpec{
						User: "alice",
						Dir:  "/home/bob/nonexistent",
						Path: "pwd",
					}, garden.ProcessIO{
						Stdout: GinkgoWriter,
						Stderr: io.MultiWriter(GinkgoWriter, out),
					})

					Expect(err).ToNot(HaveOccurred())
					exitStatus, err := process.Wait()
					Expect(exitStatus).ToNot(Equal(0))
					Expect(out).To(gbytes.Say("proc_starter: ExecAsUser: system: mkdir /home/bob/nonexistent: permission denied"))
				})
			})
		})
	})
})
