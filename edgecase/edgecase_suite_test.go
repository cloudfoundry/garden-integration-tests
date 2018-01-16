package edgecase_test

import (
	"fmt"
	"os"
	"testing"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/garden/client"
	"code.cloudfoundry.org/garden/client/connection"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var gardenClient garden.Client

func TestEdgecase(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Edgecase Suite")
}

var _ = BeforeSuite(func() {
	gardenHost := getenv("GARDEN_ADDRESS", "10.244.0.2")
	gardenPort := getenv("GARDEN_PORT", "7777")
	gardenClient = client.New(connection.New("tcp", fmt.Sprintf("%s:%s", gardenHost, gardenPort)))
})

func getenv(key string, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		if defaultValue == "" {
			Fail(fmt.Sprintf("please set %s", key))
		}
		return defaultValue
	}
	return val
}
