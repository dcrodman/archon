package internal

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/dcrodman/archon/internal/commands"
	"github.com/dcrodman/archon/internal/data"
	"github.com/dcrodman/archon/internal/debug"
	"github.com/dcrodman/archon/internal/encryption"
)

// Joinable may be either a Game or a Lobby joined by a player.
type Room interface {
	RoomName() []byte
	AddClient(ctx context.Context, c *Client) error
	RemoveClient(ctx context.Context, c *Client)
	Broadcast(ctx context.Context, sender *Client, cmd commands.Broadcast)
}

type ClientConfig struct {
	Magic        uint32 // Must be set to 0x48615467
	CharSelected uint8  // Has a character been selected?
	SlotNum      uint8  // Slot number of selected Character
	Flags        uint16
	Ports        [4]uint16
	Unused       [4]uint32
	Unused2      [2]uint32
}

// Client represents a user connected through a PSOBB game client.
type Client struct {
	connection *net.TCPConn
	// transmitLock must be held when using connection to send commands
	// to prevent the packets from being interleaved and corrupted.
	transmitLock sync.Mutex

	IPAddr string
	Port   string

	// Cipher implementation responsible for packet encryption on the connection.
	CryptoSession encryption.CryptoSession

	// Account associated with the player.
	Account *data.Account

	// ClientConfig contains the various portions of client state to represent
	// the client's progression through the login process.
	Config ClientConfig
	// LoginPhase is stored from the login packet (93) and indicates which connection
	// the client has initiated.
	LoginPhase commands.LoginPhase

	// These fields may be included in commands send to other players and therefore may be
	// accessed concurrently. All access is guarded by the embedded mutex.
	sync.Mutex
	// Character is a reference to the character selected during the login process.
	Character *commands.CharacterData
	// Joinable is the Lobby or Game the player is currently in.
	Room Room
	// LobbySlotID is the position in the the lobby or game.
	LobbySlotID uint8

	// The following fields are server-specific and mainly used as a place to store
	// client-specific information without needing a separate structure.

	// Patch server only; file list for tracking which files need updating.
	FilesToUpdate map[int]interface{}
	// Character server only; cached guildcard data for sending in chunked transfer.
	GuildcardData []byte
}

func NewClient(connection *net.TCPConn) *Client {
	addr := strings.Split(connection.RemoteAddr().String(), ":")

	return &Client{
		connection: connection,
		IPAddr:     addr[0],
		Port:       addr[1],
	}
}

// Read consumes the available bytes directly the client's TCP connection.
func (c *Client) Read(b []byte) (int, error) {
	return c.connection.Read(b)
}

// Close the TCP connection.
func (c *Client) Close() error {
	return c.connection.Close()
}

// SendRaw writes all data contained in the slice to the client
// as-is (e.g. without encrypting it first).
func (c *Client) SendRaw(ctx context.Context, packet interface{}) error {
	bytes, size := MarshalStruct(packet)

	if Config.Debugging.PacketLoggingEnabled {
		debug.PrintPacket(ctx, debug.PrintPacketParams{
			Writer:        bufio.NewWriter(os.Stdout),
			ClientCommand: false,
			Data:          bytes,
		})
	}

	c.transmitLock.Lock()
	defer c.transmitLock.Unlock()

	return c.transmit(bytes, uint16(size))
}

// Send converts a packet struct to bytes and encrypts it before  using the
// server's session key before sending the data to the client.
func (c *Client) Send(ctx context.Context, packet interface{}) error {
	data, length := MarshalStruct(packet)
	bytes, size := adjustPacketLength(data, uint16(length), c.CryptoSession.HeaderSize())

	if Config.Debugging.PacketLoggingEnabled {
		debug.PrintPacket(ctx, debug.PrintPacketParams{
			Writer:        bufio.NewWriter(os.Stdout),
			ClientCommand: false,
			Data:          bytes,
		})
	}

	c.transmitLock.Lock()
	defer c.transmitLock.Unlock()

	c.CryptoSession.Encrypt(bytes, uint32(size))
	return c.transmit(bytes, size)
}

// adjustPacketLength pads the length of a packet to a multiple of the header length and
// adjusts first two bytes of the header to the corrected size (may be a no-op). Returns
// the adjusted packet as well as the new length.
//
// PSOBB clients will reject packets that are not multiples of the header size.
func adjustPacketLength(data []byte, length uint16, headerSize uint16) ([]byte, uint16) {
	for length%headerSize != 0 {
		length++
		data = append(data, 0)
	}

	data[0] = byte(length & 0xFF)
	data[1] = byte((length & 0xFF00) >> 8)

	return data, length
}

// transmit writes the contents of data to the TCP connection until the number
// of bytes written >= length.
func (c *Client) transmit(data []byte, length uint16) error {
	// TODO: Ought to wire up a cancel here, though it's a bit tricky.
	bytesSent := 0
	for bytesSent < int(length) {
		b, err := c.connection.Write(data[:length])
		if err != nil {
			return fmt.Errorf("error sending to client %v: %s", c.IPAddr, err.Error())
		}
		bytesSent += b
	}
	return nil
}
