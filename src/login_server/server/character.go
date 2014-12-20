package server

import (
	"fmt"
	"net"
	"sync"
)

// Main worker thread for the CHARACTER portion of the server.
func openCharacterPort() {
	loginConfig := GetConfig()
	var socket *net.TCPListener = OpenSocket(loginConfig.Hostname(), loginConfig.CharacterPort())
	fmt.Printf("Waiting for CHARACTER connections on %s:%s...\n", loginConfig.Hostname(), loginConfig.CharacterPort())

	var connection *net.TCPConn
	var err error
	for {
		connection, err = socket.AcceptTCP()
		if err != nil {
			fmt.Println("Error accepting connection: " + err.Error())
			continue
		}
		fmt.Printf("Accepted CHARACTER connection from %s\n", connection.RemoteAddr())
	}

}

func StartCharacter(wg *sync.WaitGroup) {
	openCharacterPort()
	wg.Done()
}
