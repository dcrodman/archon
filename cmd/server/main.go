// The server command is the main entrypoint for running archon. It takes
// care of initializing everything as well as running as many servers are
// needed for a fully functional server backend.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/dcrodman/archon/internal"
	"github.com/dcrodman/archon/internal/core"
)

var configFlag = flag.String("config", "./", "Path to the directory containing the server config file")

func main() {
	flag.Parse()

	fmt.Println("Archon PSO Backend, Copyright (C) 2014 Andrew Rodman\n" +
		"=====================================================\n" +
		"This program is free software: you can redistribute it and/or\n" +
		"modify it under the terms of the GNU General Public License as\n" +
		"published by the Free Software Foundation, either version 3 of\n" +
		"the License, or (at your option) any later version. This program\n" +
		"is distributed WITHOUT ANY WARRANTY; See LICENSE for details.")

	config := core.LoadConfig(*configFlag)
	fmt.Println("using configuration file:", *configFlag)

	// Change to the same directory as the config file so that any relative
	// paths in the config file will resolve.
	if err := os.Chdir(filepath.Dir(*configFlag)); err != nil {
		fmt.Println("error changing to config directory:", err)
		os.Exit(1)
	}

	// Bind the Controller to one top-level server context so that we can shut down cleanly.
	ctx, cancel := context.WithCancel(context.Background())

	// Register a SIGTERM handler so that Ctrl-C will shut the servers down gracefully.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go exitHandler(cancel, c)

	// Start up the controller to handle all of the resources and server init.
	controller := &internal.Controller{
		Config: config,
	}
	if err := controller.Start(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			fmt.Println(err)
		}
	}
	fmt.Println("shut down")
}

func exitHandler(cancelFn func(), c chan os.Signal, wg ...*sync.WaitGroup) {
	<-c
	fmt.Println("waiting to shut down gracefully...")

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
		fmt.Println("hard exiting (killed)")
	case <-exitChan:
	}

	os.Exit(0)
}
