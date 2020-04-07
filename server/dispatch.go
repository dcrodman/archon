package server

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/dcrodman/archon"
	"github.com/spf13/viper"
	"io"
	"net"
	"os"
	"runtime/debug"
	"sync"
)

var hostname string

// A concurrency-safe wrapper around container/list for maintaining a collection of connected clients.
type clientList struct {
	clients *list.List
	sync.RWMutex
}

func (c *clientList) add(cl *ConnectionState) {
	c.Lock()
	c.clients.PushBack(cl)
	c.Unlock()
}

func (c *clientList) remove(cl *ConnectionState) {
	clAddr := cl.IPAddr()
	c.RLock()
	for clientElem := c.clients.Front(); clientElem != nil; clientElem = clientElem.Next() {
		client := clientElem.Value.(*ConnectionState)
		if client.IPAddr() == clAddr {
			c.clients.Remove(clientElem)
			break
		}
	}
	c.RUnlock()
}

// Note: this comparison is by IP address, not element value.
func (c *clientList) has(cl *ConnectionState) bool {
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

func (c *clientList) len() int {
	c.RLock()
	defer c.RUnlock()
	return c.clients.Len()
}

// SetHostname sets the address to which any server instances launched with Start() will be bound.
func SetHostname(h string) {
	hostname = h
}

// Start opens a TCP socket for the specified server and enters a blocking loop for accepting
// client connections and dispatching them to the server.
func Start(s Server) {
	if hostname == "" {
		fmt.Println("error initializing server: no hostname set")
	}

	// Open our server socket. All sockets must be open for the server
	// to launch correctly, so errors are terminal.
	hostAddr, err := net.ResolveTCPAddr("tcp", hostname+":"+s.Port())
	if err != nil {
		fmt.Println("error creating socket: " + err.Error())
		os.Exit(1)
	}

	socket, err := net.ListenTCP("tcp", hostAddr)
	if err != nil {
		fmt.Println("error listening on socket: " + err.Error())
		os.Exit(1)
	}

	fmt.Printf("waiting for %s connections on %v:%v\n", s.Name(), hostname, s.Port())

	startListenerLoop(s, socket)
}

// startListenerLoop implements a connection handling loop that's purely responsible for
// accepting new connections and spinning off goroutines for the Server to handle them.
func startListenerLoop(server Server, socket *net.TCPListener) {
	defer fmt.Println(server.Name() + " exiting")

	cl := clientList{
		clients: list.New(),
		RWMutex: sync.RWMutex{},
	}
	maxConnections := viper.GetInt("max_connections")

	// Poll until we can accept more clients.
	for cl.len() < maxConnections {
		conn, err := socket.AcceptTCP()
		if err != nil {
			archon.Log.Warnf("failed to accept connection: %v", err.Error())
			continue
		}

		// Immediately spin off a goroutine to handle the client so that the main accept
		// loop doesn't get blocked by one client.
		//
		// Note: If there is eventually a need to implement worker pooling rather than spawning
		// new goroutines for each client, this is where it should be implemented.
		go func(s Server, conn *net.TCPConn) {
			cs := newConnectionState(conn)

			// Prevent multiple clients from connecting from the same IP address.
			if cl.has(cs) {
				cs.connection.Close()
				return
			}

			c, err := server.AcceptClient(cs)

			if err == nil {
				archon.Log.Infof("accepted %s connection from %s", server.Name(), cs.IPAddr())
				cl.add(cs)
				startClientLoop(s, c)
				cl.remove(cs)
			} else {
				archon.Log.Warn(err.Error())
			}
		}(server, conn)
	}
}

// startClientLoop starts a blocking loop dedicated to reading data sent from
// a game client and only returns once the connection has closed.
func startClientLoop(s Server, c Client2) {
	// Defer so that we catch any panics, disconnect the client, and
	// remove them from the list regardless of the connection state.
	defer closeClientConnection(s, c)

	for {
		err := ReadNextPacket(c, s.HeaderSize())

		if err == io.EOF {
			break
		} else if err != nil {
			archon.Log.Warn(err.Error())
			break
		}

		if err = s.Handle(c); err != nil {
			archon.Log.Warn("error in client communication: " + err.Error())
			return
		}
	}
}

// ReadNextPacket is a blocking call that only returns once the client has
// sent the next packet to be processed. The buffer in c.ConnectionState is
// updated with the decrypted packet.
func ReadNextPacket(c Client2, headerSize uint16) error {
	var recvSize int = 0
	var packetSize uint16 = 0
	cs := c.ConnectionState()

	// Wait for the packet header.
	for recvSize < int(headerSize) {
		bytes, err := cs.connection.Read(cs.buffer[recvSize:headerSize])

		if bytes == 0 || err == io.EOF {
			return err
		} else if err != nil {
			return errors.New("socket Error (" + cs.IPAddr() + ") " + err.Error())
		}
		recvSize += bytes

		if recvSize >= int(headerSize) {
			// At this point the full header has arrived and needs to be decrypted.
			c.Decrypt(cs.buffer[:headerSize], uint32(headerSize))

			packetSize, err = getPacketSize(cs.buffer[:2])
			if err != nil {
				// panic() since this should never occur unless something's _very_ wrong.
				panic(err.Error())
			}
			// The PSO client occasionally sends packets that are longer than their declared
			// size, but are always a multiple of the length of the packet header. Adjust the
			// expected length just in case in order to avoid leaving stray bytes in the buffer.
			packetSize += packetSize % headerSize
		}
	}
	pktSize := int(packetSize)

	// Grow the client's receive buffer if they send us a packet bigger than its current capacity.
	if pktSize > cap(cs.buffer) {
		newSize := pktSize + len(cs.buffer)
		newBuf := make([]byte, newSize)
		copy(newBuf, cs.buffer)
		cs.buffer = newBuf
	}

	// Read in the rest of the packet.
	for recvSize < pktSize {
		remaining := pktSize - recvSize
		bytes, err := cs.connection.Read(cs.buffer[recvSize : recvSize+remaining])
		if err != nil {
			return errors.New("socket Error (" + cs.IPAddr() + ") " + err.Error())
		}
		recvSize += bytes
	}

	// We have the whole thing; decrypt the rest of it.
	if packetSize > headerSize {
		c.Decrypt(cs.buffer[headerSize:packetSize], uint32(packetSize-headerSize))
	}

	return nil
}

// Extract the packet length from the first two bytes of data.
func getPacketSize(data []byte) (uint16, error) {
	if len(data) < 2 {
		return 0, errors.New("getSize(): data must be at least two bytes")
	}

	var size uint16
	reader := bytes.NewReader(data)
	err := binary.Read(reader, binary.LittleEndian, &size)

	return size, err
}

func closeClientConnection(s Server, c Client2) {
	cs := c.ConnectionState()
	if err := recover(); err != nil {
		archon.Log.Errorf("error in client communication: %s: %s\n%s\n",
			cs.IPAddr(), err, debug.Stack())
	}

	if err := cs.connection.Close(); err != nil {
		archon.Log.Warnf("failed to close client connection: ", err)
	}

	archon.Log.Infof("disconnected %s client %s", s.Name(), cs.IPAddr())
}
