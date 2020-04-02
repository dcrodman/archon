package patch

import (
	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/server"
	"github.com/dcrodman/archon/util"
	"github.com/dcrodman/archon/util/encryption"
	"github.com/dcrodman/archon/util/relay"
)

// Client2 implementation for the PATCH and DATA servers.
type Client struct {
	cs *server.ConnectionState

	clientCrypt *encryption.PSOCrypt
	serverCrypt *encryption.PSOCrypt

	// Set of files that need to be updated.
	filesToUpdate map[int]*fileEntry
}

func (c Client) ConnectionState() *server.ConnectionState {
	return c.cs
}

func (c Client) Encrypt(bytes []byte, length uint32) {
	c.serverCrypt.Encrypt(bytes, length)
}

func (c Client) Decrypt(bytes []byte, length uint32) {
	c.clientCrypt.Decrypt(bytes, length)
}

func (c *Client) clientVector() []uint8 { return c.clientCrypt.Vector }
func (c *Client) serverVector() []uint8 { return c.serverCrypt.Vector }

func (c *Client) send(packet interface{}) error {
	return relay.SendPacket(c, packet, archon.PCHeaderSize)
}

func (c *Client) sendRaw(packet interface{}) error {
	data, size := util.BytesFromStruct(packet)
	return relay.SendRaw(c, data, uint16(size))
}
