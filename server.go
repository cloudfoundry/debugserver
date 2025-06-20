package debugserver

import (
	"errors"
	"flag"
	"io"
	"net/http"
	"net/http/pprof"
	"runtime"
	"strconv"

	lager "code.cloudfoundry.org/lager/v3"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
)

const (
	DebugFlag = "debugAddr"
)

type DebugServerConfig struct {
	DebugAddress string `json:"debug_address"`
}

type ReconfigurableSinkInterface interface {
	SetMinLevel(level lager.LogLevel)
}

func AddFlags(flags *flag.FlagSet) {
	flags.String(
		DebugFlag,
		"",
		"host:port for serving pprof debugging info",
	)
}

func DebugAddress(flags *flag.FlagSet) string {
	dbgFlag := flags.Lookup(DebugFlag)
	if dbgFlag == nil {
		return ""
	}

	return dbgFlag.Value.String()
}

func validateloglevelrequest(w http.ResponseWriter, r *http.Request, level []byte) error {
	// Only POST method is allowed for setting log level.
	if r.Method != http.MethodPost {
		return errors.New("method not allowed, use POST")
	}

	// Only http is allowed for setting log level.
	if r.TLS != nil {
		return errors.New("invalid scheme, https is not allowed")
	}

	// Ensure the log level is not empty.
	if len(level) == 0 {
		return errors.New("log level cannot be empty")
	}
	return nil
}

func Runner(address string, sink ReconfigurableSinkInterface) ifrit.Runner {
	return http_server.New(address, Handler(sink))
}

func Run(address string, sink ReconfigurableSinkInterface) (ifrit.Process, error) {
	p := ifrit.Invoke(Runner(address, sink))
	select {
	case <-p.Ready():
	case err := <-p.Wait():
		return nil, err
	}
	return p, nil
}

func Handler(sink ReconfigurableSinkInterface) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle("/log-level", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the log level from the request body.
		level, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		// Validate the log level request.
		if err = validateloglevelrequest(w, r, level); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Set the logLevel based on the input.
		// Accepts: debug, info, error, fatal, or their short forms (d, i, e, f) or numeric values.
		// If the input is not recognized, return a 400 Bad Request.
		var logLevel lager.LogLevel
		switch string(level) {
		case "debug", "d", strconv.Itoa(int(lager.DEBUG)):
			logLevel = lager.DEBUG
		case "info", "i", strconv.Itoa(int(lager.INFO)):
			logLevel = lager.INFO
		case "error", "e", strconv.Itoa(int(lager.ERROR)):
			logLevel = lager.ERROR
		case "fatal", "f", strconv.Itoa(int(lager.FATAL)):
			logLevel = lager.FATAL
		default:
			http.Error(w, "Invalid log level provided: "+string(level), http.StatusBadRequest)
			return
		}
		// Set the log level in the sink.
		// The SetMinLevel sets the global zapcore conf.level for the logger.
		sink.SetMinLevel(logLevel)

		// Respond with a success message.
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("✅ /log-level was invoked with Level: " + string(level) + "\n"))
	}))
	mux.Handle("/block-profile-rate", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_rate, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}

		rate, err := strconv.Atoi(string(_rate))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			// #nosec G104 - ignore errors writing http response to avoid spamming logs during  DoS
			w.Write([]byte(err.Error()))
			return
		}

		if rate <= 0 {
			runtime.SetBlockProfileRate(0)
		} else {
			runtime.SetBlockProfileRate(rate)
		}
	}))
	mux.Handle("/mutex-profile-fraction", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_rate, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}

		rate, err := strconv.Atoi(string(_rate))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			// #nosec G104 - ignore errors writing http response to avoid spamming logs during  DoS
			w.Write([]byte(err.Error()))
			return
		}

		if rate <= 0 {
			runtime.SetMutexProfileFraction(0)
		} else {
			runtime.SetMutexProfileFraction(rate)
		}
	}))

	return mux
}
