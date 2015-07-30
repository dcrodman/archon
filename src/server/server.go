/*
* Archon Login Server
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
* ---------------------------------------------------------------------
 */
package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"runtime/pprof"
	"server/client"
	"server/configuration"
	"server/logging"
	"sync"
)

var (
	// Our global connection list.
	conns  *client.ConnList = client.NewList()
	log    *logging.ServerLogger
	config *configuration.Config = configuration.GetConfig()
)

const (
	ServerConfigDir  = "/usr/local/share/archon"
	ServerConfigFile = "server_config.json"
	CertificateFile  = "certificate.pem"
	KeyFile          = "key.pem"
)

type handler func(c client.Client)

func dispatch(desc string, c *client.PSOClient, connHandler handler) {
	// Defer so that we catch any panics, d/c the client, and
	// remove them from the list regardless of the connection state.
	defer func() {
		if err := recover(); err != nil {
			log.Error("Error in client communication: %s: %s\n%s\n",
				c.IPAddr(), err, debug.Stack())
		}
		c.Close()
		conns.Remove(c)
		log.Info("Disconnected %s client %s", desc, c.IPAddr())
	}()
	conns.Add(c)
	// Pass along the connection handling to the registered server.
	connHandler(c)
}

// Creates the socket and starts listening for connections on the specified
// port, spawning off goroutines for calls to the packet handler argument
// to handle communications for each client. There will be one worker
// routine created for each server.
func worker(host, port, desc string, connHandler handler, wg *sync.WaitGroup) {
	// Open our server socket.
	hostAddr, err := net.ResolveTCPAddr("tcp", host+":"+port)
	if err != nil {
		fmt.Println("Error creating socket: " + err.Error())
		os.Exit(1)
	}
	socket, err := net.ListenTCP("tcp", hostAddr)
	if err != nil {
		fmt.Println("Error Listening on Socket: " + err.Error())
		os.Exit(1)
	}

	fmt.Printf("Waiting for %s connections on %v:%v\n", desc, host, port)
	for {
		// Poll until we can accept more clients.
		for conns.Count() < config.MaxConnections {
			connection, err := socket.AcceptTCP()
			if err != nil {
				log.Warn("Failed to accept connection: %v", err.Error())
				continue
			}
			c := client.NewPSOClient(connection, BBHeaderSize)
			log.Info("Accepted %s connection from %s", desc, c.IPAddr())
			go dispatch(desc, c, connHandler)
		}
	}
	wg.Done()
}

func initialize() {
	// Initialize our config singleton from one of two expected file locations.
	fmt.Printf("Loading config file %v...", ServerConfigFile)
	err := config.InitFromFile(ServerConfigFile)
	if err != nil {
		os.Chdir(ServerConfigDir)
		fmt.Printf("Failed.\nLoading config from %v...", ServerConfigDir+"/"+ServerConfigFile)
		err = config.InitFromFile(ServerConfigFile)
		if err != nil {
			fmt.Println("Failed.\nPlease check that one of these files exists and restart the server.")
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
	}
	fmt.Printf("Done.\n\n--Configuration Parameters--\n%v\n\n", config.String())

	// Initialize the database.
	fmt.Printf("Connecting to MySQL database %s:%s...", config.DBHost, config.DBPort)
	err = config.InitDb()
	if err != nil {
		fmt.Println("Failed.\nPlease make sure the database connection parameters are correct.")
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("Done.")
	defer config.CloseDB()

	// If we're in debug mode, spawn off an HTTP server that, when hit, dumps
	// pprof output containing the stack traces of all running goroutines.
	if config.DebugMode {
		http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
			pprof.Lookup("goroutine").WriteTo(resp, 1)
		})
		go http.ListenAndServe(config.WebPort, nil)
	}

	// Initialize the logger.
	log, err = logging.New(config.Logfile, config.LogLevel)
	if err != nil {
		fmt.Println("ERROR: Failed to open log file " + config.Logfile)
		os.Exit(1)
	}
	log.Important("Server Initialized")
}

func main() {
	fmt.Println("Archon Login Server, Copyright (C) 2014 Andrew Rodman\n" +
		"=====================================================\n" +
		"This program is free software: you can redistribute it and/or\n" +
		"modify it under the terms of the GNU General Public License as\n" +
		"published by the Free Software Foundation, either version 3 of\n" +
		"the License, or (at your option) any later version.\n" +
		"This program is distributed WITHOUT ANY WARRANTY; See LICENSE for details.\n")

	initialize()
	InitPatch()
	InitLogin()

	type Server struct {
		name       string
		port       string
		pktHandler handler
	}
	// Register all of the server handlers and their corresponding ports.
	listeners := []Server{
		Server{"PATCH", config.PatchPort, PatchHandler},
		Server{"DATA", config.DataPort, DataHandler},
	}

	// Spin off a goroutine for each top-level handler.
	var wg sync.WaitGroup
	for _, entry := range listeners {
		wg.Add(1)
		go worker(config.Hostname, entry.port, entry.name, entry.pktHandler, &wg)
	}
	wg.Wait()
}
