package server

import (
	"fmt"
	"net"
	"sync"
)

// Main worker thread for the LOGIN portion of the server.
func openLoginPort() {
	loginConfig := GetConfig()
	var socket *net.TCPListener = OpenSocket(loginConfig.Hostname(), loginConfig.LoginPort())
	fmt.Printf("Waiting for LOGIN connections on %s:%s...\n", loginConfig.Hostname(), loginConfig.LoginPort())

	var connection *net.TCPConn
	var err error
	for {
		connection, err = socket.AcceptTCP()
		if err != nil {
			fmt.Println("Error accepting connection: " + err.Error())
			continue
		}
		fmt.Printf("Accepted LOGIN connection from %s\n", connection.RemoteAddr())
	}

}

func StartLogin(wg *sync.WaitGroup) {
	openLoginPort()
	wg.Done()
}
