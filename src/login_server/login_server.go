package main

import (
	"fmt"
	"net"
	"os"
	"sync"
)

type serverConfig struct {
	Host string
	LoginPort string
	CharacterPort string
}

// Globals
var loginConfig *serverConfig

func load_config() *serverConfig {
	cfg := new(serverConfig)
	cfg.Host = "127.0.0.1"
	cfg.LoginPort = "12000"
	cfg.CharacterPort = "12001"
	return cfg
}

// Create a TCP socket that is listening and ready to Accept().
func openSocket(host, port string) (*net.TCPListener) {
	var hostAddress *net.TCPAddr
	var err error
	hostAddress, err = net.ResolveTCPAddr("tcp", host + ":" + port)
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
	var socket *net.TCPListener = openSocket(loginConfig.Host, loginConfig.LoginPort)
	var connection *net.TCPConn
	var err error
	fmt.Printf("Waiting for LOGIN connections on %s:%s...\n", loginConfig.Host, loginConfig.LoginPort)
	for {
		connection, err = socket.AcceptTCP()
		if err != nil{
			fmt.Println("Error accepting connection: " + err.Error())
			continue
		}
		fmt.Printf("Accepted connection from %s\n", connection.RemoteAddr())
	}
	wg.Done()
}

// Main worker thread for the CHARACTER portion of the server.
func openCharacterPort(wg *sync.WaitGroup) {
	var socket *net.TCPListener = openSocket(loginConfig.Host, loginConfig.LoginPort)
	var connection *net.TCPConn
	var err error
	fmt.Printf("Waiting for CHARACTER connections on %s:%s...\n", loginConfig.Host, loginConfig.LoginPort)
	for {
		connection, err = socket.AcceptTCP()
		if err != nil{
			fmt.Println("Error accepting connection: " + err.Error())
			continue
		}
		fmt.Printf("Accepted connection from %s\n", connection.RemoteAddr())
	}
	wg.Done()
}

// Main entry point.
func Start() {
	loginConfig = load_config()
	// Create a WaitGroup so that main won't exit until the server threads are done.
	var wg sync.WaitGroup
	wg.Add(2)
	openLoginPort(&wg)
	openCharacterPort(&wg)
	wg.Wait()
}
