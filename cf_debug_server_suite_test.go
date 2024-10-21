package debugserver_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

var address string

func TestCfDebugServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CfDebugServer Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	return nil
}, func(encodedBuiltArtifacts []byte) {
	address = fmt.Sprintf("127.0.0.1:%d", 10000+GinkgoParallelProcess())
})
