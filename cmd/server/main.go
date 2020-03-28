package main

import (
	"flag"
	"fmt"
	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/server"
	"github.com/dcrodman/archon/server/block"
	"github.com/dcrodman/archon/server/character"
	"github.com/dcrodman/archon/server/login"
	"github.com/dcrodman/archon/server/patch"
	"github.com/dcrodman/archon/server/ship"
	"github.com/dcrodman/archon/server/shipgate"
	"net/http"
	"os"
	"runtime/pprof"
	"strconv"
)

func main() {
	flag.Parse()

	fmt.Printf("Archon PSO Server, Copyright (C) 2014 Andrew Rodman\n" +
		"=====================================================\n" +
		"This program is free software: you can redistribute it and/or\n" +
		"modify it under the terms of the GNU General Public License as\n" +
		"published by the Free Software Foundation, either version 3 of\n" +
		"the License, or (at your option) any later version.\n" +
		"This program is distributed WITHOUT ANY WARRANTY; See LICENSE for details.\n\n")

	fmt.Printf("Loaded config from %s\n\n", archon.ConfigFileUsed())

	database, err := archon.InitializeDatabase(archon.Config.Database.Host, archon.Config.Database.Port)
	if err != nil {
		fmt.Println("Failed: " + err.Error())
		os.Exit(1)
	}
	defer database.Close()
	fmt.Print("Done.\n\n")

	if archon.Config.DebugMode {
		startDebugServer()
	}

	createController().Start()
}

// startDebugServer will, If we're in debug mode, spawn off an HTTP server that dumps
// pprof output containing the stack traces of all running goroutines.
func startDebugServer() {
	fmt.Println("Opening Debug port on " + archon.Config.WebServer.Port)
	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		pprof.Lookup("goroutine").WriteTo(resp, 1)
	})

	go http.ListenAndServe(":"+archon.Config.WebServer.Port, nil)
}

// Register all of the server handlers and their corresponding ports. This runner
// assumes one instance of each type of server will be deployed on this host (with
// the exception of the Block server since the number is configurable).
func createController() *server.Controller {
	controller := server.New(archon.Config.Hostname)

	servers := []server.Server{
		patch.NewServer(),
		patch.NewDataServer(),
		login.NewServer(),
		character.NewServer(),
		ship.NewServer(),
		shipgate.NewServer(),
	}
	for _, s := range servers {
		controller.RegisterServer(s)
	}

	shipPort, _ := strconv.ParseInt(archon.Config.ShipServer.Port, 10, 16)

	// The available block ports will depend on how the server is configured,
	// so once we've read the config then add the server entries on the fly.
	for i := 1; i <= archon.Config.ShipServer.NumBlocks; i++ {
		blockServer := block.NewServer(fmt.Sprintf("BLOCK%d", i), shipPort+int64(i))
		controller.RegisterServer(blockServer)
	}

	return controller
}
