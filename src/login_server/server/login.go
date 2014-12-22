package server

// LOGIN server logic.

import (
	"container/list"
	"fmt"
	"net"
	"sync"
)

var loginClients *list.List = list.New()

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
		fmt.Printf("Accepted LOGIN connection from %s\n", client.ipAddr)
	}
}

func StartLogin(wg *sync.WaitGroup) {
	handleLoginConnections()
	wg.Done()
}
