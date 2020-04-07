/*
 * Client definition for generic handling of connections.
 */
package server

import (
	"github.com/dcrodman/archon"
	"net"
	"strings"
)

// ConnectionState encapsulates the TCP connection and any other data required
// to interact with a client in its simplest form.
type ConnectionState struct {
	connection *net.TCPConn
	ipAddr     string
	port       string
	buffer     []byte
}

func newConnectionState(conn *net.TCPConn) *ConnectionState {
	addr := strings.Split(conn.RemoteAddr().String(), ":")

	return &ConnectionState{
		connection: conn,
		ipAddr:     addr[0],
		port:       addr[1],
		buffer:     make([]byte, 1024),
	}
}

func (c *ConnectionState) IPAddr() string { return c.ipAddr }
func (c *ConnectionState) Port() string   { return c.port }
func (c *ConnectionState) Data() []byte   { return c.buffer }

func (c *ConnectionState) WriteBytes(bytes []byte) (int, error) {
	return c.connection.Write(bytes)
}

// Client2 is a wrapper interface that allows the server package functions to
// perform the generic client communication while allowing the Server implementations
// to maintain session-specific data.
type Client2 interface {
	// ConnectionState returns a pointer to the internal client state.
	ConnectionState() *ConnectionState

	// Encrypt encrypts bytes in place with the encryption key for the client.
	Encrypt(bytes []byte, length uint32)

	// Encrypt decrypts bytes in place with the encryption key for the client.
	Decrypt(bytes []byte, length uint32)
}

// CommonClient encapsulates the user-specific information common to several server
// implementations and is intended to be embedded by any Client2 instance that needs it.
type CommonClient struct {
	Config archon.ClientConfig

	Flag   uint32
	TeamId uint32
	IsGm   bool

	Guildcard         uint32
	GuildcardData     []byte
	GuildcardDataSize uint16
}
