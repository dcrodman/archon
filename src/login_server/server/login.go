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
	"errors"
	"fmt"
	"libarchon/util"
	"os"
	"sync"
)

const clientVersionString = "TethVer12510"

var connections *util.ConnectionList = util.NewClientList()

func handleLogin(client *LoginClient) error {
	loginPkt, err := VerifyAccount(client)
	if err != nil {
		LogMsg(err.Error(), LogTypeInfo, LogPriorityLow)
		return err
	}
	// TODO: Already connected to the server?
	/*
		Broken since this will always be true by this point (need to check for
		multiple occurrences)
		if connections.HasClient(client) {
			SendSecurity(client, BBLoginErrorUserInUse, 0)
			return errors.New("Client already connected to login server")
		}
	*/

	// Check the version string.
	if clientVersionString != string(util.StripPadding(loginPkt.Version[:])) {
		SendSecurity(client, BBLoginErrorPatch, 0)
		return errors.New("Incorrect version string")
	}

	SendSecurity(client, BBLoginErrorNone, 0)
	return nil
}

// Process packets sent to the LOGIN port by sending them off to another handler or by
// taking some brief action.
func processLoginPacket(client *LoginClient) error {
	var pktHeader BBPktHeader
	util.StructFromBytes(client.recvData, &pktHeader)

	fmt.Printf("\nGot %v bytes from client:\n", pktHeader.Size)
	util.PrintPayload(client.recvData, int(pktHeader.Size))

	var err error = nil
	switch pktHeader.Type {
	case LoginType:
		err = handleLogin(client)
		if err != nil {
			// err = SendRedirect(client)
		}
	case DisconnectType:
		// Just wait until we recv 0 from the client to d/c.
		break
	default:
		msg := fmt.Sprintf("Received unknown packet %x from %s", pktHeader.Type, client.ipAddr)
		LogMsg(msg, LogTypeInfo, LogPriorityMedium)
	}
	return err
}

// Handle communication with a particular client until the connection is closed or an
// error is encountered.
func handleLoginClient(client *LoginClient) {
	defer func() {
		if err := recover(); err != nil {
			errMsg := fmt.Sprintf("Error in client communication: %s: %s\n", client.ipAddr, err)
			LogMsg(errMsg, LogTypeError, LogPriorityHigh)
		}
		client.conn.Close()
		connections.RemoveClient(client)
	}()

	LogMsg("Accepted LOGIN connection from "+client.ipAddr, LogTypeInfo, LogPriorityHigh)
	// We're running inside a goroutine at this point, so we can block on this connection
	// and not interfere with any other clients.
	for {
		// Wait for the packet header.
		for client.recvSize < BBHeaderSize {
			bytes, err := client.conn.Read(client.recvData[client.recvSize:])
			if err != nil {
				// Socket error, nothing we can do now. TODO: log instead of panic().
				panic(err.Error())
			} else if bytes == 0 {
				// The client disconnected, we're done.
				client.conn.Close()
				break
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
		if err := processLoginPacket(client); err != nil {
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

// Main worker for the login server. Creates the socket and starts listening for connections,
// spawning off client threads to handle communications for each client.
func StartLogin(wg *sync.WaitGroup) {
	loginConfig := GetConfig()
	socket, err := util.OpenSocket(loginConfig.Hostname, loginConfig.LoginPort)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Printf("Waiting for LOGIN connections on %s:%s...\n", loginConfig.Hostname, loginConfig.LoginPort)
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
		connections.AddClient(client)
		go handleLoginClient(client)
	}
	wg.Done()
}
