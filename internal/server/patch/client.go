package patch

import (
	"github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/internal/packets"
)

// ClientExtension implementation for the PATCH and DATA servers.
type patchClientExtension struct {
	clientCrypt *encryption.PSOCrypt
	serverCrypt *encryption.PSOCrypt

	// Set of files that need to be updated.
	filesToUpdate map[int]*fileEntry
}

func (c *patchClientExtension) HeaderSize() uint16 {
	return packets.PCHeaderSize
}

func (c *patchClientExtension) Encrypt(bytes []byte, length uint32) {
	c.serverCrypt.Encrypt(bytes, length)
}

func (c *patchClientExtension) Decrypt(bytes []byte, length uint32) {
	c.clientCrypt.Decrypt(bytes, length)
}

func (c *patchClientExtension) DebugInfo() map[string]interface{} {
	return map[string]interface{}{
		"server_type": "patch",
	}
}
