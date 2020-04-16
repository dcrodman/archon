/*
 * Client definition for generic handling of connections.
 */
package server

import (
	"github.com/dcrodman/archon/packets"
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

func (cs *ConnectionState) IPAddr() string { return cs.ipAddr }
func (cs *ConnectionState) Port() string   { return cs.port }
func (cs *ConnectionState) Data() []byte   { return cs.buffer }

func (cs *ConnectionState) WriteBytes(bytes []byte) (int, error) {
	return cs.connection.Write(bytes)
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

	DebugInfo() map[string]interface{}
}

// CommonClient encapsulates the user-specific information common to several server
// implementations and is intended to be embedded by any Client2 instance that needs it.
type CommonClient struct {
	Config packets.ClientConfig

	Flag   uint32
	TeamId uint32
	IsGm   bool

	Guildcard     uint32
	GuildcardData []byte
}
