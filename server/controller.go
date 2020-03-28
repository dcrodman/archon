package server

import (
	"container/list"
	"fmt"
	"github.com/dcrodman/archon"
	"io"
	"net"
	"os"
	"runtime/debug"
	"strconv"
	"sync"

	"github.com/dcrodman/archon/util"
)

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

// Controller is responsible for standing up the server instances we need and
// for dispatching handlers for each connected client.
type Controller struct {
	host        string
	servers     []Server
	connections *clientList
}

func New(hostname string) *Controller {
	return &Controller{
		host:        hostname,
		servers:     make([]Server, 0),
		connections: &clientList{clients: list.New()},
	}
}

// Registers a server instance to be brought up once the dispatcher is run.
func (controller *Controller) RegisterServer(s Server) {
	controller.servers = append(controller.servers, s)
}

// Iterate over our registered servers, initializing TCP sockets on each of the
// defined ports and setting up the connection handlers.
func (controller *Controller) Start() {
	var wg sync.WaitGroup

	for _, s := range controller.servers {
		if err := s.Init(); err != nil {
			fmt.Printf("Error initializing %s: %s\n", s.Name(), err.Error())
			return
		}

		// Open our server socket. All sockets must be open for the server
		// to launch correctly, so errors are terminal.
		hostAddr, err := net.ResolveTCPAddr("tcp", archon.Config.Hostname+":"+s.Port())
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
	archon.Log.Infof("Controller: Server Initialized")
	wg.Wait()
}

// Register all of the server handlers and their corresponding ports.
func registerServers(controller *server.controller) {
	servers := []Server{
		new(PatchServer),
		new(DataServer),
		new(LoginServer),
		new(CharacterServer),
		new(ShipgateServer),
		new(ShipServer),
	}
	for _, server := range servers {
		controller.registerServer(server)
	}

	shipPort, _ := strconv.ParseInt(Config.ShipServer.Port, 10, 16)

	// The available block ports will depend on how the server is configured,
	// so once we've read the config then add the server entries on the fly.
	for i := 1; i <= Config.ShipServer.NumBlocks; i++ {
		controller.registerServer(&BlockServer{
			name: fmt.Sprintf("BLOCK%d", i),
			port: strconv.FormatInt(shipPort+int64(i), 10),
		})
	}
}

// Client connection handling loop, started for each server.
func (controller *Controller) startHandler(server Server, socket *net.TCPListener) {
	defer fmt.Println(server.Name() + " shutdown.")

	// Poll until we can accept more clients.
	for controller.connections.Len() < archon.Config.MaxConnections {
		conn, err := socket.AcceptTCP()
		if err != nil {
			archon.Log.Warnf("Failed to accept connection: %v", err.Error())
			continue
		}
		c, err := server.NewClient(conn)
		// TODO: Disconnect the client if we already have a matching connection.
		if err != nil {
			archon.Log.Warn(err.Error())
		} else {
			archon.Log.Infof("Accepted %s connection from %s", server.Name(), c.IPAddr())
			controller.handleClient(c, server)
		}
	}
}

// Spawn a dedicated goroutine for each Client for the length of each connection.
func (controller *Controller) handleClient(c *Client, s Server) {
	go func() {
		// Defer so that we catch any panics, disconnect the client, and
		// remove them from the list regardless of the connection state.
		defer func() {
			if err := recover(); err != nil {
				archon.Log.Errorf("Error in client communication: %s: %s\n%s\n",
					c.IPAddr(), err, debug.Stack())
			}
			c.Close()
			controller.connections.Remove(c)
			archon.Log.Infof("Disconnected %s client %s", s.Name(), c.IPAddr())
		}()
		controller.connections.Add(c)

		// Connection loop; process packets until the connection is closed.
		var pktHeader archon.PCHeader
		for {
			err := c.Process()
			if err == io.EOF {
				break
			} else if err != nil {
				// Error communicating with the client.
				archon.Log.Warn(err.Error())
				break
			}

			// PC and BB header packets have the same structure for the first four
			// bytes, so for basic inspection it's safe to treat them the same way.
			util.StructFromBytes(c.Data()[:archon.PCHeaderSize], &pktHeader)
			if archon.Config.DebugMode {
				fmt.Printf("%s: Got %v bytes from client:\n", s.Name(), pktHeader.Size)
				util.PrintPayload(c.Data(), int(pktHeader.Size))
				fmt.Println()
			}

			if err = s.Handle(c); err != nil {
				archon.Log.Warn("Error in client communication: " + err.Error())
				return
			}
		}
	}()
}
