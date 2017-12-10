/*
* Archon PSO Server
* Copyright (C) 2014 Andrew Rodman
*
* This program is free software: you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License
* along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */
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
