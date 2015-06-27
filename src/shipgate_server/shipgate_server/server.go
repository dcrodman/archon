/*
* Archon Shipgate Server
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

package shipgate_server

import (
	"fmt"
	"libarchon/logger"
	"libarchon/server"
	// "libarchon/util"
	// "encoding/json"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"sync"
)

var log *logger.Logger
var shipgateConnections *server.ConnectionList = server.NewClientList()

// Struct for holding client-specific data.
type ShipgateClient struct {
	conn   *net.TCPConn
	ipAddr string
	port   string

	recvData   []byte
	recvSize   int
	packetSize uint16
}

func (lc ShipgateClient) Connection() *net.TCPConn { return lc.conn }
func (lc ShipgateClient) IPAddr() string           { return lc.ipAddr }

// Create and initialize a new struct to hold client information.
func newClient(conn *net.TCPConn) (*ShipgateClient, error) {
	client := new(ShipgateClient)
	client.conn = conn
	addr := strings.Split(conn.RemoteAddr().String(), ":")
	client.ipAddr = addr[0]
	client.port = addr[1]

	var err error = nil
	// if SendWelcome(client) != 0 {
	// 	err = errors.New("Error sending welcome packet to: " + client.ipAddr)
	// 	client = nil
	// }
	return client, err
}

func handleShipCountRequest(w http.ResponseWriter, req *http.Request) {
	// TODO: Grab this from a cache instead computing it on the fly.
	if shipgateConnections.Count() == 0 {
		w.Write([]byte("{}"))
	}
}

// Handle communication with a particular client until the connection is
// closed or an error is encountered.
func handleClient(client *ShipgateClient) {
	defer func() {
		if err := recover(); err != nil {
			errMsg := fmt.Sprintf("Error in client communication: %s: %s\n%s\n",
				client.ipAddr, err, debug.Stack())
			log.Error(errMsg, logger.CriticalPriority)
		}
		client.conn.Close()
		shipgateConnections.RemoveClient(client)
		log.Info("Disconnected ship "+client.ipAddr, logger.MediumPriority)
	}()

	// log.Info("Accepted "+desc+" connection from "+client.ipAddr, logger.MediumPriority)
	// // We're running inside a goroutine at this point, so we can block on this connection
	// // and not interfere with any other clients.
	// for {
	// 	// Wait for the packet header.
	// 	for client.recvSize < BBHeaderSize {
	// 		bytes, err := client.conn.Read(client.recvData[client.recvSize:BBHeaderSize])
	// 		if bytes == 0 || err == io.EOF {
	// 			// The client disconnected, we're done.
	// 			client.conn.Close()
	// 			return
	// 		} else if err != nil {
	// 			// Socket error, nothing we can do now
	// 			log.Warn("Socket Error ("+client.ipAddr+") "+err.Error(),
	// 				logger.MediumPriority)
	// 			return
	// 		}
	// 		client.recvSize += bytes

	// 		if client.recvSize >= BBHeaderSize {
	// 			// We have our header; decrypt it.
	// 			// client.clientCrypt.Decrypt(client.recvData[:BBHeaderSize], BBHeaderSize)
	// 			client.packetSize, err = util.GetPacketSize(client.recvData[:2])
	// 			if err != nil {
	// 				// Something is seriously wrong if this causes an error. Bail.
	// 				panic(err.Error())
	// 			}
	// 			// PSO likes to occasionally send us packets that are longer than their
	// 			// declared size. Adjust the expected length just in case in order to
	// 			// avoid leaving stray bytes in the buffer.
	// 			for client.packetSize%BBHeaderSize != 0 {
	// 				client.packetSize++
	// 			}
	// 		}
	// 	}
	// 	pktSize := int(client.packetSize)
	// 	// Grow the client's receive buffer if they send us a packet bigger
	// 	// than its current capacity.
	// 	if pktSize > cap(client.recvData) {
	// 		newSize := pktSize + len(client.recvData)
	// 		newBuf := make([]byte, newSize)
	// 		copy(newBuf, client.recvData)
	// 		client.recvData = newBuf
	// 		msg := fmt.Sprintf("Reallocated buffer to %v bytes", newSize)
	// 		log.Info(msg, logger.LowPriority)
	// 	}

	// 	// Read in the rest of the packet.
	// 	for client.recvSize < pktSize {
	// 		remaining := pktSize - client.recvSize
	// 		bytes, err := client.conn.Read(
	// 			client.recvData[client.recvSize : client.recvSize+remaining])
	// 		if err != nil {
	// 			log.Warn("Socket Error ("+client.ipAddr+") "+err.Error(),
	// 				logger.MediumPriority)
	// 			return
	// 		}
	// 		client.recvSize += bytes
	// 	}

	// 	// We have the whole thing; decrypt the rest of it if needed and pass it along.
	// 	if client.packetSize > BBHeaderSize {
	// 		client.clientCrypt.Decrypt(
	// 			client.recvData[BBHeaderSize:client.packetSize],
	// 			uint32(client.packetSize-BBHeaderSize))
	// 	}
	// 	if err := processShipgatePacket(client); err != nil {
	// 		log.Info(err.Error(), logger.LowPriority)
	// 		break
	// 	}

	// 	// Extra bytes left in the buffer will just be ignored.
	// 	client.recvSize = 0
	// 	client.packetSize = 0
	// }
}

// Creates the socket and starts listening for connections on the specified
// port, spawning off goroutines to handle communications for each client.
func startWorker(wg *sync.WaitGroup, id, port string) {
	cfg := GetConfig()
	socket, err := server.OpenSocket(cfg.Hostname, port)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Printf("Waiting for %s connections on %s:%s...\n", id, cfg.Hostname, port)
	for {
		// Poll until we can accept more clients.
		for shipgateConnections.Count() < cfg.MaxConnections {
			connection, err := socket.AcceptTCP()
			if err != nil {
				log.Error("Failed to accept connection: "+err.Error(), logger.HighPriority)
				continue
			}
			client, err := newClient(connection)
			if err != nil {
				continue
			}
			go handleClient(client)
		}
	}
	wg.Done()
}

func StartServer() {
	fmt.Println("Initializing Archon Shipgate server...")
	config := GetConfig()

	// Initialize our config singleton from one of two expected file locations.
	fmt.Printf("Loading config file %v...", loginConfigFile)
	err := config.InitFromFile(loginConfigFile)
	if err != nil {
		os.Chdir(ServerConfigDir)
		fmt.Printf("Failed.\nLoading config from %v...", ServerConfigDir+"/"+loginConfigFile)
		err = config.InitFromFile(loginConfigFile)
		if err != nil {
			fmt.Println("Failed.\nPlease check that one of these files exists and restart the server.")
			fmt.Printf("%s\n", err.Error())
			os.Exit(-1)
		}
	}
	fmt.Printf("Done.\n\n--Configuration Parameters--\n%v\n\n", config.String())

	// Initialize the logger.
	log = logger.New(config.logWriter, config.LogLevel)
	log.Info("Server Initialized", logger.CriticalPriority)

	// Start our debugging server if needed.
	if config.DebugMode {
		go server.CreateStackTraceServer("127.0.0.1:8082", "/")
	}

	// Open up our web port for retrieving player counts and the like.
	http.HandleFunc("/list", handleShipCountRequest)
	if http.ListenAndServe(":"+config.WebPort, nil) != nil {
		fmt.Printf("Failed to open web port on " + config.WebPort)
		os.Exit(-1)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go startWorker(&wg, "SHIP", config.ShipgatePort)
	wg.Wait()
}
