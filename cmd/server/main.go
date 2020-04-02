package main

import (
	"flag"
	"fmt"
	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/server"
	"github.com/dcrodman/archon/server/patch"
	"github.com/spf13/viper"
	"net/http"
	"runtime/pprof"
	"sync"
)

func main() {
	flag.Parse()

	fmt.Printf("Archon PSO Server, Copyright (C) 2014 Andrew Rodman\n" +
		"=====================================================\n" +
		"This program is free software: you can redistribute it and/or\n" +
		"modify it under the terms of the GNU General Public License as\n" +
		"published by the Free Software Foundation, either version 3 of\n" +
		"the License, or (at your option) any later version. This program\n" +
		"is distributed WITHOUT ANY WARRANTY; See LICENSE for details.\n\n")

	fmt.Printf("configuration loaded from %s\n\n", archon.ConfigFileUsed())

	archon.InitLogger()
	//
	//database, err := archon.InitializeDatabase(
	//	viper.GetString("database.host"),
	//	viper.GetString("database.port"),
	//)
	//if err != nil {
	//	fmt.Println("Failed: " + err.Error())
	//	os.Exit(1)
	//}
	//defer database.Close()
	//fmt.Print("Done.\n\n")

	if viper.GetBool("debug_mode") {
		startDebugServer()
	}

	startServers()
}

// If the server was configured in debug mode, this function will launch an HTTP server
// that responds with pprof output containing the stack traces of all running goroutines.
func startDebugServer() {
	webPort := viper.GetString("web.http_port")

	fmt.Println("opening Debug port on " + webPort)
	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		pprof.Lookup("goroutine").WriteTo(resp, 1)
	})

	go http.ListenAndServe(":"+webPort, nil)
}

// Register all of the server handlers and their corresponding ports. This runner
// assumes one instance of each type of server will be deployed on this host (with
// the exception of the Block server since the number is configurable).
func startServers() {
	server.SetHostname(viper.GetString("hostname"))

	dataServer := patch.NewDataServer(
		"DATA",
		viper.GetString("patch_server.data_port"),
	)
	patchServer := patch.NewPatchServer(
		"PATCH",
		viper.GetString("patch_server.patch_port"),
		dataServer.Port(),
	)

	//login.NewServer("LOGIN", viper.GetString("login_server.login_port"), viper.GetString("login_server.character_port")),
	//character.NewServer(),
	//ship.NewServer(),
	//shipgate.NewServer(),

	//shipPort, _ := strconv.ParseInt(archon.Config.ShipServer.Port, 10, 16)

	// The available block ports will depend on how the server is configured,
	// so once we've read the config then add the server entries on the fly.
	//for i := 1; i <= archon.Config.ShipServer.NumBlocks; i++ {
	//	blockServer := block.NewServer(fmt.Sprintf("BLOCK%d", i), shipPort+int64(i))
	//	startBlockingServer(blockServer, &wg)
	//}

	servers := []server.Server{
		dataServer,
		patchServer,
	}

	var wg sync.WaitGroup
	for _, s := range servers {
		wg.Add(1)
		startBlockingServer(s, &wg)
	}

	wg.Wait()
}

func startBlockingServer(s server.Server, wg *sync.WaitGroup) {
	go func(s server.Server) {
		server.Start(s)
		wg.Done()
	}(s)
}
