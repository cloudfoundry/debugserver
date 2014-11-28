package cf_debug_server

import (
	"flag"
	"net"
	"net/http"
	"net/http/pprof"
	"runtime"
)

var debugAddr = flag.String(
	"debugAddr",
	"",
	"host:port for serving pprof debugging info",
)

var blockProfileRate = flag.Int(
	"blockProfileRate",
	0, // disabled
	"sample an average of one blocking event per rate nanoseconds spent blocked",
)

func Run() {
	if *debugAddr == "" {
		return
	}

	mux := http.NewServeMux()
	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))

	runtime.SetBlockProfileRate(*blockProfileRate)

	listener, err := net.Listen("tcp", *debugAddr)
	if err != nil {
		panic(err)
	}

	go http.Serve(listener, mux)
}

func Addr() string {
	return *debugAddr
}

func SetAddr(addr string) {
	debugAddr = &addr
}
