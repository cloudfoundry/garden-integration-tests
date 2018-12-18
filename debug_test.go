package garden_integration_tests_test

import (
	"encoding/json"
	"fmt"
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type Debug struct {
	MemStats runtime.MemStats `json:"memstats"`
}

func loadDebug() Debug {
	response := httpGet(fmt.Sprintf("http://%s:%s/debug/vars", gardenHost, gardenDebugPort))
	defer response.Body.Close()

	debug := Debug{}
	Expect(json.Unmarshal(readAll(response.Body), &debug)).To(Succeed())

	return debug
}

var _ = Describe("Debug", func() {
	Describe("Memory", func() {
		It("should have non-zero allocated memory", func() {
			debug := loadDebug()
			Expect(debug.MemStats.Alloc).NotTo(BeZero())
		})
	})
})
