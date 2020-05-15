package ship

import (
	"github.com/dcrodman/archon/internal/data"
	"github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/internal/packets"
)

type shipClientExtension struct {
	clientCrypt *encryption.PSOCrypt
	serverCrypt *encryption.PSOCrypt

	account *data.Account
}

func (c *shipClientExtension) HeaderSize() uint16 {
	return packets.BBHeaderSize
}

func (c *shipClientExtension) Encrypt(bytes []byte, length uint32) {
	c.serverCrypt.Encrypt(bytes, length)
}

func (c *shipClientExtension) Decrypt(bytes []byte, length uint32) {
	c.clientCrypt.Decrypt(bytes, length)
}

func (c *shipClientExtension) DebugInfo() map[string]interface{} {
	return map[string]interface{}{
		"server_type": "ship",
	}
}
