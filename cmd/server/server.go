// Package server is the main entrypoint for running archon. It takes
// care of initializing everything as well as running as many servers are
// needed for a fully functional server backend.
package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/dcrodman/archon/internal"
	"github.com/dcrodman/archon/internal/block"
	"github.com/dcrodman/archon/internal/character"
	"github.com/dcrodman/archon/internal/core/data"
	"github.com/dcrodman/archon/internal/core/debug"
	"github.com/dcrodman/archon/internal/login"
	patch2 "github.com/dcrodman/archon/internal/patch"
	"github.com/dcrodman/archon/internal/ship"
	"github.com/dcrodman/archon/internal/shipgate"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"

	"github.com/dcrodman/archon"
)

const databaseURITemplate = "host=%s port=%d dbname=%s user=%s password=%s sslmode=%s"

func server(cc *cli.Context) error {
	config := cc.String("config")

	archon.LoadConfig(config)
	archon.InitLogger()

	archon.Log.Info("Archon PSO Backend, Copyright (C) 2014 Andrew Rodman\n" +
		"=====================================================\n" +
		"This program is free software: you can redistribute it and/or\n" +
		"modify it under the terms of the GNU General Public License as\n" +
		"published by the Free Software Foundation, either version 3 of\n" +
		"the License, or (at your option) any later version. This program\n" +
		"is distributed WITHOUT ANY WARRANTY; See LICENSE for details.")

	archon.Log.Infof("loaded configuration from %s", config)

	// Change to the same directory as the config file so that any relative
	// paths in the config file will resolve.
	if err := os.Chdir(filepath.Dir(config)); err != nil {
		err = errors.Wrap(err, "failed to change config directory")
		archon.Log.Error(err.Error())
		return err
	}

	// Connect to the database.
	if err := data.Initialize(dataSource(), debug.Enabled()); err != nil {
		archon.Log.Error(err.Error())
		return err
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
	var blockServers []*internal.Frontend
	for i := 1; i <= viper.GetInt("ship_server.num_blocks"); i++ {
		name := fmt.Sprintf("BLOCK%02d", i)
		address := buildAddress(viper.GetInt("block_server.port") + i)

		blocks = append(blocks, ship.Block{
			Name: name, Address: address, ID: i,
		})
		blockServer := &internal.Frontend{
			Address: address,
			Backend: block.NewServer(
				name,
				shipgateAddr,
				viper.GetInt("block_server.num_lobbies"),
			),
		}
		blockServers = append(blockServers, blockServer)
	}

	servers := []*internal.Frontend{
		{
			Address: buildAddress(patchPort),
			Backend: patch2.NewServer("PATCH", dataPort),
		},
		{
			Address: buildAddress(dataPort),
			Backend: patch2.NewDataServer("DATA"),
		},
		{
			Address: buildAddress(loginPort),
			Backend: login.NewServer("LOGIN", characterPort, shipgateAddr),
		},
		{
			Address: buildAddress(characterPort),
			Backend: character.NewServer("CHARACTER", shipgateAddr),
		},
		// TODO: Eventually the ship and block servers should be able to be run
		// independently of the other four servers
		{
			Address: buildAddress(shipPort),
			Backend: ship.NewServer("SHIP", blocks, shipgateAddr),
		},
	}
	servers = append(servers, blockServers...)

	// Bind the server loops to one top-level server context so that we can shut down cleanly.
	ctx, cancel := context.WithCancel(cc.Context)
	defer cancel()

	// Start the shipgate gRPC server and make sure it launches before the other servers start.
	readyChan := make(chan bool)
	errChan := make(chan error)
	go shipgate.Start(ctx, shipgateAddr, readyChan, errChan)
	go func() {
		if err := <-errChan; err != nil {
			archon.Log.Errorf("exiting due to SHIPGATE error: %v", err)
			cancel()
		}
	}()
	<-readyChan

	// Start all of our servers. Failure to initialize one of the registered servers is considered terminal.
	var serverWg sync.WaitGroup
	for _, server := range servers {
		if err := server.Start(ctx, &serverWg); err != nil {
			err = errors.Wrapf(err, "failed to start %s server\n", server.Backend.Name())
			archon.Log.Errorf(err.Error())
			return err
		}
	}

	// Register a SIGTERM handler so that Ctrl-C will shut the servers down gracefully.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go exitHandler(cancel, c, &serverWg)

	serverWg.Wait()

	return nil
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

func exitHandler(cancelFn func(), c chan os.Signal, wg ...*sync.WaitGroup) {
	<-c
	archon.Log.Infof("shutting down...")

	cancelFn()
	exitChan := make(chan bool)
	go func() {
		for _, wg := range wg {
			wg.Wait()
		}
		exitChan <- true
	}()

	select {
	case <-c:
		archon.Log.Info("hard exiting (killed)")
		os.Exit(0)
	case <-exitChan:
	}
}
