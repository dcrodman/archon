package server

// LOGIN server logic.

import (
	"container/list"
	"fmt"
	"libtethealla/util"
	"net"
	"sync"
)

var loginClients *list.List = list.New()

// Handle communication with a particular client until the connection is closed or an error
// is encountered.
func handleLoginClient(client *Client) {
	defer func() {
		if err := recover(); err != nil {
			// TODO: Log to error file instead of stdout.
			fmt.Printf("Encountered communicating with client %s: %s\n", client.ipAddr, err)
		}
	}()

	fmt.Printf("Accepted LOGIN connection from %s\n", client.ipAddr)
	for {
		// We're running inside a goroutine at this point, so we can block on this connection
		// and not interfere with any other clients.
		for client.recvSize < BBHeaderSize {
			bytes, err := client.conn.Read(client.recvData[client.recvSize:])
			if err != nil {
				// Socket error, nothing we can do now. TODO: log instead of panic().
				panic(err.Error())
			}
			client.recvSize += bytes

			if client.recvSize >= BBHeaderSize {
				// We have our header; decrypt it.
				client.clientCrypt.Decrypt(client.recvData[:BBHeaderSize], BBHeaderSize)
				client.packetSize, err = util.GetPacketSize(client.recvData[:2])
				if err != nil {
					// Something is seriously wrong if this causes an error. Bail.
					panic(err)
				}
			}
		}

		for client.recvSize < int(client.packetSize) {
			// Wait until we have the rest of the packet.
			bytes, err := client.conn.Read(client.recvData[client.recvSize:])
			if err != nil {
				panic(err.Error())
			}
			client.recvSize += bytes
		}

		// We have the whole thing; decrypt the rest of it.
		client.clientCrypt.Decrypt(client.recvData[BBHeaderSize:client.recvSize], uint32(client.packetSize))
		fmt.Printf("\nGot %v bytes from client:\n", client.recvSize)
		util.PrintPayload(client.recvData, client.recvSize)

		// TODO: Pass client and packet off to handler

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
func handleLoginConnections() {
	loginConfig := GetConfig()
	var socket *net.TCPListener = OpenSocket(loginConfig.Hostname(), loginConfig.LoginPort())
	fmt.Printf("Waiting for LOGIN connections on %s:%s...\n", loginConfig.Hostname(), loginConfig.LoginPort())

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
		loginClients.PushBack(connection)
		go handleLoginClient(client)
	}
}

func StartLogin(wg *sync.WaitGroup) {
	handleLoginConnections()
	wg.Done()
}
