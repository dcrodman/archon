package internal

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/dcrodman/archon/internal/data"
	"github.com/dcrodman/archon/internal/debug"
	"github.com/dcrodman/archon/internal/encryption"
)

type ClientConfig struct {
	// The rest of this holds various portions of client state to represent
	// the client's progression through the login process.
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

	IPAddr string
	Port   string

	// Cipher implementation responsible for packet encryption.
	CryptoSession encryption.CryptoSession

	// Account associated with the player.
	Account *data.Account

	// Client information shared amongst most Backend implementations.
	Config ClientConfig

	Flag   uint32
	TeamID uint32
	IsGm   bool
	// The slot corresponding to the currently active character.
	ActiveSlot uint32
	// Guildcard linked to the account.
	Guildcard     uint32
	GuildcardData []byte

	// File list used exclusively by the Patch server for tracking which files need updating.
	FilesToUpdate map[int]interface{}
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

	c.CryptoSession.Encrypt(bytes, uint32(size))
	return c.transmit(bytes, size)
}

// transmit writes the contents of data to the TCP connection until the number
// of bytes written >= length.
func (c *Client) transmit(data []byte, length uint16) error {
	bytesSent := 0

	// TODO: Ought to wire up a cancel here, though it's a bit tricky.
	for bytesSent < int(length) {
		b, err := c.connection.Write(data[:length])
		if err != nil {
			return fmt.Errorf("error sending to client %v: %s", c.IPAddr, err.Error())
		}
		bytesSent += b
	}

	return nil
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
