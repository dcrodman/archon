// The server command is the main entrypoint for running archon. It takes
// care of initializing everything as well as running as many servers are
// needed for a fully functional server backend.
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/data"
	"github.com/dcrodman/archon/internal/debug"
	"github.com/dcrodman/archon/internal/server/character"
	"github.com/dcrodman/archon/internal/server/frontend"
	"github.com/dcrodman/archon/internal/server/login"
	"github.com/dcrodman/archon/internal/server/patch"
	"github.com/dcrodman/archon/internal/server/shipgate"
	"github.com/spf13/viper"
	"os"
	"sync"
)

func main() {
	flag.Parse()

	fmt.Printf("Archon PSO Backend, Copyright (C) 2014 Andrew Rodman\n" +
		"=====================================================\n" +
		"This program is free software: you can redistribute it and/or\n" +
		"modify it under the terms of the GNU General Public License as\n" +
		"published by the Free Software Foundation, either version 3 of\n" +
		"the License, or (at your option) any later version. This program\n" +
		"is distributed WITHOUT ANY WARRANTY; See LICENSE for details.\n\n")

	archon.Load()
	fmt.Println("configuration loaded from", archon.ConfigFileUsed())

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
		os.Exit(1)
	}
	defer data.Shutdown()

	fmt.Printf("connected to database %s:%d\n\n",
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
	)

	if debug.Enabled() {
		debug.StartUtilities()
	}

	startServers()
}

// Register all of the server handlers and their corresponding ports. This runner
// assumes one instance of each type of server will be deployed on this host (with
// the exception of the Block server since the number is configurable).
func startServers() {
	hostname := viper.GetString("hostname")
	shipInfoServiceAddr := fmt.Sprintf(
		"%s:%s", hostname, viper.GetString("shipgate_server.meta_service_port"))
	shipgateServiceAddr := fmt.Sprintf(
		"%s:%s", hostname, viper.GetString("shipgate_server.ship_service_port"))

	ctx, _ := context.WithCancel(context.Background())
	shipgateWg := startShipgate(ctx, shipInfoServiceAddr, shipgateServiceAddr)

	registerServers(shipInfoServiceAddr, shipgateServiceAddr)

	frontend.SerHostname(hostname)
	serverWg := frontend.Start(ctx)

	shipgateWg.Wait()
	serverWg.Wait()
}

func startShipgate(ctx context.Context, shipInfoServiceAddress, shipServiceAddress string) (wg *sync.WaitGroup) {
	var shipgateWg sync.WaitGroup
	shipgateWg.Add(1)

	go func() {
		err := shipgate.Start(ctx, shipInfoServiceAddress, shipServiceAddress)

		if err != nil {
			fmt.Println("failed to start ship server:", err)
			os.Exit(1)
		}

		wg.Done()
	}()

	return &shipgateWg
}

func registerServers(shipInfoServiceAddress, shipgateServiceAddress string) {
	dataPort := viper.GetString("patch_server.data_port")
	frontend.AddServer(dataPort, patch.NewDataServer("DATA"))
	frontend.AddServer(viper.GetString("patch_server.patch_port"), patch.NewServer("PATCH", dataPort))

	characterPort := viper.GetString("character_server.port")
	frontend.AddServer(characterPort, character.NewServer("CHARACTER", shipInfoServiceAddress))
	frontend.AddServer(viper.GetString("login_server.port"), login.NewServer("LOGIN", characterPort))
}
