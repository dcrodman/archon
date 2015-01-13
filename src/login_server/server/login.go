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
* LOGIN server logic.
 */

package server

import (
	"fmt"
	"libarchon/util"
	"os"
	"sync"
)

var connections *util.ConnectionList = util.NewClientList()

func handleLogin(client *LoginClient, pkt []byte) {

}

// Process packets sent to the LOGIN port by sending them off to another handler or by
// taking some brief action.
func processPacket(client *LoginClient, pkt []byte) error {
	var pktHeader BBPktHeader
	util.StructFromBytes(pkt, &pktHeader)

	switch pktHeader.Type {
	case LoginType:
		handleLogin(client, pkt)
	case DisconnectType:
		// Just wait until we recv 0 from the client to d/c.
		break
	default:
		fmt.Printf("Received unknown packet %x from %s", pktHeader.Type, client.ipAddr)
	}

	fmt.Printf("\nGot %v bytes from client:\n", pktHeader.Size)
	util.PrintPayload(pkt, int(pktHeader.Size))
	return nil
}

// Handle communication with a particular client until the connection is closed or an
// error is encountered.
func handleLoginClient(client *LoginClient) {
	defer func() {
		if err := recover(); err != nil {
			// TODO: Log to error file instead of stdout.
			fmt.Printf("Encountered communicating with client %s: %s\n", client.ipAddr, err)
		}
		connections.RemoveClient(client)
	}()

	fmt.Printf("Accepted LOGIN connection from %s\n", client.ipAddr)
	// We're running inside a goroutine at this point, so we can block on this connection
	// and not interfere with any other clients.
	var recvSize int
	var packetSize uint16
	recvData := make([]byte, 1024)
	for {
		// Wait for the packet header.
		for recvSize < BBHeaderSize {
			bytes, err := client.conn.Read(recvData[recvSize:])
			if err != nil {
				// Socket error, nothing we can do now. TODO: log instead of panic().
				panic(err.Error())
			} else if bytes == 0 {
				// The client disconnected, we're done.
				client.conn.Close()
				break
			}

			recvSize += bytes
			if recvSize >= BBHeaderSize {
				// We have our header; decrypt it.
				client.clientCrypt.Decrypt(recvData[:BBHeaderSize], BBHeaderSize)
				packetSize, err = util.GetPacketSize(recvData[:2])
				if err != nil {
					// Something is seriously wrong if this causes an error. Bail.
					panic(err.Error())
				}
			}
		}

		// Wait until we have the entire packet.
		for recvSize < int(packetSize) {
			bytes, err := client.conn.Read(recvData[recvSize:])
			if err != nil {
				panic(err.Error())
			}
			recvSize += bytes
		}

		// We have the whole thing; decrypt the rest of it and pass it along.
		client.clientCrypt.Decrypt(recvData[BBHeaderSize:recvSize], uint32(packetSize))
		if err := processPacket(client, recvData); err != nil {
			fmt.Println(err.Error())
			break
		}

		// Alternatively, we could set the slice to to nil here and make() a new one in order
		// to allow the garbage collector to handle cleanup, but I expect that would have a
		// noticable impact on performance. Instead, we're going to clear it manually.
		util.ZeroSlice(recvData, recvSize)
		recvSize = 0
		packetSize = 0
	}
}

// Main worker for the login server. Creates the socket and starts listening for connections,
// spawning off client threads to handle communications for each client.
func StartLogin(wg *sync.WaitGroup) {
	loginConfig := GetConfig()
	socket, err := util.OpenSocket(loginConfig.Hostname, loginConfig.LoginPort)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Printf("Waiting for LOGIN connections on %s:%s...\n",
		loginConfig.Hostname, loginConfig.LoginPort)

	for {
		connection, err := socket.AcceptTCP()
		if err != nil {
			fmt.Println("Error accepting connection: " + err.Error())
			continue
		}
		client, err := NewClient(connection)
		if err != nil {
			continue
		}
		connections.AddClient(client)
		go handleLoginClient(client)
	}
	wg.Done()
}
