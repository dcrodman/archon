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
	"log"
	"net"
	"os"
	"server/client"
	"server/configuration"
	"server/logging"
	"sync"
)

var (
	// Our global connection list.
	conns *client.ConnList = client.NewList()
	// Global, threadsafe logger.
	log    *logging.ServerLogger
	config *configuration.Config
)

// Creates the socket and starts listening for connections on the specified
// port, spawning off goroutines for calls to the packet handler argument
// to handle communications for each client.
func worker(host, port string, handler func(c *client.Client)) {
	// Open our server socket.
	hostAddr, err := net.ResolveTCPAddr("tcp", host+":"+port)
	if err != nil {
		log.Fatalln("Error creating socket: " + err.Error())
	}
	socket, err := net.ListenTCP("tcp", hostAddr)
	if err != nil {
		log.Fatalln("Error Listening on Socket: " + err.Error())
	}

	for {
		// Poll until we can accept more clients.
		for conns.Count() < config.MaxConnections {
			connection, err := socket.AcceptTCP()
			if err != nil {
				log.Warn("Failed to accept connection: %v", err.Error())
				continue
			}
			c := client.NewPSOClient(conn, BBHeaderSize)
			if err != nil {
				log.Warn(err.Error())
				return
			}

			// Pass along the client handling to the registered server.
			go func() {
				// Defer so that we catch panics and remove the client from the
				// list regardless of the state of the connection.
				defer func() {
					if err := recover(); err != nil {
						log.Error("Error in client communication: %s: %s\n%s\n",
							c.IPAddr(), err, debug.Stack())
					}
					c.Close()
					conns.Remove(c)
					log.Info("Disconnected PATCH client %s", c.IPAddr())
				}()

				log.Info("Accepted PATCH connection from %s", c.IPAddr())
				conns.Add(c)
				handler(c)
			}()
		}
	}
}

func initialize() {
	// Initialize our config singleton from one of two expected file locations.
	fmt.Printf("Loading config file %v...", ServerConfigFile)
	config = config.GetConfig()
	err := config.InitFromFile(ServerConfigFile)
	if err != nil {
		os.Chdir(ServerConfigDir)
		fmt.Printf("Failed.\nLoading config from %v...", ServerConfigDir+"/"+ServerConfigFile)
		err = config.InitFromFile(ServerConfigFile)
		if err != nil {
			fmt.Println("Failed.\nPlease check that one of these files exists and restart the server.")
			log.Fatalln("Error: %s\n", err)
		}
	}
	fmt.Printf("Done.\n\n--Configuration Parameters--\n%v\n\n", config.String())

	// Initialize the database.
	fmt.Printf("Connecting to MySQL database %s:%s...", config.DBHost, config.DBPort)
	err = config.InitDb()
	if err != nil {
		fmt.Println("Failed.\nPlease make sure the database connection parameters are correct.")
		log.Fatalln("Error: %s\n", err)
	}
	fmt.Println("Done.")
	defer config.CloseDb()

	// If we're in debug mode, spawn off an HTTP server that, when hit, dumps
	// pprof output containing the stack traces of all running goroutines.
	if config.DebugMode {
		http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
			pprof.Lookup("goroutine").WriteTo(resp, 1)
		})
		go http.ListenAndServe(config.WebPort, nil)
	}

	// Initialize the logger.
	log, err = logger.New(config.Logfile, config.LogLevel)
	if err != nil {
		log.Fatalln("ERROR: Failed to open log file " + config.Logfile)
	}
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
	log.Important("Server Initialized")

	// Register all of the server handlers and their corresponding ports.
	var listeners = map[string]Handler{
		config.PatchPort: patch.PatchHandler,
	}

	// Spin off a goroutine for each top-level handler.
	var wg sync.WaitGroup
	for port, handler := range listeners {
		go func() {
			wg.Add(1)
			worker(config.Hostname, port, handler)
			wg.Done()
		}()
	}
	wg.Wait()
}
