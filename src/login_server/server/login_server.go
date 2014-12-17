package server

import (
	"fmt"
	"net"
	"os"
	"sync"
)

// Create a TCP socket that is listening and ready to Accept().
func openSocket(host, port string) *net.TCPListener {
	var hostAddress *net.TCPAddr
	var err error
	hostAddress, err = net.ResolveTCPAddr("tcp", host+":"+port)
	if err != nil {
		fmt.Println("Error Creating Socket: " + err.Error())
		os.Exit(1)
	}
	var socket *net.TCPListener
	socket, err = net.ListenTCP("tcp", hostAddress)
	if err != nil {
		fmt.Println("Error Listening on Socket: " + err.Error())
		os.Exit(1)
	}
	return socket
}

// Main worker thread for the LOGIN portion of the server.
func openLoginPort(wg *sync.WaitGroup) {
	loginConfig := GetConfig()
	var socket *net.TCPListener = openSocket(loginConfig.Hostname(), loginConfig.LoginPort())
	var connection *net.TCPConn
	var err error
	fmt.Printf("Waiting for LOGIN connections on %s:%s...\n", loginConfig.Hostname(), loginConfig.LoginPort())
	for {
		connection, err = socket.AcceptTCP()
		if err != nil {
			fmt.Println("Error accepting connection: " + err.Error())
			continue
		}
		fmt.Printf("Accepted connection from %s\n", connection.RemoteAddr())
	}
	wg.Done()
}

// Main worker thread for the CHARACTER portion of the server.
func openCharacterPort(wg *sync.WaitGroup) {
	loginConfig := GetConfig()
	var socket *net.TCPListener = openSocket(loginConfig.Hostname(), loginConfig.CharacterPort())
	var connection *net.TCPConn
	var err error
	fmt.Printf("Waiting for CHARACTER connections on %s:%s...\n", loginConfig.Hostname(), loginConfig.CharacterPort())
	for {
		connection, err = socket.AcceptTCP()
		if err != nil {
			fmt.Println("Error accepting connection: " + err.Error())
			continue
		}
		fmt.Printf("Accepted connection from %s\n", connection.RemoteAddr())
	}
	wg.Done()
}

func Start() {
	GetConfig().InitFromMap(map[string]string{
		"hostname":      "127.0.0.1",
		"loginPort":     "12000",
		"characterPort": "12001",
	})

	// Create a WaitGroup so that main won't exit until the server threads are done.
	var wg sync.WaitGroup
	wg.Add(2)
	go openLoginPort(&wg)
	go openCharacterPort(&wg)
	wg.Wait()
}
