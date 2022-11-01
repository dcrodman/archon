package internal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/client"
	archdebug "github.com/dcrodman/archon/internal/core/debug"
)

var connectedClients = make(map[string]*client.Client)

// frontend implements the concurrent client connection logic.
//
// Data is read from any connected clients and passed to a backend instance, abstracting
// the lower level connection details away from the Backends.
type frontend struct {
	Address string
	Backend Backend
	Config  *core.Config
	Logger  *logrus.Logger
}

// Start initializes the server backend and opens a TCP socket for the specified server.
// A blocking loop for accepting client connections is spun off in its own goroutine and
// added to the WaitGroup. Context cancellations will stop the server.
func (f *frontend) Start(ctx context.Context, wg *sync.WaitGroup) error {
	if err := f.Backend.Init(ctx); err != nil {
		return fmt.Errorf("error initializing %s server: %v", f.Backend.Identifier(), err)
	}

	socket, err := f.createSocket()
	if err != nil {
		return fmt.Errorf("error creating socket on %s: %v", f.Address, err)
	}

	wg.Add(1)
	go f.startBlockingLoop(ctx, socket, wg)

	return nil
}

// createSocket opens a TCP socket to listen for client connections on the Address
// provided to the frontend.
func (f *frontend) createSocket() (*net.TCPListener, error) {
	hostAddr, err := net.ResolveTCPAddr("tcp", f.Address)
	if err != nil {
		return nil, fmt.Errorf("error resolving address %s", err.Error())
	}

	socket, err := net.ListenTCP("tcp", hostAddr)
	if err != nil {
		return nil, fmt.Errorf("error listening on socket: %s", err.Error())
	}

	return socket, nil
}

// startBlockingLoop implements a connection handling loop that's purely responsible for
// accepting new connections and spinning off goroutines for the Backend to handle them.
func (f *frontend) startBlockingLoop(ctx context.Context, socket *net.TCPListener, wg *sync.WaitGroup) {
	defer wg.Done()

	f.Logger.Printf("[%s] waiting for connections on %v", f.Backend.Identifier(), f.Address)

	connections := make(chan *net.TCPConn)
	go func() {
		for {
			// Poll until we can accept more clients.
			for len(connectedClients) > f.Config.MaxConnections {
				time.Sleep(10 * time.Second)
			}

			connection, err := socket.AcceptTCP()
			if err != nil {
				f.Logger.Warnf("failed to accept connection: %s", err.Error())
				continue
			}

			connections <- connection
		}
	}()

	clientWg := &sync.WaitGroup{}
handleLoop:
	for {
		select {
		case <-ctx.Done():
			break handleLoop
		case connection := <-connections:
			clientWg.Add(1)
			// Note: If there is eventually a need to implement worker pooling rather than spawning
			// new goroutines for each client, this is where it should be implemented.
			go f.acceptClient(ctx, connection, clientWg)
		}
	}

	f.Logger.Infof("[%v] shutting down (waiting for connections to close)", f.Backend.Identifier())
	clientWg.Wait()
	f.Logger.Infof("[%v] exited", f.Backend.Identifier())
}

// acceptClient takes a connection and attempts to initiate a "session" by setting up
// the Client and sending the welcome packets. If it succeeds, the goroutine moves
// into the packet processing loop.
func (f *frontend) acceptClient(ctx context.Context, connection *net.TCPConn, wg *sync.WaitGroup) {
	defer wg.Done()

	c := client.NewClient(connection)
	f.Backend.SetUpClient(c)
	c.Debug = f.Config.Debugging.PacketLoggingEnabled

	f.Logger.Infof("[%s] accepted connection from %s", f.Backend.Identifier(), c.IPAddr())

	if err := f.Backend.Handshake(c); err != nil {
		f.Logger.Errorf("Handshake() failed for client %s: %s", c.IPAddr(), err)
	}

	// Prevent multiple clients from connecting from the same IP address.
	if _, ok := connectedClients[c.IPAddr()]; ok {
		f.Logger.Infof("[%s] rejected second connection from %s", f.Backend.Identifier(), c.IPAddr())
		_ = connection.Close()
		return
	}

	connectedClients[c.IPAddr()] = c
	f.processPackets(ctx, c)
}

// processPackets starts a blocking loop dedicated to reading data sent from
// a game client and only returns once the connection has closed.
func (f *frontend) processPackets(ctx context.Context, c *client.Client) {
	defer f.closeConnectionAndRecover(f.Backend.Identifier(), c)

	buffer := make([]byte, 2048)
	var err error

	for {
		select {
		case <-ctx.Done():
			// For now just allow the deferred function to close the connection.
			return
		default:
		}

		buffer, err = f.readNextPacket(c, buffer)

		if err == io.EOF {
			break
		} else if err != nil {
			f.Logger.Warn(err.Error())
			break
		}

		if f.Config.Debugging.PacketLoggingEnabled {
			archdebug.PrintPacket(archdebug.PrintPacketParams{
				Writer:       bufio.NewWriter(os.Stdout),
				ServerType:   archdebug.ServerType(c.DebugTags[archdebug.SERVER_TYPE]),
				ClientPacket: true,
				Data:         buffer,
			})
		}

		if err = f.Backend.Handle(ctx, c, buffer); err != nil {
			f.Logger.Warn("error in client communication: " + err.Error())
			return
		}
	}
}

// closeConnectionAndRecover is the failsafe that catches any panics, disconnects the
// client, and removes them from the list regardless of the state of the connection.
func (f *frontend) closeConnectionAndRecover(serverName string, c *client.Client) {
	if err := recover(); err != nil {
		f.Logger.Errorf("error in client communication with %s: error=%s, trace: %s",
			c.IPAddr(), err, debug.Stack())
	}

	if err := c.Close(); err != nil {
		f.Logger.Warnf("failed to close client connection: %s", err)
	}

	delete(connectedClients, c.IPAddr())

	f.Logger.Infof("[%s] disconnected client %s", serverName, c.IPAddr())
}

// readNextPacket is a blocking call that only returns once the client has
// sent the next packet to be processed. The buffer in c.ConnectionState is
// updated with the decrypted packet.
func (f *frontend) readNextPacket(c *client.Client, buffer []byte) ([]byte, error) {
	headerSize := int(c.CryptoSession.HeaderSize())

	// Read and decrypt the packet header.
	if err := f.readDataFromClient(c, headerSize, buffer); err != nil {
		return buffer, err
	}

	c.CryptoSession.Decrypt(buffer[:headerSize], uint32(headerSize))

	packetSize := f.determinePacketSize(buffer[:2], uint16(headerSize))

	// Grow the client's receive buffer if they send us a packet bigger than its current capacity.
	if packetSize > cap(buffer) {
		newBuf := make([]byte, cap(buffer)+packetSize)
		copy(newBuf, buffer)
		buffer = newBuf
	}

	// Read and decrypt the rest of the packet.
	if err := f.readDataFromClient(c, packetSize-headerSize, buffer[headerSize:]); err != nil {
		return buffer, err
	}

	c.CryptoSession.Decrypt(buffer[headerSize:packetSize], uint32(packetSize-headerSize))

	return buffer, nil
}

func (f *frontend) readDataFromClient(c *client.Client, n int, buffer []byte) error {
	received := 0

	for received < n {
		bytesRead, err := c.Read(buffer[received:n])
		received += bytesRead

		if bytesRead == 0 || err == io.EOF {
			return err
		} else if err != nil {
			return errors.New("socket error (" + c.IPAddr() + ") " + err.Error())
		}
	}

	return nil
}

// Extract the packet length from the first two bytes of data.
func (f *frontend) determinePacketSize(data []byte, headerSize uint16) int {
	if len(data) < 2 {
		// Panic since this shouldn't happen unless something's very wrong.
		panic(errors.New("getSize(): data must be at least two bytes"))
	}

	var size uint16
	reader := bytes.NewReader(data)
	err := binary.Read(reader, binary.LittleEndian, &size)

	if err != nil {
		f.Logger.Warn("error decoding packet size:", err)
	}

	// The PSO client occasionally sends packets that are longer than their declared
	// size, but are always a multiple of the length of the packet header. Adjust the
	// expected length just in case in order to avoid leaving stray bytes in the buffer.
	size += size % headerSize

	return int(size)
}
