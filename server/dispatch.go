package server

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/dcrodman/archon"
	archon_debug "github.com/dcrodman/archon/debug"
	"io"
	"net"
	"os"
	"runtime/debug"
	"time"
)

var hostname string

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

	for {
		// Poll until we can accept more clients.
		for isServerFull() {
			time.Sleep(time.Second)
		}

		conn, err := socket.AcceptTCP()
		if err != nil {
			archon.Log.Warnf("failed to accept connection: %s", err.Error())
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
			if globalClientList.has(cs) {
				archon.Log.Infof("rejected second %s connection from %s", server.Name(), cs.IPAddr())
				cs.connection.Close()
				return
			}

			c, err := server.AcceptClient(cs)

			if err == nil {
				archon.Log.Infof("accepted %s connection from %s", server.Name(), cs.IPAddr())
				globalClientList.add(cs)
				startClientLoop(s, c)
				globalClientList.remove(cs)
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
	defer closeConnectionAndRecover(s, c)

	for {
		packetSize, err := ReadNextPacket(c, s.HeaderSize())

		if err == io.EOF {
			break
		} else if err != nil {
			archon.Log.Warn(err.Error())
			break
		}

		if archon_debug.Enabled() {
			archon_debug.SendClientPacketToAnalyzer(c, c.ConnectionState().Data(), uint16(packetSize))
		}

		if err = s.Handle(c); err != nil {
			archon.Log.Warn("error in client communication: " + err.Error())
			return
		}
	}
}

func closeConnectionAndRecover(s Server, c Client2) {
	cs := c.ConnectionState()

	if err := recover(); err != nil {
		archon.Log.Errorf("error in client communication: %s: %s\n%s\n",
			cs.IPAddr(), err, debug.Stack())
	}

	if err := cs.connection.Close(); err != nil {
		archon.Log.Warnf("failed to close client connection: %s", err)
	}

	archon.Log.Infof("disconnected %s client %s", s.Name(), cs.IPAddr())
}

// ReadNextPacket is a blocking call that only returns once the client has
// sent the next packet to be processed. The buffer in c.ConnectionState is
// updated with the decrypted packet.
func ReadNextPacket(c Client2, headerSize uint16) (int, error) {
	recvSize, packetSize := 0, 0
	cs := c.ConnectionState()

	// Read the packet header.
	for recvSize < int(headerSize) {
		bytesRead, err := cs.connection.Read(cs.buffer[recvSize:headerSize])
		recvSize += bytesRead

		if bytesRead == 0 || err == io.EOF {
			return -1, err
		} else if err != nil {
			return -1, errors.New("socket Error (" + cs.IPAddr() + ") " + err.Error())
		}

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
			packetSize += packetSize % int(headerSize)
		}
	}

	// Grow the client's receive buffer if they send us a packet bigger than its current capacity.
	if packetSize > cap(cs.buffer) {
		newSize := packetSize + len(cs.buffer)
		newBuf := make([]byte, newSize)
		copy(newBuf, cs.buffer)
		cs.buffer = newBuf
	}

	// Read the rest of the packet.
	for recvSize < packetSize {
		remaining := packetSize - recvSize
		bytesRead, err := cs.connection.Read(cs.buffer[recvSize : recvSize+remaining])

		if err != nil {
			return -1, errors.New("socket Error (" + cs.IPAddr() + ") " + err.Error())
		}

		recvSize += bytesRead
	}

	// We have the whole thing; decrypt the rest of it.
	if packetSize > int(headerSize) {
		c.Decrypt(cs.buffer[headerSize:packetSize], uint32(packetSize-int(headerSize)))
	}

	return packetSize, nil
}

// Extract the packet length from the first two bytes of data.
func getPacketSize(data []byte) (int, error) {
	if len(data) < 2 {
		return 0, errors.New("getSize(): data must be at least two bytes")
	}

	var size uint16
	reader := bytes.NewReader(data)
	err := binary.Read(reader, binary.LittleEndian, &size)

	return int(size), err
}
