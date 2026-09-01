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
	rdbg "runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dcrodman/archon/internal/debug"
	"github.com/dcrodman/archon/internal/shipgate"
)

// Start initializes all of Archon's servers and blocks, waiting until all of the servers
// have been gracefully shut down before exiting.
func Start(ctx context.Context) {
	if Config.FilePath == "" {
		panic("configuration must be initialized before starting the server")
	} else if Logger == nil {
		panic("logger must be initialized before starting the server")
	}

	// Start the shipgate service and make sure it launches before the other servers start.
	shipgate.Init(shipgate.DBConfig{
		Engine:         Config.Database.Engine,
		Filename:       Config.QualifiedPath(Config.Database.Filename),
		URL:            Config.DatabaseURL(),
		LoggingEnabled: Config.Debugging.DatabaseLoggingEnabled,
	}, Logger)

	// Stop the shipgate after all of the other servers have stopped in order to avoid
	// errors from any shipgate calls during the shutdown process.
	defer shipgate.Shutdown(ctx)

	runServers(ctx)
}

func runServers(ctx context.Context) {
	// Set up all of the client-facing servers we'll be running.
	servers := map[int]Backend{
		Config.PatchServer.AuthPort:     &PatchAuthServer{},
		Config.PatchServer.DataPort:     &PatchServer{},
		Config.CharacterServer.AuthPort: &CharacterAuthServer{},
		Config.CharacterServer.DataPort: &CharacterServer{},
		// Note: Eventually the ship and block servers should be able to be run
		// independently of the other four servers
		Config.ShipServer.AuthPort: &GameAuthServer{},
		Config.ShipServer.GamePort: &GameServer{},
	}

	// Configure, initialize, run all of our servers.
	var wg sync.WaitGroup
	for _, server := range servers {
		if err := server.Init(ctx); err != nil {
			Logger.Errorf("error initializing %s server: %v", server.Identifier(), err)
			return
		}
	}
	for port, server := range servers {
		wg.Add(1)
		address := fmt.Sprintf("%s:%v", Config.Hostname, port)
		if err := startAccepting(ctx, &wg, address, server); err != nil {
			Logger.Errorf("error starting %s server: %v", server.Identifier(), err)
			return
		}
	}
	wg.Wait()
}

// Backend is an interface for a sub-server that handles a specific set of client
// interactions as part of the game flow.
type Backend interface {
	// Name returns a uniquely identifying string.
	Identifier() string

	// Init is called before a Backend is started as a hook for the Backend to
	// perform any necessary initialization before it can accept clients.
	Init(ctx context.Context) error

	// Handshake performs any connection initialization necessary to begin
	// communicating with the client. This likely involves sending a "welcome"
	// command and choosing/initializing the appropriate encryption implementation.
	Handshake(ctx context.Context, c *Client) error

	// Handle is the main entry point for processing client commands. It's responsible
	// for generally handling all commands from a client as well as sending any responses.
	Handle(ctx context.Context, c *Client, data []byte) error
}

func startAccepting(ctx context.Context, wg *sync.WaitGroup, addr string, backend Backend) error {
	hostAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return fmt.Errorf("error resolving address %s", err.Error())
	}

	socket, err := net.ListenTCP("tcp", hostAddr)
	if err != nil {
		return fmt.Errorf("error listening on socket: %s", err.Error())
	}

	Logger.Infof("[%s] waiting for connections on %v", backend.Identifier(), addr)
	go startHandlingConnections(ctx, wg, socket, backend)
	return nil
}

var (
	numClients       atomic.Uint32
	connectedClients sync.Map
)

// var connectedClients = make(map[string]*Client)

// startHandlingConnections implements a connection handling loop that's purely responsible for
// accepting new connections and spinning off goroutines for the Backend to handle them.
func startHandlingConnections(ctx context.Context, wg *sync.WaitGroup, socket *net.TCPListener, backend Backend) {
	defer wg.Done()

	connections := make(chan *net.TCPConn)
	go func() {
		for {
			// Poll until we can accept more clients.
			for numClients.Load() >= uint32(Config.MaxConnections) {
				time.Sleep(5 * time.Second)
			}

			connection, err := socket.AcceptTCP()
			if err != nil {
				Logger.Warnf("failed to accept connection: %s", err.Error())
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
			go func() {
				defer clientWg.Done()
				acceptClient(ctx, backend, connection)
			}()
		}
	}

	Logger.Infof("[%v] shutting down (waiting for connections to close)", backend.Identifier())
	clientWg.Wait()
	Logger.Infof("[%v] exited", backend.Identifier())
}

// acceptClient takes a connection and attempts to initiate a "session" by setting up
// the Client and sending the welcome command. If it succeeds, the goroutine moves
// into the packet processing loop.
func acceptClient(ctx context.Context, backend Backend, conn *net.TCPConn) {
	c := NewClient(conn)

	clientCtx, clientCancel := context.WithCancel(debug.WithServerContext(ctx, backend.Identifier()))
	defer func() {
		clientCancel()
		if err := recover(); err != nil {
			Logger.Errorf("error communicating with client %s: error=%s, trace: %s", c.IPAddr, err, rdbg.Stack())
		}

		// This is a bit of a hack, but we need to make sure that any disconnected clients
		// are always removed from the lobby they were in.
		if gs, ok := backend.(*GameServer); ok {
			gs.cleanupDisconnectedClient(ctx, c)
		}

		closeConnection(c)
		Logger.Infof("[%s] disconnected client %s", backend.Identifier(), c.IPAddr)
	}()

	Logger.Infof("[%s] accepted connection from %s", backend.Identifier(), c.IPAddr)

	if err := backend.Handshake(clientCtx, c); err != nil {
		Logger.Errorf("Handshake() failed for client %s: %s", c.IPAddr, err)
	}

	// Prevent multiple clients from connecting from the same IP address.
	if _, ok := connectedClients.Load(c.IPAddr); ok {
		Logger.Infof("[%s] rejected second connection from %s", backend.Identifier(), c.IPAddr)
		_ = conn.Close()
		return
	}

	connectedClients.Store(c.IPAddr, c)
	processPackets(clientCtx, backend, c)
}

// processPackets starts a blocking loop dedicated to reading data sent from
// a game client and only returns once the connection has closed.
func processPackets(ctx context.Context, backend Backend, c *Client) {
	buffer := make([]byte, 2048)
	var err error

	for {
		select {
		case <-ctx.Done():
			// For now just allow the deferred function to close the connection.
			return
		default:
		}

		buffer, err = readNextPacket(c, buffer)

		if err == io.EOF {
			break
		} else if err != nil {
			Logger.Warn(err.Error())
			break
		}

		if Config.Debugging.PacketLoggingEnabled {
			debug.PrintPacket(ctx, debug.PrintPacketParams{
				Writer:        bufio.NewWriter(os.Stdout),
				ClientCommand: true,
				Data:          buffer,
			})
		}

		if err = backend.Handle(ctx, c, buffer); err != nil {
			Logger.Warnf("error communicating with client %s: %s", c.IPAddr, err)
			return
		}
	}
}

// closeConnection disconnects the client and removes them from the list
// regardless of the state of the connection.
func closeConnection(c *Client) {
	if err := c.Close(); err != nil {
		Logger.Warnf("error closing client connection: %s", err)
	}
	connectedClients.Delete(c.IPAddr)
}

// readNextPacket is a blocking call that only returns once the client has
// sent the next packet to be processed. The buffer in c.ConnectionState is
// updated with the decrypted packet.
func readNextPacket(c *Client, buffer []byte) ([]byte, error) {
	headerSize := int(c.CryptoSession.HeaderSize())

	// Read and decrypt the packet header.
	if err := readDataFromClient(c, headerSize, buffer); err != nil {
		return buffer, err
	}

	c.CryptoSession.Decrypt(buffer[:headerSize], uint32(headerSize))

	packetSize := determinePacketSize(buffer[:2], uint16(headerSize))

	// Grow the client's receive buffer if they send us a packet bigger than its current capacity.
	if packetSize > cap(buffer) {
		newBuf := make([]byte, cap(buffer)+packetSize)
		copy(newBuf, buffer)
		buffer = newBuf
	}

	// Read and decrypt the rest of the packet.
	if err := readDataFromClient(c, packetSize-headerSize, buffer[headerSize:]); err != nil {
		return buffer, err
	}

	c.CryptoSession.Decrypt(buffer[headerSize:packetSize], uint32(packetSize-headerSize))

	return buffer, nil
}

func readDataFromClient(c *Client, n int, buffer []byte) error {
	received := 0

	for received < n {
		bytesRead, err := c.Read(buffer[received:n])
		received += bytesRead

		if bytesRead == 0 || err == io.EOF {
			return err
		} else if err != nil {
			return errors.New("socket error (" + c.IPAddr + ") " + err.Error())
		}
	}

	return nil
}

// Extract the packet length from the first two bytes of data.
func determinePacketSize(data []byte, headerSize uint16) int {
	if len(data) < 2 {
		// Panic since this shouldn't happen unless something's very wrong.
		panic(errors.New("getSize(): data must be at least two bytes"))
	}

	var size uint16
	reader := bytes.NewReader(data)
	err := binary.Read(reader, binary.LittleEndian, &size)

	if err != nil {
		Logger.Warn("error decoding packet size:", err)
	}

	// The PSO client occasionally sends packets that are longer than their declared
	// size, but are always a multiple of the length of the packet header. Adjust the
	// expected length just in case in order to avoid leaving stray bytes in the buffer.
	size += size % headerSize

	return int(size)
}
