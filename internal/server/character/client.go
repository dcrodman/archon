package character

import (
	"github.com/dcrodman/archon/internal/data"
	"github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/internal/packets"
)

// Client implementation for the CHARACTER server.
type characterClientExtension struct {
	clientCrypt *encryption.PSOCrypt
	serverCrypt *encryption.PSOCrypt

	account *data.Account
}

func (c *characterClientExtension) HeaderSize() uint16 {
	return packets.BBHeaderSize
}

func (c *characterClientExtension) Encrypt(bytes []byte, length uint32) {
	c.serverCrypt.Encrypt(bytes, length)
}

func (c *characterClientExtension) Decrypt(bytes []byte, length uint32) {
	c.clientCrypt.Decrypt(bytes, length)
}

func (c *characterClientExtension) DebugInfo() map[string]interface{} {
	return map[string]interface{}{
		"server_type": "character",
	}
}
