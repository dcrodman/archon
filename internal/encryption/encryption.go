/*
* Blowfish implementation adapted to work with PSOBB's protocol.
 */
package encryption

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

// CryptoSession is an interface for the cryptographic operations required
// to exchange commands between a PSO game client and the server. It consists
// of one or more ciphers that handle encrypting commands from the server and
// decrypting commands from the client.
type CryptoSession interface {
	// HeaderSize returns the length of the header of all client commands.
	HeaderSize() uint16

	// Encrypt encrypts bytes in place with the encryption key for the server.
	Encrypt(bytes []byte, length uint32)

	// Decrypt decrypts bytes in place with the encryption key for the client.
	Decrypt(bytes []byte, length uint32)

	// ServerVector returns the key used to initialize the server's block cipher.
	ServerVector() []byte

	// ClientVector returns the key used to initialize the client's's block cipher.
	ClientVector() []byte

	// Neither of these methods are used for communicating with the game but are
	// available for tests to use.
	DecryptServer(bytes []byte, length uint32)
	EncryptClient(bytes []byte, length uint32)
}

// Generate a cryptographically secure random string of bytes.
func createKey(size int) []byte {
	key := make([]byte, size)

	for i := 0; i < size; i++ {
		if err := binary.Read(rand.Reader, binary.LittleEndian, &key[i]); err != nil {
			panic(fmt.Errorf("error creating key: %v", err))
		}
	}

	return key
}

// Condense four bytes into a LE 32-bit value.
func toLittleEndian(b []byte) uint32 {
	for i := 3; len(b) < 4; i-- {
		b = append(b, 0)
	}
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}
