package debugserver_test

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"

	cf_debug_server "code.cloudfoundry.org/debugserver"
	lager "code.cloudfoundry.org/lager/v3"
	"github.com/tedsuo/ifrit"
	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("CF Debug Server", func() {
	var (
		logBuf *gbytes.Buffer
		sink   *lager.ReconfigurableSink

		process ifrit.Process
	)

	BeforeEach(func() {
		logBuf = gbytes.NewBuffer()
		sink = lager.NewReconfigurableSink(
			lager.NewWriterSink(logBuf, lager.DEBUG),
			// permit no logging by default, for log reconfiguration below
			lager.FATAL+1,
		)
	})

	AfterEach(func() {
		ginkgomon.Interrupt(process)
	})

	Describe("AddFlags", func() {
		It("adds flags to the flagset", func() {
			flags := flag.NewFlagSet("test", flag.ContinueOnError)
			cf_debug_server.AddFlags(flags)

			f := flags.Lookup(cf_debug_server.DebugFlag)
			Expect(f).NotTo(BeNil())
		})
	})

	Describe("DebugAddress", func() {
		Context("when flags are not added", func() {
			It("returns the empty string", func() {
				flags := flag.NewFlagSet("test", flag.ContinueOnError)
				Expect(cf_debug_server.DebugAddress(flags)).To(Equal(""))
			})
		})

		Context("when flags are added", func() {
			var flags *flag.FlagSet
			BeforeEach(func() {
				flags = flag.NewFlagSet("test", flag.ContinueOnError)
				cf_debug_server.AddFlags(flags)
			})

			Context("when set", func() {
				It("returns the address", func() {
					flags.Parse([]string{"-debugAddr", address})

					Expect(cf_debug_server.DebugAddress(flags)).To(Equal(address))
				})
			})

			Context("when not set", func() {
				It("returns the empty string", func() {
					Expect(cf_debug_server.DebugAddress(flags)).To(Equal(""))
				})
			})
		})
	})

	Describe("Run", func() {
		It("serves debug information", func() {
			var err error
			process, err = cf_debug_server.Run(address, sink)
			Expect(err).NotTo(HaveOccurred())

			debugResponse, err := http.Get(fmt.Sprintf("http://%s/debug/pprof/goroutine", address))
			Expect(err).NotTo(HaveOccurred())
			defer debugResponse.Body.Close()
		})

		Context("when the address is already in use", func() {
			var listener net.Listener

			BeforeEach(func() {
				var err error
				listener, err = net.Listen("tcp", address)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				listener.Close()
			})

			It("returns an error", func() {
				var err error
				process, err = cf_debug_server.Run(address, sink)
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(&net.OpError{}))
				netErr := err.(*net.OpError)
				Expect(netErr.Op).To(Equal("listen"))
			})
		})
	})

	Describe("checking log-level endpoint with various inputs", func() {
		var (
			req    *http.Request
			writer *httptest.ResponseRecorder
		)

		BeforeEach(func() {
			writer = httptest.NewRecorder()
			req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/log-level", address), nil)
		})

		Context("valid log levels", func() {
			DescribeTable("returns normalized log level",
				func(input string, expected string) {
					req.Body = io.NopCloser(strings.NewReader(input))
					levelBytes, _ := io.ReadAll(req.Body)

					actual, err := cf_debug_server.ValidateAndNormalize(writer, req, levelBytes)
					Expect(err).ToNot(HaveOccurred())
					Expect(actual).To(Equal(expected))
				},

				// Debug
				Entry("debug - 0", "0", "debug"),
				Entry("debug - d", "d", "debug"),
				Entry("debug - debug", "debug", "debug"),
				Entry("debug - DEBUG", "DEBUG", "debug"),
				Entry("debug - DeBuG", "DeBuG", "debug"),

				// Info
				Entry("info - 1", "1", "info"),
				Entry("info - i", "i", "info"),
				Entry("info - info", "info", "info"),
				Entry("info - INFO", "INFO", "info"),
				Entry("info - InFo", "InFo", "info"),

				// Warn
				Entry("warn - 2", "2", "warn"),
				Entry("warn - w", "w", "warn"),
				Entry("warn - warn", "warn", "warn"),
				Entry("warn - WARN", "WARN", "warn"),
				Entry("warn - wARn", "wARn", "warn"),

				// Error
				Entry("error - 3", "3", "error"),
				Entry("error - e", "e", "error"),
				Entry("error - error", "error", "error"),
				Entry("error - ERROR", "ERROR", "error"),
				Entry("error - eRroR", "eRroR", "error"),

				// Fatal
				Entry("fatal - 4", "4", "fatal"),
				Entry("fatal - f", "f", "fatal"),
				Entry("fatal - fatal", "fatal", "fatal"),
				Entry("fatal - FATAL", "FATAL", "fatal"),
				Entry("fatal - FaTaL", "FaTaL", "fatal"),
			)
		})

		Context("invalid log levels", func() {
			It("fails on unsupported level", func() {
				level := []byte("invalid")
				actual, err := cf_debug_server.ValidateAndNormalize(writer, req, level)
				Expect(err).To(HaveOccurred())
				Expect(actual).To(BeEmpty())
			})

			It("fails on empty level", func() {
				level := []byte("")
				actual, err := cf_debug_server.ValidateAndNormalize(writer, req, level)
				Expect(err).To(HaveOccurred())
				Expect(actual).To(BeEmpty())
			})
		})

		Context("invalid request method", func() {
			It("returns error for non-POST", func() {
				req.Method = http.MethodGet
				actual, err := cf_debug_server.ValidateAndNormalize(writer, req, []byte("info"))
				Expect(err).To(MatchError(ContainSubstring("method not allowed")))
				Expect(actual).To(BeEmpty())
			})
		})

		Context("invalid TLS scheme", func() {
			It("returns error if TLS is used", func() {
				req.TLS = &tls.ConnectionState{}
				actual, err := cf_debug_server.ValidateAndNormalize(writer, req, []byte("debug"))
				Expect(err).To(MatchError(ContainSubstring("invalid scheme")))
				Expect(actual).To(BeEmpty())
			})
		})

		It("returns error if the request is made over HTTPS", func() {
			// Simulate HTTPS by assigning a non-nil TLS connection state
			req.TLS = &tls.ConnectionState{}
			actual, err := cf_debug_server.ValidateAndNormalize(writer, req, []byte("debug"))

			Expect(err).To(MatchError(ContainSubstring("invalid scheme")))
			Expect(actual).To(BeEmpty())
		})

	})

})
