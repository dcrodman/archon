package character

import (
	"github.com/dcrodman/archon/internal/data"
	"github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/dcrodman/archon/internal/server"
	"github.com/dcrodman/archon/internal/server/internal"
	"github.com/dcrodman/archon/internal/server/internal/relay"
)

// Client implementation for the CHARACTER server.
type client struct {
	cs *server.ConnectionState

	clientCrypt *encryption.PSOCrypt
	serverCrypt *encryption.PSOCrypt

	server.CommonClient

	account *data.Account
}

func (c *client) ConnectionState() *server.ConnectionState { return c.cs }

func (c *client) clientVector() []uint8 { return c.clientCrypt.Vector }
func (c *client) serverVector() []uint8 { return c.serverCrypt.Vector }

func (c *client) Encrypt(bytes []byte, length uint32) {
	c.serverCrypt.Encrypt(bytes, length)
}

func (c *client) Decrypt(bytes []byte, length uint32) {
	c.clientCrypt.Decrypt(bytes, length)
}

func (c *client) DebugInfo() map[string]interface{} {
	return map[string]interface{}{
		"server_type": "character",
	}
}

func (c *client) send(packet interface{}) error {
	return relay.SendPacket(c, packet, packets.BBHeaderSize)
}

func (c *client) sendRaw(packet interface{}) error {
	bytes, size := internal.BytesFromStruct(packet)
	return relay.SendRaw(c, bytes, uint16(size))
}
