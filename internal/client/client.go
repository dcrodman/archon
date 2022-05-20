package client

import (
	"fmt"
	"net"
	"strings"

	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/data"
	"github.com/dcrodman/archon/internal/core/debug"
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
	ipAddr     string
	port       string

	// Cipher implementation responsible for packet encryption.
	CryptoSession CryptoSession

	// Account associated with the player.
	Account *data.Account

	// Client information shared amongst most Backend implementations.
	Config ClientConfig

	Flag   uint32
	TeamID uint32
	IsGm   bool
	// Guildcard linked to the account.
	Guildcard     uint32
	GuildcardData []byte

	// File list used exclusively by the Data server for tracking which
	// files need updating. TODO: This ought to be expressed more gracefully
	// but we have very little information by which we can identify a unique
	// PSO client in the patch phase and this is easy so...here we are.
	FilesToUpdate map[int]interface{}

	// Debugging information used for logging purposes.
	DebugTags map[string]interface{}
}

func NewClient(connection *net.TCPConn) *Client {
	addr := strings.Split(connection.RemoteAddr().String(), ":")

	return &Client{
		connection: connection,
		ipAddr:     addr[0],
		port:       addr[1],
		DebugTags:  make(map[string]interface{}),
	}
}

func (c *Client) IPAddr() string { return c.ipAddr }
func (c *Client) Port() string   { return c.port }

// Read consumes the available bytes directly the client's TCP connection.
func (c *Client) Read(b []byte) (int, error) {
	return c.connection.Read(b)
}

// Write directly sends data to the client over its TCP connection.
func (c *Client) Write(bytes []byte) (int, error) {
	return c.connection.Write(bytes)
}

// Close the TCP connection.
func (c *Client) Close() error {
	return c.connection.Close()
}

// SendRaw writes all data contained in the slice to the client
// as-is (e.g. without encrypting it first).
func (c *Client) SendRaw(packet interface{}) error {
	bytes, size := bytes.BytesFromStruct(packet)

	if debug.Enabled() {
		debug.SendServerPacketToAnalyzer(c.DebugTags, bytes, uint16(size))
	}

	return c.transmit(bytes, uint16(size))
}

// transmit writes the contents of data to the TCP connection until the number
// of bytes written >= length.
func (c *Client) transmit(data []byte, length uint16) error {
	bytesSent := 0

	for bytesSent < int(length) {
		b, err := c.Write(data[:length])
		if err != nil {
			return fmt.Errorf("failed to send to client %v: %s", c.IPAddr(), err.Error())
		}
		bytesSent += b
	}

	return nil
}

// Send converts a packet struct to bytes and encrypts it before  using the
// server's session key before sending the data to the client.
func (c *Client) Send(packet interface{}) error {
	data, length := bytes.BytesFromStruct(packet)
	bytes, size := adjustPacketLength(data, uint16(length), c.CryptoSession.HeaderSize())

	if debug.Enabled() {
		debug.SendServerPacketToAnalyzer(c.DebugTags, bytes, size)
	}

	c.CryptoSession.Encrypt(bytes, uint32(size))
	return c.transmit(bytes, size)
}

// adjustPacketLength pads the length of a packet to a multiple of the header length and
// adjusts first two bytes of the header to the corrected size (may be a no-op). PSOBB
// clients will reject packets that are not padded in this manner.
func adjustPacketLength(data []byte, length uint16, headerSize uint16) ([]byte, uint16) {
	for length%headerSize != 0 {
		length++
		data = append(data, 0)
	}

	data[0] = byte(length & 0xFF)
	data[1] = byte((length & 0xFF00) >> 8)

	return data, length
}
