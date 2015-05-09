/*
* Archon Patch Server
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
package patch_server

import (
	"errors"
	"fmt"
	"io"
	"libarchon/encryption"
	"libarchon/logger"
	"libarchon/util"
	"net"
	"os"
	"runtime/debug"
	"sync"
)

var log *logger.Logger

// Struct for holding client-specific data.
type PatchClient struct {
	conn   *net.TCPConn
	ipAddr string

	clientCrypt *encryption.PSOCrypt
	serverCrypt *encryption.PSOCrypt
	recvData    []byte
	recvSize    int
	packetSize  uint16
}

func (lc PatchClient) Connection() *net.TCPConn { return lc.conn }
func (lc PatchClient) IPAddr() string           { return lc.ipAddr }

var patchConnections *util.ConnectionList = util.NewClientList()

// Create and initialize a new struct to hold client information.
func newClient(conn *net.TCPConn) (*PatchClient, error) {
	client := new(PatchClient)
	client.conn = conn
	client.ipAddr = conn.RemoteAddr().String()

	client.clientCrypt = encryption.NewCrypt()
	client.serverCrypt = encryption.NewCrypt()
	client.clientCrypt.CreateKeys()
	client.serverCrypt.CreateKeys()
	client.recvData = make([]byte, 2048)

	var err error = nil
	if SendWelcome(client) != 0 {
		err = errors.New("Error sending welcome packet to: " + client.ipAddr)
		client = nil
	}
	return client, err
}

func processPatchPacket(client *PatchClient) error {
	var pktHeader BBPktHeader
	util.StructFromBytes(client.recvData[:BBHeaderSize], &pktHeader)

	if GetConfig().DebugMode {
		fmt.Printf("Got %v bytes from client:\n", pktHeader.Size)
		util.PrintPayload(client.recvData, int(pktHeader.Size))
		fmt.Println()
	}
	var err error = nil
	switch pktHeader.Type {
	case WelcomeType:
		SendWelcomeAck(client)
	case LoginType:
		// Send welcome message and redirect
		SendWelcomeMessage(client)
	default:
		msg := fmt.Sprintf("Received unknown packet %x from %s", pktHeader.Type, client.ipAddr)
		log.Info(msg, logger.LogPriorityMedium)
	}
	return err
}

// Handle communication with a particular client until the connection is closed or an
// error is encountered.
func handlePatchClient(client *PatchClient) {
	defer func() {
		if err := recover(); err != nil {
			errMsg := fmt.Sprintf("Error in client communication: %s: %s\n%s\n",
				client.ipAddr, err, debug.Stack())
			log.Error(errMsg, logger.LogPriorityCritical)
		}
		client.conn.Close()
		patchConnections.RemoveClient(client)
		log.Info("Disconnected PATCH client "+client.ipAddr, logger.LogPriorityMedium)
	}()

	log.Info("Accepted PATCH connection from "+client.ipAddr, logger.LogPriorityMedium)
	// We're running inside a goroutine at this point, so we can block on this connection
	// and not interfere with any other clients.
	for {
		// Wait for the packet header.
		for client.recvSize < BBHeaderSize {
			bytes, err := client.conn.Read(client.recvData[client.recvSize:])
			if bytes == 0 || err == io.EOF {
				// The client disconnected, we're done.
				client.conn.Close()
				return
			} else if err != nil {
				// Socket error, nothing we can do now
				log.Warn("Socket Error ("+client.ipAddr+") "+err.Error(),
					logger.LogPriorityMedium)
				return
			}

			client.recvSize += bytes
			if client.recvSize >= BBHeaderSize {
				// We have our header; decrypt it.
				client.clientCrypt.Decrypt(client.recvData[:BBHeaderSize], BBHeaderSize)
				client.packetSize, err = util.GetPacketSize(client.recvData[:2])
				if err != nil {
					// Something is seriously wrong if this causes an error. Bail.
					panic(err.Error())
				}
			}
		}

		// Wait until we have the entire packet.
		for client.recvSize < int(client.packetSize) {
			bytes, err := client.conn.Read(client.recvData[client.recvSize:])
			if err != nil {
				log.Warn("Socket Error ("+client.ipAddr+") "+err.Error(),
					logger.LogPriorityMedium)
				return
			}
			client.recvSize += bytes
		}

		// We have the whole thing; decrypt the rest of it if needed and pass it along.
		if client.packetSize > BBHeaderSize {
			client.clientCrypt.Decrypt(
				client.recvData[BBHeaderSize:client.packetSize],
				uint32(client.packetSize-BBHeaderSize))
		}
		if err := processPatchPacket(client); err != nil {
			log.Info(err.Error(), logger.LogPriorityLow)
			break
		}

		// Alternatively, we could set the slice to to nil here and make() a new one in order
		// to allow the garbage collector to handle cleanup, but I expect that would have a
		// noticable impact on performance. Instead, we're going to clear it manually.
		util.ZeroSlice(client.recvData, client.recvSize)
		client.recvSize = 0
		client.packetSize = 0
	}
}

// Main worker for the patch server. Creates the socket and starts listening for connections,
// spawning off client threads to handle communications for each client.
func startPatch(wg *sync.WaitGroup) {
	patchConfig := GetConfig()
	socket, err := util.OpenSocket(patchConfig.Hostname, patchConfig.PatchPort)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Printf("Waiting for PATCH connections on %s:%s...\n",
		patchConfig.Hostname, patchConfig.PatchPort)
	for {
		connection, err := socket.AcceptTCP()
		if err != nil {
			log.Error("Failed to accept connection: "+err.Error(), logger.LogPriorityHigh)
			continue
		}
		client, err := newClient(connection)
		if err != nil {
			continue
		}
		patchConnections.AddClient(client)
		go handlePatchClient(client)
	}
	wg.Done()
}

// Main worker for the data server. Creates the socket and starts listening for connections,
// spawning off client threads to handle communications for each client.
func startData(wg *sync.WaitGroup) {
	patchConfig := GetConfig()
	socket, err := util.OpenSocket(patchConfig.Hostname, patchConfig.DataPort)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Printf("Waiting for DATA connections on %s:%s...\n",
		patchConfig.Hostname, patchConfig.DataPort)
	for {
		connection, err := socket.AcceptTCP()
		if err != nil {
			log.Error("Failed to accept connection: "+err.Error(), logger.LogPriorityHigh)
			continue
		}
		client, err := newClient(connection)
		if err != nil {
			continue
		}
		patchConnections.AddClient(client)
		go handlePatchClient(client)
	}
	wg.Done()
}

func StartServer() {
	fmt.Println("Initializing Archon PATCH and DATA servers...")
	config := GetConfig()
	// Initialize our config singleton from one of two expected file locations.
	fmt.Printf("Loading config file %v...", patchConfigFile)
	err := config.InitFromFile(patchConfigFile)
	if err != nil {
		os.Chdir(ServerConfigDir)
		fmt.Printf("Failed.\nLoading config from %v...", ServerConfigDir+"/"+patchConfigFile)
		err = config.InitFromFile(patchConfigFile)
		if err != nil {
			fmt.Println("Failed.\nPlease check that one of these files exists and restart the server.")
			fmt.Printf("%s\n", err.Error())
			os.Exit(-1)
		}
	}
	fmt.Printf("Done.\n\n--Configuration Parameters--\n%v\n\n", config.String())

	// Initialize the logger.
	log = logger.New(config.logWriter, config.LogLevel)
	log.Info("Server Initialized", logger.LogPriorityCritical)

	// Create a WaitGroup so that main won't exit until the server threads have exited.
	var wg sync.WaitGroup
	wg.Add(2)
	go startPatch(&wg)
	go startData(&wg)
	wg.Wait()
}
