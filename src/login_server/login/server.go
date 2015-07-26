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
*
* Starting point for the login server. Initializes the configuration
* package and sets up the workers listening on the necessary ports.
* Also provides top-level functions and other code shared between
* the two (found in login.go and character.go).
 */
package login

import (
	"errors"
	"fmt"
	"io"
	"libarchon/logger"
	"libarchon/server"
	"libarchon/util"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"runtime/pprof"
	"sync"
)

var (
	log              *logger.ServerLogger
	loginConnections = server.NewClientList()
	shipConnections  = server.NewClientList()

	defaultShip   ShipEntry
	shipList      []ShipEntry
	shipListMutex sync.RWMutex
)

// Struct for holding client-specific data.
type LoginClient struct {
	c         *server.PSOClient
	guildcard uint32
	teamId    uint32
	isGm      bool

	gcData     []byte
	gcDataSize uint16
	config     ClientConfig
	flag       uint32
}

func (lc *LoginClient) IPAddr() string        { return lc.c.IPAddr() }
func (lc *LoginClient) Client() server.Client { return lc.c }
func (lc *LoginClient) Data() []byte          { return lc.c.Data() }

// Struct for representing available ships in the ship selection menu.
type ShipEntry struct {
	Unknown  uint16 // Always 0x12
	Id       uint32
	Padding  uint16
	Shipname [23]byte
}

type pktHandler func(p *LoginClient) error

// Return a JSON string to the client with the name, hostname, port,
// and player count.
func handleShipCountRequest(w http.ResponseWriter, req *http.Request) {
	if shipConnections.Count() == 0 {
		w.Write([]byte("[]"))
	} else {
		// TODO: Pull this from a cache
		w.Write([]byte("[]"))
	}
}

// Create and initialize a new struct to hold client information.
func newClient(conn *net.TCPConn) (*LoginClient, error) {
	loginClient := new(LoginClient)
	loginClient.c = server.NewPSOClient(conn, BBHeaderSize)

	var err error
	if SendWelcome(loginClient) != 0 {
		err = errors.New("Error sending welcome packet to: " + loginClient.IPAddr())
		loginClient = nil
	}
	return loginClient, err
}

// Handle communication with a particular client until the connection is
// closed or an error is encountered.
func handleClient(client *LoginClient, desc string, handler pktHandler) {
	defer func() {
		if err := recover(); err != nil {
			log.Error("Error in client communication: %s: %s\n%s\n",
				client.IPAddr(), err, debug.Stack())
		}
		client.c.Close()
		loginConnections.RemoveClient(client)
		log.Info("Disconnected %s client %s", desc, client.IPAddr())
	}()

	log.Info("Accepted %s connection from %s", desc, client.IPAddr())
	for {
		err := client.c.Process()
		if err == io.EOF {
			break
		} else if err != nil {
			// Error communicating with the client.
			log.Warn(err.Error())
			break
		}

		if err = handler(client); err != nil {
			log.Warn(err.Error())
			break
		}
	}
}

// Creates the socket and starts listening for connections on the specified
// port, spawning off goroutines to handle communications for each client.
func startWorker(wg *sync.WaitGroup, id, port string, handler pktHandler) {
	socket, err := server.OpenSocket(config.Hostname, port)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Printf("Waiting for %s connections on %s:%s...\n", id, config.Hostname, port)
	for {
		// Poll until we can accept more clients.
		for loginConnections.Count() < config.MaxConnections {
			connection, err := socket.AcceptTCP()
			if err != nil {
				log.Warn("Failed to accept connection: %s", err.Error())
				continue
			}
			client, err := newClient(connection)
			if err != nil {
				continue
			}
			if loginConnections.HasClient(client) {
				SendClientMessage(client, "Client is already connected to the server.")
				client.c.Close()
			} else {
				loginConnections.AddClient(client)
				go handleClient(client, id, handler)
			}
		}
	}
	wg.Done()
}

func StartServer() {
	// Initialize our config singleton from one of two expected file locations.
	fmt.Printf("Loading config file %v...", LoginConfigFile)
	err := config.InitFromFile(LoginConfigFile)
	if err != nil {
		os.Chdir(ServerConfigDir)
		fmt.Printf("Failed.\nLoading config from %v...", ServerConfigDir+"/"+LoginConfigFile)
		err = config.InitFromFile(LoginConfigFile)
		if err != nil {
			fmt.Println("Failed.\nPlease check that one of these files exists and restart the server.")
			fmt.Printf("%s\n", err.Error())
			os.Exit(-1)
		}
	}
	fmt.Printf("Done.\n\n--Configuration Parameters--\n%v\n\n", config.String())

	// Initialize the logger.
	log, err = logger.New(config.Logfile, config.LogLevel)
	if err != nil {
		fmt.Println("ERROR: Failed to open log file " + config.Logfile)
		os.Exit(1)
	}

	loadParameterFiles()
	loadBaseStats()

	// Create our "No Ships" item to indicate the absence of any ship servers.
	defaultShip.Unknown = 0x12
	defaultShip.Id = 1
	copy(defaultShip.Shipname[:], util.ConvertToUtf16("No Ships"))
	shipList = append(shipList, defaultShip)

	// Initialize the database.
	fmt.Printf("Connecting to MySQL database %s:%s...", config.DBHost, config.DBPort)
	err = config.InitDb()
	if err != nil {
		fmt.Println("Failed.\nPlease make sure the database connection parameters are correct.")
		fmt.Printf("Error: %s\n", err)
		os.Exit(-1)
	}
	fmt.Println("Done.")
	defer config.CloseDb()

	// Open up our web port for retrieving player counts. If we're in debug mode, add a path
	// for dumping pprof output containing the stack traces of all running goroutines.
	http.HandleFunc("/list", handleShipCountRequest)
	if config.DebugMode {
		http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
			pprof.Lookup("goroutine").WriteTo(resp, 1)
		})
	}
	go http.ListenAndServe(":"+config.WebPort, nil)

	log.Important("Server Initialized")
	// Create a WaitGroup so that main won't exit until the server threads have exited.
	var wg sync.WaitGroup
	wg.Add(3)
	go startShipgate(&wg)
	go startWorker(&wg, "LOGIN", config.LoginPort, processLoginPacket)
	go startWorker(&wg, "CHARACTER", config.CharacterPort, processCharacterPacket)
	wg.Wait()
}
