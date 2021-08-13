package client

import (
	encryption2 "github.com/dcrodman/archon/internal/core/encryption"
)

// CryptoSession is an interface for the cryptographic operations required
// to exchange packets between a PSO game client and the server. It consists
// of one or more ciphers that handle encrypting packets from the server and
// decrypting packets from the client.
type CryptoSession interface {
	// HeaderSize returns the length of the header of all client packets.
	HeaderSize() uint16

	// Encrypt encrypts bytes in place with the encryption key for the server.
	Encrypt(bytes []byte, length uint32)

	// Decrypt decrypts bytes in place with the encryption key for the client.
	Decrypt(bytes []byte, length uint32)

	// ServerVector returns the key used to initialize the server's block cipher.
	ServerVector() []byte

	// ClientVector returns the key used to initialize the client's's block cipher.
	ClientVector() []byte
}

type blueBurstCryptSession struct {
	clientCrypt *encryption2.PSOCrypt
	serverCrypt *encryption2.PSOCrypt
}

// NewBlueBurstCryptoSession returns a CryptoSession with newly initialized
// ciphers that can be used to communicate with a PSO Blue Burst client.
func NewBlueBurstCryptoSession() CryptoSession {
	return &blueBurstCryptSession{
		serverCrypt: encryption2.NewBBCrypt(),
		clientCrypt: encryption2.NewBBCrypt(),
	}
}

func (c *blueBurstCryptSession) HeaderSize() uint16 {
	return encryption2.BlowfishBlockSize
}

func (c *blueBurstCryptSession) Encrypt(bytes []byte, length uint32) {
	c.serverCrypt.Encrypt(bytes, length)
}

func (c *blueBurstCryptSession) Decrypt(bytes []byte, length uint32) {
	c.clientCrypt.Decrypt(bytes, length)
}

func (c *blueBurstCryptSession) ServerVector() []byte {
	return c.serverCrypt.Vector
}

func (c *blueBurstCryptSession) ClientVector() []byte {
	return c.clientCrypt.Vector
}

type pcCryptSession struct {
	clientCrypt *encryption2.PSOCrypt
	serverCrypt *encryption2.PSOCrypt
}

// NewPCCryptoSession returns a CryptoSession with newly initialized
// ciphers that can be used to communicate with either a PSO PC client or the
// patch protocol used by the PSO Blue Burst client.
func NewPCCryptoSession() CryptoSession {
	return &pcCryptSession{
		serverCrypt: encryption2.NewPCCrypt(),
		clientCrypt: encryption2.NewPCCrypt(),
	}
}

func (c *pcCryptSession) HeaderSize() uint16 {
	return encryption2.PSOPCBlockSize
}

func (c *pcCryptSession) Encrypt(bytes []byte, length uint32) {
	c.serverCrypt.Encrypt(bytes, length)
}

func (c *pcCryptSession) Decrypt(bytes []byte, length uint32) {
	c.clientCrypt.Decrypt(bytes, length)
}

func (c *pcCryptSession) ServerVector() []byte {
	return c.serverCrypt.Vector
}

func (c *pcCryptSession) ClientVector() []byte {
	return c.clientCrypt.Vector
}
