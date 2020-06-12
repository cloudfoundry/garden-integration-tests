package garden_integration_tests_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/garden-integration-tests/testhelpers"
	"code.cloudfoundry.org/garden/client"
	"code.cloudfoundry.org/garden/client/connection"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

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
		host = "10.244.0.2"
	}
	return []byte(resolveHost(host))
}, func(data []byte) {
	gardenHost = string(data)
})

func TestGardenIntegrationTests(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(15 * time.Second)

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
		retryingConnection := testhelpers.RetryingConnection{Connection: connection.New("tcp", fmt.Sprintf("%s:%s", gardenHost, gardenPort))}
		gardenClient = client.New(&retryingConnection)
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
			Expect(destroyContainer(container)).To(Succeed())
		}
	})

	RunSpecs(t, "GardenIntegrationTests Suite")
}

func destroyContainer(c garden.Container) error {
	// ignoring the error since it can return unknown handle error
	theContainer, _ := gardenClient.Lookup(c.Handle())

	if theContainer != nil {
		return gardenClient.Destroy(theContainer.Handle())
	}

	return nil
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
	ctr, err := gardenClient.Create(garden.ContainerSpec{})
	Expect(err).NotTo(HaveOccurred())
	defer gardenClient.Destroy(ctr.Handle())

	var outBytes bytes.Buffer
	process, err := ctr.Run(garden.ProcessSpec{
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

func skipIfRootless() {
	if rootless() {
		Skip("behaviour being tested is either not relevant or not implemented in rootless")
	}
}

func rootless() bool {
	return os.Getenv("ROOTLESS") != ""
}

func skipIfWoot(reason string) {
	if woot() {
		Skip("Skipping this test because I am WOOT: " + reason)
	}
}

func woot() bool {
	return os.Getenv("WOOT") != ""
}

func skipIfShed() {
	if shed() {
		Skip("Skipping this test - not applicable to shed")
	}
}

func shed() bool {
	return os.Getenv("SHED") != ""
}

func skipIfContainerdForProcesses() {
	if os.Getenv("CONTAINERD_FOR_PROCESSES_ENABLED") != "" {
		Skip("Skipping because containerd support for processes is enabled")
	}
}

func setPrivileged() {
	privilegedContainer = true
	skipIfRootless()
}

func runProcessWithIO(container garden.Container, processSpec garden.ProcessSpec, pio garden.ProcessIO) int {
	proc, err := container.Run(processSpec, pio)
	Expect(err).NotTo(HaveOccurred())
	processExitCode, err := proc.Wait()
	Expect(err).NotTo(HaveOccurred())
	return processExitCode
}

func runProcess(container garden.Container, processSpec garden.ProcessSpec) (exitCode int, stdout, stderr *gbytes.Buffer) {
	stdout, stderr = gbytes.NewBuffer(), gbytes.NewBuffer()
	pio := garden.ProcessIO{
		Stdout: io.MultiWriter(stdout, GinkgoWriter),
		Stderr: io.MultiWriter(stderr, GinkgoWriter),
	}
	exitCode = runProcessWithIO(container, processSpec, pio)
	return
}

func runForStdin(container garden.Container, processSpec garden.ProcessSpec, stdinContent []byte) (exitCode int, stdout, stderr *gbytes.Buffer) {
	stdin := gbytes.BufferWithBytes(stdinContent)
	stdout, stderr = gbytes.NewBuffer(), gbytes.NewBuffer()
	pio := garden.ProcessIO{
		Stdin:  stdin,
		Stdout: io.MultiWriter(stdout, GinkgoWriter),
		Stderr: io.MultiWriter(stderr, GinkgoWriter),
	}
	exitCode = runProcessWithIO(container, processSpec, pio)
	return
}

func runForStdout(container garden.Container, processSpec garden.ProcessSpec) (stdout *gbytes.Buffer) {
	exitCode, stdout, _ := runProcess(container, processSpec)
	Expect(exitCode).To(Equal(0))
	return stdout
}

func readAll(r io.Reader) []byte {
	b, err := ioutil.ReadAll(r)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return b
}

func httpGet(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}
