// The server command is the main entrypoint for running archon. It takes
// care of initializing everything as well as running as many servers are
// needed for a fully functional server backend.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/spf13/viper"

	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/data"
	"github.com/dcrodman/archon/internal/debug"
	"github.com/dcrodman/archon/internal/server"
	"github.com/dcrodman/archon/internal/server/block"
	"github.com/dcrodman/archon/internal/server/character"
	"github.com/dcrodman/archon/internal/server/login"
	"github.com/dcrodman/archon/internal/server/patch"
	"github.com/dcrodman/archon/internal/server/ship"
	"github.com/dcrodman/archon/internal/server/shipgate"
)

const databaseURITemplate = "host=%s port=%d dbname=%s user=%s password=%s sslmode=%s"

var config = flag.String("config", "./", "Path to the directory containing the server config file")

func main() {
	flag.Parse()
	archon.Load(*config)
	archon.InitLogger()

	archon.Log.Info("Archon PSO Backend, Copyright (C) 2014 Andrew Rodman\n" +
		"=====================================================\n" +
		"This program is free software: you can redistribute it and/or\n" +
		"modify it under the terms of the GNU General Public License as\n" +
		"published by the Free Software Foundation, either version 3 of\n" +
		"the License, or (at your option) any later version. This program\n" +
		"is distributed WITHOUT ANY WARRANTY; See LICENSE for details.")

	archon.Log.Infof("loaded configuration from %s", *config)

	// Change to the same directory as the config file so that any relative
	// paths in the config file will resolve.
	if err := os.Chdir(filepath.Dir(*config)); err != nil {
		archon.Log.Errorf("failed to change to config directory: %v", err)
		os.Exit(1)
	}

	// Connect to the database.
	if err := data.Initialize(dataSource(), debug.Enabled()); err != nil {
		archon.Log.Errorf(err.Error())
		os.Exit(1)
	}
	defer data.Shutdown()

	archon.Log.Infof("connected to database %s:%d",
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
	)

	// Start any debug utilities if we're configured to do so.
	if debug.Enabled() {
		debug.StartUtilities()
	}

	// Set up all of the servers we want to run.
	patchPort := viper.GetString("patch_server.patch_port")
	dataPort := viper.GetString("patch_server.data_port")
	loginPort := viper.GetString("login_server.port")
	characterPort := viper.GetString("character_server.port")
	shipPort := viper.GetString("ship_server.port")

	shipgateAddr := buildAddress(viper.GetString("shipgate_server.port"))

	// Automatically configure the block servers based on the number of
	// ship blocks requested.
	var blocks []ship.Block
	var blockServers []*server.Frontend
	for i := 1; i <= viper.GetInt("ship_server.num_blocks"); i++ {
		name := fmt.Sprintf("BLOCK%02d", i)
		address := buildAddress(viper.GetInt("block_server.port") + i)

		blocks = append(blocks, ship.Block{
			Name: name, Address: address, ID: i,
		})
		blockServers = append(blockServers, &server.Frontend{
			Address: address, Backend: block.NewServer(name),
		})
	}

	servers := []*server.Frontend{
		{
			Address: buildAddress(patchPort),
			Backend: patch.NewServer("PATCH", dataPort),
		},
		{
			Address: buildAddress(dataPort),
			Backend: patch.NewDataServer("DATA"),
		},
		{
			Address: buildAddress(loginPort),
			Backend: login.NewServer("LOGIN", characterPort),
		},
		{
			Address: buildAddress(characterPort),
			Backend: character.NewServer("CHARACTER", shipgateAddr),
		},
		// TODO: Eventually the ship and block servers should be able to be run
		// independently of the other four servers (and possibly each other).
		{
			Address: buildAddress(shipPort),
			Backend: ship.NewServer("SHIP", blocks, shipgateAddr),
		},
	}
	servers = append(servers, blockServers...)

	// Bind the server loops to one top-level server context so that we can shut down cleanly.
	ctx, cancel := context.WithCancel(context.Background())

	// Start the shipgate gRPC server and make sure it launches before the other servers start.
	readyChan := make(chan bool)
	errChan := make(chan error)
	go shipgate.Start(ctx, shipgateAddr, readyChan, errChan)
	go func() {
		if err := <-errChan; err != nil {
			archon.Log.Errorf("exiting due to SHIPGATE error: %v", err)
			os.Exit(1)
		}
	}()
	<-readyChan

	// Start all of our servers. Failure to initialize one of the registered servers is considered terminal.
	var serverWg sync.WaitGroup
	for _, server := range servers {
		if err := server.Start(ctx, &serverWg); err != nil {
			archon.Log.Errorf("failed to start %s server: %v\n", server.Backend.Name(), err)
			os.Exit(1)
		}
	}

	// Register a SIGTERM handler so that Ctrl-C will shut the servers down gracefully.
	registerExitHandler(cancel, &serverWg)

	serverWg.Wait()
}

// Returns the database URI of the game database.
func dataSource() string {
	return fmt.Sprintf(
		databaseURITemplate,
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
		viper.GetString("database.name"),
		viper.GetString("database.username"),
		viper.GetString("database.password"),
		viper.GetString("database.sslmode"),
	)
}

func buildAddress(port interface{}) string {
	return fmt.Sprintf("%s:%v", viper.GetString("hostname"), port)
}

func registerExitHandler(cancelFn func(), wg ...*sync.WaitGroup) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		archon.Log.Infof("shutting down...")

		cancelFn()
		// TODO: add a timeout here.
		for _, wg := range wg {
			wg.Wait()
		}

		os.Exit(0)
	}()
}
