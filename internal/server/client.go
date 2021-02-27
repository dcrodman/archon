package server

import (
	"fmt"
	"net"
	"strings"

	"github.com/dcrodman/archon/internal/debug"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/dcrodman/archon/internal/server/internal"
)

// ClientExtension is an interface for implementing Backend-specific behavior
// or state required by one of the sub-servers.
type ClientExtension interface {
	// HeaderSize returns the length of the header of all client packets.
	HeaderSize() uint16

	// Encrypt encrypts bytes in place with the encryption key for the client.
	Encrypt(bytes []byte, length uint32)

	// Encrypt decrypts bytes in place with the encryption key for the client.
	Decrypt(bytes []byte, length uint32)

	// DebugInfo returns a set of KV pairs used for server debugging/logging.
	DebugInfo() map[string]interface{}
}

// Client represents a user connected through a PSOBB game client.
type Client struct {
	connection *net.TCPConn
	ipAddr     string
	port       string

	Extension ClientExtension

	// Client information shared amongst most Backend implementations.
	Config packets.ClientConfig

	Flag   uint32
	TeamID uint32
	IsGm   bool

	Guildcard     uint32
	GuildcardData []byte
}

func NewClient(connection *net.TCPConn) *Client {
	addr := strings.Split(connection.RemoteAddr().String(), ":")

	return &Client{
		connection: connection,
		ipAddr:     addr[0],
		port:       addr[1],
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
	bytes, size := internal.BytesFromStruct(packet)

	if debug.Enabled() {
		debug.SendServerPacketToAnalyzer(c.Extension.DebugInfo(), bytes, uint16(size))
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

// send converts a packet struct to bytes and encrypts it before  using the
// server's session key before sending the data to the client.
func (c *Client) Send(packet interface{}) error {
	data, length := internal.BytesFromStruct(packet)
	bytes, size := adjustPacketLength(data, uint16(length), c.Extension.HeaderSize())

	if debug.Enabled() {
		debug.SendServerPacketToAnalyzer(c.Extension.DebugInfo(), bytes, size)
	}

	c.Extension.Encrypt(bytes, uint32(size))
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
