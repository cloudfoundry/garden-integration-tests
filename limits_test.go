package garden_integration_tests_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/cloudfoundry-incubator/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Limits", func() {
	Describe("LimitMemory", func() {
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
						User: "vcap",
						Path: "dd",
						Args: []string{"if=/dev/urandom", "of=/dev/shm/too-big", "bs=1M", "count=65"},
					}, garden.ProcessIO{})
					Expect(err).ToNot(HaveOccurred())

					Expect(process.Wait()).ToNot(Equal(0))
				})
			})
		})
	})

	Describe("LimitDisk", func() {
		Context("when quotas are enabled and there is a disk limit", func() {
			var byteSoftQuota uint64
			var byteHardQuota uint64
			var quotaScope garden.DiskLimitScope

			const BTRFS_WAIT_TIME = 120

			BeforeEach(func() {
				if os.Getenv("BTRFS_SUPPORTED") == "" {
					Skip("btrfs not available")
				}

				byteSoftQuota = 180 * 1024 * 1024
				byteHardQuota = 180 * 1024 * 1024
				quotaScope = garden.DiskLimitScopeTotal
			})

			JustBeforeEach(func() {
				err := container.LimitDisk(garden.DiskLimits{
					ByteSoft: byteSoftQuota,
					ByteHard: byteHardQuota,
					Scope:    quotaScope,
				})
				Expect(err).ToNot(HaveOccurred())
			})

			Context("on a directory rootfs container", func() {
				It("reports correct disk usage", func() {
					var diskUsage uint64
					stdout := gbytes.NewBuffer()

					process, err := container.Run(garden.ProcessSpec{
						User: "vcap",
						Path: "sh",
						Args: []string{"-c", "du -d 0 / | awk ' {print $1 }'"},
					}, garden.ProcessIO{Stdout: stdout})
					Expect(err).ToNot(HaveOccurred())
					Expect(process.Wait()).To(Equal(0))

					_, err = fmt.Sscanf(strings.TrimSpace(string(stdout.Contents())), "%d", &diskUsage)
					Expect(err).ToNot(HaveOccurred())

					process, err = container.Run(garden.ProcessSpec{
						User: "vcap",
						Path: "dd",
						Args: []string{"if=/dev/urandom", "of=/home/vcap/some-file", "bs=1M", "count=10"},
					}, garden.ProcessIO{})
					Expect(err).ToNot(HaveOccurred())
					Expect(process.Wait()).To(Equal(0))

					metrics := func() garden.ContainerDiskStat {
						metricsAfter, err := container.Metrics()
						Expect(err).ToNot(HaveOccurred())

						return metricsAfter.DiskStat
					}

					metricsTotalBytes := func() uint64 {
						return metrics().TotalBytesUsed
					}

					metricsExclusiveBytes := func() uint64 {
						return metrics().ExclusiveBytesUsed
					}

					expectedTotalBytes := (diskUsage * 1024) + uint64(6*1024*1024)
					Eventually(metricsTotalBytes, BTRFS_WAIT_TIME, 30).Should(BeNumerically("~", expectedTotalBytes, uint64(4*1024*1024)))
					Eventually(metricsExclusiveBytes, BTRFS_WAIT_TIME, 30).Should(BeNumerically("~", uint64(9*1024*1024), uint64(11*1024*1024)))

					process, err = container.Run(garden.ProcessSpec{
						User: "vcap",
						Path: "dd",
						Args: []string{"if=/dev/urandom", "of=/home/vcap/another-file", "bs=1M", "count=10"},
					}, garden.ProcessIO{})
					Expect(err).ToNot(HaveOccurred())
					Expect(process.Wait()).To(Equal(0))

					expectedTotalBytes = (diskUsage * 1024) + uint64(6*1024*1024)
					Eventually(metricsTotalBytes, BTRFS_WAIT_TIME, 30).Should(BeNumerically("~", expectedTotalBytes, uint64(4*1024*1024)))
					Eventually(metricsExclusiveBytes, BTRFS_WAIT_TIME, 30).Should(BeNumerically("~", uint64(19*1024*1024), uint64(21*1024*1024)))
				})
			})

			Context("on a Docker container", func() {
				BeforeEach(func() {
					privilegedContainer = false
					rootfs = "docker:///busybox"
					byteSoftQuota = 10 * 1024 * 1024
					byteHardQuota = 10 * 1024 * 1024
					quotaScope = garden.DiskLimitScopeTotal
				})

				It("reports the correct disk limit size of the container", func() {
					limit, err := container.CurrentDiskLimits()
					Expect(err).ToNot(HaveOccurred())
					Expect(limit).To(Equal(garden.DiskLimits{
						ByteHard: byteHardQuota,
						ByteSoft: byteSoftQuota,
						Scope:    quotaScope,
					}))
				})

				Context("when the scope is total (the default)", func() {
					Context("and run a process that does not exceed the limit", func() {
						It("does not kill the process", func() {
							dd, err := container.Run(garden.ProcessSpec{
								User: "root",
								Path: "dd",
								Args: []string{"if=/dev/zero", "of=/root/test", "bs=1M", "count=3"}, // should succeed, even though equivalent with 'total' scope does not
							}, garden.ProcessIO{Stdout: GinkgoWriter, Stderr: GinkgoWriter})
							Expect(err).ToNot(HaveOccurred())
							Expect(dd.Wait()).To(Equal(0))
						})
					})

					Context("and run a process that exceeds the quota due to the size of the rootfs", func() {
						It("kills the process", func() {
							dd, err := container.Run(garden.ProcessSpec{
								User: "root",
								Path: "dd",
								Args: []string{"if=/dev/zero", "of=/root/test", "bs=1M", "count=9"}, // assume busybox itself accounts for a few MB
							}, garden.ProcessIO{})
							Expect(err).ToNot(HaveOccurred())
							Expect(dd.Wait()).ToNot(Equal(0))
						})
					})
				})

				Context("when the scope is exclusive", func() {
					BeforeEach(func() {
						quotaScope = garden.DiskLimitScopeExclusive
					})

					Context("and run a process that would exceed the quota due to the size of the rootfs (but doesnt since this is not included)", func() {
						It("does not kill the process", func() {
							dd, err := container.Run(garden.ProcessSpec{
								User: "root",
								Path: "dd",
								Args: []string{"if=/dev/zero", "of=/root/test", "bs=1M", "count=9"}, // should succeed, even though equivalent with 'total' scope does not
							}, garden.ProcessIO{Stdout: GinkgoWriter, Stderr: GinkgoWriter})
							Expect(err).ToNot(HaveOccurred())
							Expect(dd.Wait()).To(Equal(0))
						})
					})

					Context("and run a process that exceeds the quota", func() {
						It("kills the process", func() {
							dd, err := container.Run(garden.ProcessSpec{
								User: "root",
								Path: "dd",
								Args: []string{"if=/dev/zero", "of=/root/test", "bs=1M", "count=11"},
							}, garden.ProcessIO{})
							Expect(err).ToNot(HaveOccurred())
							Expect(dd.Wait()).ToNot(Equal(0))
						})
					})
				})

				Context("on a rootfs with pre-existing users", func() {
					BeforeEach(func() {
						rootfs = "docker:///cloudfoundry/preexisting_users"
					})

					Context("and run a process that exceeds the quota as bob", func() {
						BeforeEach(func() {
							byteSoftQuota = 10 * 1024 * 1024
							byteHardQuota = 10 * 1024 * 1024
							quotaScope = garden.DiskLimitScopeTotal
						})

						It("kills the process", func() {
							dd, err := container.Run(garden.ProcessSpec{
								User: "bob",
								Path: "dd",
								Args: []string{"if=/dev/zero", "of=/home/bob/test", "bs=1M", "count=11"},
							}, garden.ProcessIO{})
							Expect(err).ToNot(HaveOccurred())
							Expect(dd.Wait()).ToNot(Equal(0))
						})
					})

					Context("and run a process that exceeds the quota as alice", func() {
						BeforeEach(func() {
							byteSoftQuota = 10 * 1024 * 1024
							byteHardQuota = 10 * 1024 * 1024
							quotaScope = garden.DiskLimitScopeTotal
						})

						It("kills the process", func() {
							dd, err := container.Run(garden.ProcessSpec{
								User: "alice",
								Path: "dd",
								Args: []string{"if=/dev/zero", "of=/home/alice/test", "bs=1M", "count=11"},
							}, garden.ProcessIO{})
							Expect(err).ToNot(HaveOccurred())
							Expect(dd.Wait()).ToNot(Equal(0))
						})
					})

					Context("user alice is getting near the set limit", func() {
						JustBeforeEach(func() {
							metrics := func() garden.ContainerDiskStat {
								metricsAfter, err := container.Metrics()
								Expect(err).ToNot(HaveOccurred())
								return metricsAfter.DiskStat
							}

							totalMetrics := func() uint64 {
								return metrics().TotalBytesUsed
							}

							exclusiveMetrics := func() uint64 {
								return metrics().ExclusiveBytesUsed
							}

							Eventually(totalMetrics, BTRFS_WAIT_TIME, 30).Should(BeNumerically("~", uint64(6*1024*1024), uint64(4*1024*1024)))
							Eventually(exclusiveMetrics, BTRFS_WAIT_TIME, 30).Should(BeNumerically("~", uint64(0), uint64(4*1024*1024)))

							bytesUsed := totalMetrics()

							err := container.LimitDisk(garden.DiskLimits{
								ByteSoft: 10*1024*1024 + bytesUsed,
								ByteHard: 10*1024*1024 + bytesUsed,
							})
							Expect(err).ToNot(HaveOccurred())

							dd, err := container.Run(garden.ProcessSpec{
								User: "alice",
								Path: "dd",
								Args: []string{"if=/dev/zero", "of=/home/alice/test", "bs=1M", "count=8"},
							}, garden.ProcessIO{
								Stderr: GinkgoWriter,
								Stdout: GinkgoWriter,
							})
							Expect(err).ToNot(HaveOccurred())
							Expect(dd.Wait()).To(Equal(0))
						})

						It("kills the process if user bob tries to exceed the shared limit", func() {
							dd, err := container.Run(garden.ProcessSpec{
								User: "bob",
								Path: "dd",
								Args: []string{"if=/dev/zero", "of=/home/bob/test", "bs=1M", "count=3"},
							}, garden.ProcessIO{
								Stderr: GinkgoWriter,
								Stdout: GinkgoWriter,
							})
							Expect(err).ToNot(HaveOccurred())
							Expect(dd.Wait()).ToNot(Equal(0))
						})
					})
				})

				Context("that is privileged", func() {
					BeforeEach(func() {
						privilegedContainer = true
					})

					Context("and run a process that exceeds the quota as root", func() {
						It("kills the process", func() {
							dd, err := container.Run(garden.ProcessSpec{
								User: "root",
								Path: "dd",
								Args: []string{"if=/dev/zero", "of=/root/test", "bs=1M", "count=11"},
							}, garden.ProcessIO{})
							Expect(err).ToNot(HaveOccurred())
							Expect(dd.Wait()).ToNot(Equal(0))
						})
					})

					Context("and run a process that exceeds the quota as a new user", func() {
						It("kills the process", func() {
							addUser, err := container.Run(garden.ProcessSpec{
								User: "root",
								Path: "adduser",
								Args: []string{"-D", "-g", "", "bob"},
							}, garden.ProcessIO{})
							Expect(err).ToNot(HaveOccurred())
							Expect(addUser.Wait()).To(Equal(0))

							dd, err := container.Run(garden.ProcessSpec{
								User: "bob",
								Path: "dd",
								Args: []string{"if=/dev/zero", "of=/home/bob/test", "bs=1M", "count=11"},
							}, garden.ProcessIO{})
							Expect(err).ToNot(HaveOccurred())
							Expect(dd.Wait()).ToNot(Equal(0))
						})
					})
				})
			})

			Context("when multiple containers are created for the same user", func() {
				var container2 garden.Container
				var err error

				BeforeEach(func() {
					byteSoftQuota = 50 * 1024 * 1024
					byteHardQuota = 50 * 1024 * 1024
					quotaScope = garden.DiskLimitScopeTotal
				})

				JustBeforeEach(func() {
					container2, err = gardenClient.Create(garden.ContainerSpec{
						Privileged: privilegedContainer,
						RootFSPath: rootfs,
					})
					Expect(err).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					if container2 != nil {
						Expect(gardenClient.Destroy(container2.Handle())).To(Succeed())
					}
				})

				It("gives each container its own quota", func() {
					process, err := container.Run(garden.ProcessSpec{
						User: "vcap",
						Path: "dd",
						Args: []string{"if=/dev/urandom", "of=/home/vcap/some-file", "bs=1M", "count=40"},
					}, garden.ProcessIO{})
					Expect(err).ToNot(HaveOccurred())
					Expect(process.Wait()).To(Equal(0))

					process, err = container2.Run(garden.ProcessSpec{
						User: "vcap",
						Path: "dd",
						Args: []string{"if=/dev/urandom", "of=/home/vcap/some-file", "bs=1M", "count=40"},
					}, garden.ProcessIO{})
					Expect(err).ToNot(HaveOccurred())
					Expect(process.Wait()).To(Equal(0))
				})
			})
		})
	})
})
