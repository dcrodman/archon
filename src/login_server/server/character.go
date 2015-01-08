package server

// CHARACTER server logic.

import (
	"fmt"
	"libarchon/util"
	"os"
	"sync"
)

// Main worker thread for the CHARACTER portion of the server.
func StartCharacter(wg *sync.WaitGroup) {
	loginConfig := GetConfig()
	socket, err := util.OpenSocket(loginConfig.GetHostname(), loginConfig.GetCharacterPort())
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Printf("Waiting for CHARACTER connections on %s:%s...\n\n", loginConfig.GetHostname(), loginConfig.GetCharacterPort())

	for {
		connection, err := socket.AcceptTCP()
		if err != nil {
			fmt.Println("Error accepting connection: " + err.Error())
			continue
		}
		fmt.Printf("Accepted CHARACTER connection from %s\n", connection.RemoteAddr())
	}
	wg.Done()
}
