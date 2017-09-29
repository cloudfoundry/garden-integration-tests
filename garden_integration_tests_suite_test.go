package garden_integration_tests_test

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/garden/client"
	"code.cloudfoundry.org/garden/client/connection"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const KNOWN_NESTED_UMOUNT_ERR = "unmounting the loop device: unmounting file: exit status 1"

var (
	gardenHost            string
	gardenPort            string
	gardenDebugPort       string
	gardenClient          garden.Client
	container             garden.Container
	containerCreateErr    error
	assertContainerCreate bool

	handle              string
	imageRef            garden.ImageRef
	networkSpec         string
	privilegedContainer bool
	properties          garden.Properties
	limits              garden.Limits
	env                 []string
	ginkgoIO            garden.ProcessIO = garden.ProcessIO{
		Stdout: GinkgoWriter,
		Stderr: GinkgoWriter,
	}
)

// We suspect that bosh powerdns lookups have a low success rate (less than
// 99%) and when it fails, we get an empty string IP address instead of an
// actual error.
// Therefore, we explicity look up the IP once at the start of the suite with
// retries to minimise flakes.
func resolveHost(host string) string {
	if net.ParseIP(host) != nil {
		return host
	}

	var ip net.IP
	Eventually(func() error {
		ips, err := net.LookupIP(host)
		if err != nil {
			return err
		}
		if len(ips) == 0 {
			return errors.New("0 IPs returned from DNS")
		}
		ip = ips[0]
		return nil
	}, time.Minute, time.Second*5).Should(Succeed())

	return ip.String()
}

var _ = SynchronizedBeforeSuite(func() []byte {
	host := os.Getenv("GARDEN_ADDRESS")
	if host == "" {
		host = "10.244.16.6"
	}
	return []byte(resolveHost(host))
}, func(data []byte) {
	gardenHost = string(data)
})

func TestGardenIntegrationTests(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(5 * time.Second)

	BeforeEach(func() {
		assertContainerCreate = true
		handle = ""
		imageRef = garden.ImageRef{}
		networkSpec = ""
		privilegedContainer = false
		properties = garden.Properties{}
		limits = garden.Limits{}
		env = []string{}
		gardenPort = os.Getenv("GARDEN_PORT")
		if gardenPort == "" {
			gardenPort = "7777"
		}
		gardenDebugPort = os.Getenv("GARDEN_DEBUG_PORT")
		if gardenDebugPort == "" {
			gardenDebugPort = "17013"
		}
		gardenClient = client.New(connection.New("tcp", fmt.Sprintf("%s:%s", gardenHost, gardenPort)))
	})

	JustBeforeEach(func() {
		container, containerCreateErr = gardenClient.Create(garden.ContainerSpec{
			Handle:     handle,
			Image:      imageRef,
			Privileged: privilegedContainer,
			Properties: properties,
			Env:        env,
			Limits:     limits,
			Network:    networkSpec,
		})

		if container != nil {
			fmt.Fprintf(GinkgoWriter, "Container handle: %s\n", container.Handle())
		}

		if assertContainerCreate {
			Expect(containerCreateErr).ToNot(HaveOccurred())
		}
	})

	AfterEach(func() {
		if container != nil {
			// ignoring the error since it can return unknown handle error
			theContainer, _ := gardenClient.Lookup(container.Handle())

			if theContainer != nil {
				destroyContainer(gardenClient, container.Handle())
			}
		}
	})

	RunSpecs(t, "GardenIntegrationTests Suite")
}

func getContainerHandles() []string {
	containers, err := gardenClient.Containers(nil)
	Expect(err).ToNot(HaveOccurred())

	handles := make([]string, len(containers))
	for i, c := range containers {
		handles[i] = c.Handle()
	}

	return handles
}

func createUser(container garden.Container, username string) {
	if container == nil {
		return
	}

	process, err := container.Run(garden.ProcessSpec{
		User: "root",
		Path: "sh",
		Args: []string{"-c", fmt.Sprintf("id -u %s || adduser -D %s", username, username)},
	}, garden.ProcessIO{
		Stdout: GinkgoWriter,
		Stderr: GinkgoWriter,
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(process.Wait()).To(Equal(0))
}

func getKernelVersion() (int, int) {
	container, err := gardenClient.Create(garden.ContainerSpec{})
	Expect(err).NotTo(HaveOccurred())
	defer gardenClient.Destroy(container.Handle())

	var outBytes bytes.Buffer
	process, err := container.Run(garden.ProcessSpec{
		User: "root",
		Path: "uname",
		Args: []string{"-r"},
	}, garden.ProcessIO{
		Stdout: &outBytes,
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(process.Wait()).To(Equal(0))

	vSplit := strings.Split(outBytes.String(), ".")
	major, err := strconv.Atoi(vSplit[0])
	Expect(err).NotTo(HaveOccurred())
	minor, err := strconv.Atoi(vSplit[1])
	Expect(err).NotTo(HaveOccurred())

	return major, minor
}

func destroyContainer(client garden.Client, handle string) {
	err := client.Destroy(handle)
	if err != nil && os.Getenv("NESTED") == "true" && err.Error() == KNOWN_NESTED_UMOUNT_ERR {
		fmt.Printf("Ignoring known nested umount error: %s\n", err)
		return
	}
	Expect(err).NotTo(HaveOccurred())
}

func skipIfRootless() {
	if rootless() {
		Skip("behaviour being tested is either not relevant or not implemented in rootless")
	}
}

func rootless() bool {
	return os.Getenv("ROOTLESS") != ""
}
