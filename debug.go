package main

import (
	"fmt"
	"net/http"
	"runtime/pprof"
)

// StartDebugServer will, If we're in debug mode, spawn off an HTTP server that dumps
// pprof output containing the stack traces of all running goroutines.
func StartDebugServer() {
	if config.DebugMode {
		fmt.Println("Opening Debug port on " + config.WebPort)
		http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
			pprof.Lookup("goroutine").WriteTo(resp, 1)
		})
		go http.ListenAndServe(":"+config.WebPort, nil)
	}
}

// DebugLog is a trivial utility that will only write message if debug mode is on.
func DebugLog(message string) {
	if config.DebugMode {
		fmt.Println(message)
	}
}
