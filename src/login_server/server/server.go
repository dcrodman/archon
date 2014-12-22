package server

/*
 * Starting point for the login server. Initializes the configuration package and takes care of
 * launching the LOGIN and CHARACTER servers. Also provides top-level functions and other code
 * shared between the two (found in login.go and character.go).
 */

import (
	"errors"
	"fmt"
	"libtethealla/encryption"
	"net"
	"os"
	"sync"
)

type Client struct {
	conn   *net.TCPConn
	ipAddr string

	clientCrypt *encryption.PSOCrypt
	serverCrypt *encryption.PSOCrypt
}

// Create and initialize a new struct to hold client information.
func NewClient(conn *net.TCPConn) (*Client, error) {
	client := new(Client)
	client.conn = conn
	client.ipAddr = conn.RemoteAddr().String()

	client.clientCrypt = encryption.NewCrypt()
	client.serverCrypt = encryption.NewCrypt()
	client.clientCrypt.CreateKeys()
	client.serverCrypt.CreateKeys()

	var err error = nil
	if SendWelcome(client) != 0 {
		err = errors.New("Error sending welcome packet to: " + client.ipAddr)
		client = nil
	}
	return client, err
}

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
