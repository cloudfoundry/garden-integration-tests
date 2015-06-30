package garden_integration_tests_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGardenIntegrationTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GardenIntegrationTests Suite")
}
