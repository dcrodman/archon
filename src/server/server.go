/*
* Archon PSO Server
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
	"server/configuration"
	"server/logging"
	"sync"
)

const (
	ServerConfigDir  = "/usr/local/share/archon"
	ServerConfigFile = "server_config.json"
	CertificateFile  = "certificate.pem"
	KeyFile          = "key.pem"
)

var (
	// Our global connection list.
	conns  *ConnList             = NewClientList()
	config *configuration.Config = configuration.GetConfig()
	log    *logging.ServerLogger
	host   string
	// Register all of the server handlers and their corresponding ports.
	servers = []Server{
		Server{"PATCH", config.PatchPort, NewPatchClient, PatchHandler},
		Server{"DATA", config.DataPort, NewPatchClient, DataHandler},
		Server{"LOGIN", config.LoginPort, NewLoginClient, LoginHandler},
		Server{"CHARACTER", config.CharacterPort, NewLoginClient, CharacterHandler},
		Server{"BLOCK", config.BlockPort, NewShipClient, BlockHandler},
		Server{"SHIP", config.ShipPort, NewShipClient, ShipHandler},
	}
)

type Server struct {
	name string
	port string
	// Allow each server to define their client structures.
	newClient func(conn *net.TCPConn) (ClientWrapper, error)
	handler   func(cw ClientWrapper)
}

func (s Server) Start(wg *sync.WaitGroup) {
	// Open our server socket. All sockets must be open for the server
	// to launch correctly, so errors are terminal.
	hostAddr, err := net.ResolveTCPAddr("tcp", config.Hostname+":"+s.port)
	if err != nil {
		fmt.Println("Error creating socket: " + err.Error())
		os.Exit(1)
	}
	socket, err := net.ListenTCP("tcp", hostAddr)
	if err != nil {
		fmt.Println("Error listening on socket: " + err.Error())
		os.Exit(1)
	}

	go func() {
		fmt.Printf("Waiting for %s connections on %v:%v\n", s.name, host, s.port)
		// Poll until we can accept more clients.
		for conns.Count() < config.MaxConnections {
			connection, err := socket.AcceptTCP()
			if err != nil {
				log.Warn("Failed to accept connection: %v", err.Error())
				continue
			}
			c, err := s.newClient(connection)
			if err != nil {
				log.Warn(err.Error())
			} else {
				log.Info("Accepted %s connection from %s", s.name, c.Client().IPAddr())
				s.dispatch(c)
			}
		}
		wg.Done()
	}()
}

func (s Server) dispatch(cw ClientWrapper) {
	c := cw.Client()
	go func() {
		// Defer so that we catch any panics, d/c the client, and
		// remove them from the list regardless of the connection state.
		defer func() {
			if err := recover(); err != nil {
				log.Error("Error in client communication: %s: %s\n%s\n",
					c.IPAddr(), err, debug.Stack())
			}
			c.Close()
			conns.Remove(c)
			log.Info("Disconnected %s client %s", s.name, c.IPAddr())
		}()
		conns.Add(c)
		// Pass along the connection handling to the registered server.
		s.handler(cw)
	}()
}

func main() {
	fmt.Println("Archon PSO Server, Copyright (C) 2014 Andrew Rodman\n" +
		"=====================================================\n" +
		"This program is free software: you can redistribute it and/or\n" +
		"modify it under the terms of the GNU General Public License as\n" +
		"published by the Free Software Foundation, either version 3 of\n" +
		"the License, or (at your option) any later version.\n" +
		"This program is distributed WITHOUT ANY WARRANTY; See LICENSE for details.\n")

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
	host = config.Hostname

	// Initialize the database.
	fmt.Printf("Connecting to MySQL database %s:%s...", config.DBHost, config.DBPort)
	err = config.InitDb()
	if err != nil {
		fmt.Println("Failed.\nPlease make sure the database connection parameters are correct.")
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("Done.\n")
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

	// Initialize the remaining servers and spin off our top-level
	// goroutines to listen on each port.
	InitPatch()
	fmt.Println()
	InitLogin()
	fmt.Println()

	var wg sync.WaitGroup
	for _, server := range servers {
		wg.Add(1)
		server.Start(&wg)
	}
	log.Important("Server Initialized")
	wg.Wait()
}
