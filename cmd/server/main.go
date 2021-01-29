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

	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/data"
	"github.com/dcrodman/archon/internal/debug"
	"github.com/dcrodman/archon/internal/server/character"
	"github.com/dcrodman/archon/internal/server/frontend"
	"github.com/dcrodman/archon/internal/server/login"
	"github.com/dcrodman/archon/internal/server/patch"
	"github.com/dcrodman/archon/internal/server/shipgate"
	"github.com/spf13/viper"
)

const databaseURITemplate = "host=%s port=%d dbname=%s user=%s password=%s sslmode=%s"

var config = flag.String("config", "config.yaml", "Path to the config file for the server")

func main() {
	fmt.Printf("Archon PSO Backend, Copyright (C) 2014 Andrew Rodman\n" +
		"=====================================================\n" +
		"This program is free software: you can redistribute it and/or\n" +
		"modify it under the terms of the GNU General Public License as\n" +
		"published by the Free Software Foundation, either version 3 of\n" +
		"the License, or (at your option) any later version. This program\n" +
		"is distributed WITHOUT ANY WARRANTY; See LICENSE for details.\n\n")

	flag.Parse()

	fmt.Println("loading configuration from", *config)
	archon.Load(*config)
	archon.InitLogger()

	// Change to the same directory as the config file so that any relative
	// paths in the config file will resolve.
	if err := os.Chdir(filepath.Dir(*config)); err != nil {
		fmt.Printf("failed to change to config directory: %v\n", err)
		os.Exit(1)
	}

	// Connect to the database.
	if err := data.Initialize(dataSource()); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer data.Shutdown()

	fmt.Printf("connected to database %s:%d\n\n",
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

	shipgateAddr := buildAddress(viper.GetString("shipgate_server.port"))

	servers := []*frontend.Frontend{
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
	}

	// Bind the server loops to one top-level server context so that we can shut down cleanly.
	ctx, cancel := context.WithCancel(context.Background())

	// Start the shipgate gRPC server and make sure it launches before the other servers start.
	readyChan := make(chan bool)
	errChan := make(chan error)
	go shipgate.Start(ctx, shipgateAddr, readyChan, errChan)
	go func() {
		if err := <-errChan; err != nil {
			fmt.Printf("exiting due to SHIPGATE error: %v", err)
			os.Exit(1)
		}
	}()
	<-readyChan

	// Start all of our servers. Failure to initialize one of the registered servers is considered terminal.
	var serverWg sync.WaitGroup
	for _, server := range servers {
		if err := server.Start(ctx, &serverWg); err != nil {
			fmt.Printf("failed to start %s server: %v\n", server.Backend.Name(), err)
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

func buildAddress(port string) string {
	return fmt.Sprintf("%s:%s", viper.GetString("hostname"), port)
}

func registerExitHandler(cancelFn func(), wg ...*sync.WaitGroup) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("shutting down...")

		cancelFn()
		// TODO: add a timeout here.
		for _, wg := range wg {
			wg.Wait()
		}

		os.Exit(0)
	}()
}
