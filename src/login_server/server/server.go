package server

/*
 * Starting point for the login server. Initializes the configuration package and takes care of
 * launching the LOGIN and CHARACTER servers. Also provides top-level functions and other code
 * shared between the two (found in login.go and character.go).
 */

import (
	"fmt"
	"net"
	"os"
	"sync"
)

// Create a TCP socket that is listening and ready to Accept().
func OpenSocket(host, port string) *net.TCPListener {
	hostAddress, err := net.ResolveTCPAddr("tcp", host+":"+port)
	if err != nil {
		fmt.Println("Error Creating Socket: " + err.Error())
		os.Exit(1)
	}
	socket, err := net.ListenTCP("tcp", hostAddress)
	if err != nil {
		fmt.Println("Error Listening on Socket: " + err.Error())
		os.Exit(1)
	}
	return socket
}

func Start() {
	GetConfig().InitFromMap(map[string]string{
		"hostname":      "127.0.0.1",
		"loginPort":     "12000",
		"characterPort": "12001",
	})

	// Create a WaitGroup so that main won't exit until the server threads have exited.
	var wg sync.WaitGroup
	wg.Add(2)
	go StartLogin(&wg)
	go StartCharacter(&wg)
	wg.Wait()
}
