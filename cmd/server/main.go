// The server command is the main entrypoint for running archon. It takes
// care of initializing everything as well as running as many servers are
// needed for a fully functional server backend.
package main

import (
	"flag"
	"fmt"
	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/data"
	"github.com/dcrodman/archon/debug"
	"github.com/dcrodman/archon/server"
	"github.com/dcrodman/archon/server/character"
	"github.com/dcrodman/archon/server/login"
	"github.com/dcrodman/archon/server/patch"
	"github.com/dcrodman/archon/server/shipgate"
	"github.com/spf13/viper"
	"os"
	"sync"
	"time"
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

	fmt.Println("configuration loaded from", archon.ConfigFileUsed())

	if debug.Enabled() {
		go debug.StartPprofServer()
	}

	archon.InitLogger()

	dataSource := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
		viper.GetString("database.name"),
		viper.GetString("database.username"),
		viper.GetString("database.password"),
		viper.GetString("database.sslmode"),
	)
	if err := data.Initialize(dataSource); err != nil {
		fmt.Println(err.Error())
	}
	defer data.Shutdown()

	fmt.Printf("connected to database %s:%d\n\n",
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
	)

	startServers()
}

// Register all of the server handlers and their corresponding ports. This runner
// assumes one instance of each type of server will be deployed on this host (with
// the exception of the Block server since the number is configurable).
func startServers() {
	hostname := viper.GetString("hostname")
	server.SetHostname(hostname)

	var wg sync.WaitGroup
	wg.Add(1)

	shipgateMetaAddr, _ := startShipgate(hostname, &wg)

	dataServer := patch.NewDataServer(
		"DATA",
		viper.GetString("patch_server.data_port"),
	)
	patchServer := patch.NewPatchServer(
		"PATCH",
		viper.GetString("patch_server.patch_port"),
		dataServer.Port(),
	)
	characterServer := character.NewServer(
		"CHARACTER",
		viper.GetString("character_server.port"),
		shipgateMetaAddr,
	)
	loginServer := login.NewServer(
		"LOGIN",
		viper.GetString("login_server.port"),
		characterServer.Port(),
	)

	//ship.NewServer(),
	//shipPort, _ := strconv.ParseInt(archon.Config.ShipServer.Port, 10, 16)

	// The available block ports will depend on how the server is configured,
	// so once we've read the config then add the server entries on the fly.
	//for i := 1; i <= archon.Config.ShipServer.NumBlocks; i++ {
	//	blockServer := block.NewServer(fmt.Sprintf("BLOCK%d", i), shipPort+int64(i))
	//	startBlockingServer(blockServer, &wg)
	//}

	servers := []server.Server{
		patchServer,
		dataServer,
		loginServer,
		characterServer,
	}

	wg.Add(len(servers))

	launchBlockingServers(servers, &wg)

	wg.Wait()
}

func launchBlockingServers(servers []server.Server, wg *sync.WaitGroup) {
	for _, s := range servers {
		if err := s.Init(); err != nil {
			fmt.Printf("failed to initialize %s server: %s\n", s.Name(), err)
			os.Exit(1)
		}
	}

	fmt.Println()

	for _, s := range servers {
		go func(s server.Server) {
			server.Start(s)
			wg.Done()
		}(s)
	}
}

func startShipgate(hostname string, wg *sync.WaitGroup) (string, string) {
	shipgateMetaAddr := fmt.Sprintf(
		"%s:%s", hostname, viper.GetString("shipgate_server.meta_service_port"))
	shipgateAddr := fmt.Sprintf(
		"%s:%s", hostname, viper.GetString("shipgate_server.ship_service_port"))

	go func() {
		if err := shipgate.Start(shipgateMetaAddr, shipgateAddr); err != nil {
			fmt.Println("failed to start ship server:", err)
			os.Exit(1)
		}

		wg.Done()
	}()

	// Hack in a second for the shipgate to initialize.
	time.Sleep(time.Second)
	fmt.Println()

	return shipgateMetaAddr, shipgateAddr
}
