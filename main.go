package main

import (
	"container/list"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/pprof"
	"strconv"

	"github.com/sirupsen/logrus"
)

// Global variables that should not be globals at some point.
var Log *logrus.Logger

func main() {
	flag.Parse()

	fmt.Printf("Archon PSO Server, Copyright (C) 2014 Andrew Rodman\n" +
		"=====================================================\n" +
		"This program is free software: you can redistribute it and/or\n" +
		"modify it under the terms of the GNU General Public License as\n" +
		"published by the Free Software Foundation, either version 3 of\n" +
		"the License, or (at your option) any later version.\n" +
		"This program is distributed WITHOUT ANY WARRANTY; See LICENSE for details.\n\n")

	fmt.Printf("Loaded config from %s\n\n", ConfigFileUsed())

	database, err := InitializeDatabase(Config.Database.Host, Config.Database.Port)
	if err != nil {
		fmt.Println("Failed: " + err.Error())
		os.Exit(1)
	}
	defer database.Close()
	fmt.Print("Done.\n\n")

	initializeLogger(Config.Logfile)

	if Config.DebugMode {
		startDebugServer()
	}

	c := controller{
		host:        Config.Hostname,
		servers:     make([]Server, 0),
		connections: &clientList{clients: list.New()},
	}
	registerServers(&c)

	// Start up all of our servers and block until they exit.
	wg := c.start()
	if wg != nil {
		wg.Wait()
	}
}

// startDebugServer will, If we're in debug mode, spawn off an HTTP server that dumps
// pprof output containing the stack traces of all running goroutines.
func startDebugServer() {
	fmt.Println("Opening Debug port on " + Config.WebServer.Port)
	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		pprof.Lookup("goroutine").WriteTo(resp, 1)
	})

	go http.ListenAndServe(":"+Config.WebServer.Port, nil)
}

// Set up the logger to write to the specified filename.
func initializeLogger(filename string) {
	var w io.Writer
	var err error
	if filename != "" {
		w, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Println("ERROR: Failed to open Log file " + Config.Logfile)
			os.Exit(1)
		}
	} else {
		w = os.Stdout
	}

	logLvl, err := logrus.ParseLevel(Config.LogLevel)
	if err != nil {
		fmt.Println("ERROR: Failed to parse Log level: " + err.Error())
		os.Exit(1)
	}

	Log = &logrus.Logger{
		Out: w,
		Formatter: &logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
			DisableSorting:  true,
		},
		Hooks: make(logrus.LevelHooks),
		Level: logLvl,
	}
}

// Register all of the server handlers and their corresponding ports.
func registerServers(controller *controller) {
	servers := []Server{
		new(PatchServer),
		new(DataServer),
		new(LoginServer),
		new(CharacterServer),
		new(ShipgateServer),
		new(ShipServer),
	}
	for _, server := range servers {
		controller.registerServer(server)
	}

	shipPort, _ := strconv.ParseInt(Config.ShipServer.Port, 10, 16)

	// The available block ports will depend on how the server is configured,
	// so once we've read the config then add the server entries on the fly.
	for i := 1; i <= Config.ShipServer.NumBlocks; i++ {
		controller.registerServer(&BlockServer{
			name: fmt.Sprintf("BLOCK%d", i),
			port: strconv.FormatInt(shipPort+int64(i), 10),
		})
	}
}
