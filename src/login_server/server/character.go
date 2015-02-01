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
* CHARACTER server logic.
 */
package server

import (
	"fmt"
	"io"
	"libarchon/util"
	"os"
	"sync"
)

var charConnections *util.ConnectionList = util.NewClientList()

func handleCharLogin(client *LoginClient) error {
	return nil
}

func processCharacterPacket(client *LoginClient) error {
	var pktHeader BBPktHeader
	util.StructFromBytes(client.recvData, &pktHeader)

	if GetConfig().DebugMode {
		fmt.Printf("Got %v bytes from client:\n", pktHeader.Size)
		util.PrintPayload(client.recvData, int(pktHeader.Size))
		fmt.Println()
	}

	var err error = nil
	switch pktHeader.Type {
	case LoginType:
		err = handleCharLogin(client)
	case DisconnectType:
		// Just wait until we recv 0 from the client to d/c.
		break
	default:
		msg := fmt.Sprintf("Received unknown packet %x from %s", pktHeader.Type, client.ipAddr)
		LogMsg(msg, LogTypeInfo, LogPriorityMedium)
	}
	return err
}

func handleCharacterClient(client *LoginClient) {
	defer func() {
		if err := recover(); err != nil {
			errMsg := fmt.Sprintf("Error in client communication: %s: %s\n", client.ipAddr, err)
			LogMsg(errMsg, LogTypeError, LogPriorityHigh)
		}
		client.conn.Close()
		charConnections.RemoveClient(client)
		LogMsg("Disconnected CHARACTER client "+client.ipAddr, LogTypeInfo, LogPriorityMedium)
	}()

	LogMsg("Accepted CHARACTER connection from "+client.ipAddr, LogTypeInfo, LogPriorityMedium)
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
				LogMsg("Socket Error ("+client.ipAddr+") "+err.Error(),
					LogTypeWarning, LogPriorityMedium)
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
				panic(err.Error())
			}
			client.recvSize += bytes
		}

		// We have the whole thing; decrypt the rest of it and pass it along.
		client.clientCrypt.Decrypt(client.recvData[BBHeaderSize:client.recvSize], uint32(client.packetSize))
		if err := processCharacterPacket(client); err != nil {
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

// Main worker thread for the CHARACTER portion of the server.
func StartCharacter(wg *sync.WaitGroup) {
	loginConfig := GetConfig()
	socket, err := util.OpenSocket(loginConfig.Hostname, loginConfig.CharacterPort)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Printf("Waiting for CHARACTER connections on %s:%s...\n\n",
		loginConfig.Hostname, loginConfig.CharacterPort)

	for {
		connection, err := socket.AcceptTCP()
		if err != nil {
			LogMsg("Failed to accept connection: "+err.Error(), LogTypeError, LogPriorityHigh)
			continue
		}
		client, err := NewClient(connection)
		if err != nil {
			continue
		}
		charConnections.AddClient(client)
		go handleCharacterClient(client)
	}
	wg.Done()
}
