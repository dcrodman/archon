package login

import (
	"github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/internal/packets"
)

// Client implementation for the LOGIN server.
type loginClientExtension struct {
	clientCrypt *encryption.PSOCrypt
	serverCrypt *encryption.PSOCrypt
}

func (c *loginClientExtension) HeaderSize() uint16 {
	return packets.BBHeaderSize
}

func (c *loginClientExtension) Encrypt(bytes []byte, length uint32) {
	c.serverCrypt.Encrypt(bytes, length)
}

func (c *loginClientExtension) Decrypt(bytes []byte, length uint32) {
	c.clientCrypt.Decrypt(bytes, length)
}

func (c *loginClientExtension) DebugInfo() map[string]interface{} {
	return map[string]interface{}{
		"server_type": "login",
	}
}
