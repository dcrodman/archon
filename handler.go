package main

import (
	"container/list"
	"fmt"
	"io"
	"net"
	"os"
	"runtime/debug"
	"sync"

	"github.com/dcrodman/archon/util"
)

// Server defines the methods implemented by all sub-servers that can be
// registered and started when the server is brought up.
type Server interface {
	// Uniquely identifying string, mostly used for logging.
	Name() string
	// Port on which the server should listen for connections.
	Port() string
	// Perform any pre-startup initialization.
	Init() error
	// Client factory responsible for performing whatever initialization is
	// needed for Client objects to represent new connections.
	NewClient(conn *net.TCPConn) (*Client, error)
	// Process the packet in the client's buffer. The dispatcher will
	// read the latest packet from the client before calling.
	Handle(c *Client) error
}

// Synchronized list for maintaining a list of connected clients.
type clientList struct {
	clients *list.List
	sync.RWMutex
}

func (c *clientList) Add(cl *Client) {
	c.Lock()
	c.clients.PushBack(cl)
	c.Unlock()
}

func (c *clientList) Remove(cl *Client) {
	clAddr := cl.IPAddr()
	c.RLock()
	for clientElem := c.clients.Front(); clientElem != nil; clientElem = clientElem.Next() {
		client := clientElem.Value.(*Client)
		if client.IPAddr() == clAddr {
			c.clients.Remove(clientElem)
			break
		}
	}
	c.RUnlock()
}

// Returns true if the list has a Client matching the IP address of c.
// Note that this comparison is by IP address, not element value.
func (c *clientList) Has(cl *Client) bool {
	clAddr := cl.IPAddr()
	c.RLock()
	defer c.RUnlock()
	for clientElem := c.clients.Front(); clientElem != nil; clientElem = clientElem.Next() {
		if cl.IPAddr() == clAddr {
			return true
		}
	}
	return false
}

func (c *clientList) Len() int {
	c.RLock()
	defer c.RUnlock()
	return c.clients.Len()
}

// controller is responsible for standing up the server instances we need and
// for dispatching handlers for each connected client.
type controller struct {
	host        string
	servers     []Server
	connections *clientList
}

// Registers a server instance to be brought up once the dispatcher is run.
func (controller *controller) registerServer(s Server) {
	controller.servers = append(controller.servers, s)
}

// Iterate over our registered servers, initializing TCP sockets on each of the
// defined ports and setting up the connection handlers.
func (controller *controller) start() *sync.WaitGroup {
	var wg sync.WaitGroup
	for _, s := range controller.servers {
		if err := s.Init(); err != nil {
			fmt.Printf("Error initializing %s: %s\n", s.Name(), err.Error())
			return nil
		}

		// Open our server socket. All sockets must be open for the server
		// to launch correctly, so errors are terminal.
		hostAddr, err := net.ResolveTCPAddr("tcp", config.Hostname+":"+s.Port())
		if err != nil {
			fmt.Println("Error creating socket: " + err.Error())
			os.Exit(1)
		}
		socket, err := net.ListenTCP("tcp", hostAddr)
		if err != nil {
			fmt.Println("Error listening on socket: " + err.Error())
			os.Exit(1)
		}

		wg.Add(1)
		go func(s Server, socket *net.TCPListener) {
			controller.startHandler(s, socket)
			wg.Done()
		}(s, socket)
	}

	for _, s := range controller.servers {
		fmt.Printf("Waiting for %s connections on %v:%v\n", s.Name(), controller.host, s.Port())
	}
	log.Infof("Controller: Server Initialized")
	return &wg
}

// Client connection handling loop, started for each server.
func (controller *controller) startHandler(server Server, socket *net.TCPListener) {
	defer fmt.Println(server.Name() + " shutdown.")

	// Poll until we can accept more clients.
	for controller.connections.Len() < config.MaxConnections {
		conn, err := socket.AcceptTCP()
		if err != nil {
			log.Warnf("Failed to accept connection: %v", err.Error())
			continue
		}
		c, err := server.NewClient(conn)
		// TODO: Disconnect the client if we already have a matching connection.
		if err != nil {
			log.Warn(err.Error())
		} else {
			log.Infof("Accepted %s connection from %s", server.Name(), c.IPAddr())
			controller.handleClient(c, server)
		}
	}
}

// Spawn a dedicated goroutine for each Client for the length of each connection.
func (controller *controller) handleClient(c *Client, s Server) {
	go func() {
		// Defer so that we catch any panics, disconnect the client, and
		// remove them from the list regardless of the connection state.
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("Error in client communication: %s: %s\n%s\n",
					c.IPAddr(), err, debug.Stack())
			}
			c.Close()
			controller.connections.Remove(c)
			log.Infof("Disconnected %s client %s", s.Name(), c.IPAddr())
		}()
		controller.connections.Add(c)

		// Connection loop; process packets until the connection is closed.
		var pktHeader PCHeader
		for {
			err := c.Process()
			if err == io.EOF {
				break
			} else if err != nil {
				// Error communicating with the client.
				log.Warn(err.Error())
				break
			}

			// PC and BB header packets have the same structure for the first four
			// bytes, so for basic inspection it's safe to treat them the same way.
			util.StructFromBytes(c.Data()[:PCHeaderSize], &pktHeader)
			if config.DebugMode {
				fmt.Printf("%s: Got %v bytes from client:\n", s.Name(), pktHeader.Size)
				util.PrintPayload(c.Data(), int(pktHeader.Size))
				fmt.Println()
			}

			if err = s.Handle(c); err != nil {
				log.Warn("Error in client communication: " + err.Error())
				return
			}
		}
	}()
}
