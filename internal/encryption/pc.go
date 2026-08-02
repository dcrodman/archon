package encryption

type PCCryptSession struct {
	clientCrypt *pcCipher
	serverCrypt *pcCipher
}

// NewPCCryptoSession returns a CryptoSession with newly initialized
// ciphers that can be used to communicate with either a PSO PC client or the
// patch protocol used by the PSO Blue Burst client.
func NewPCCryptoSession() CryptoSession {
	return NewPCCryptoSessionWithVector(createKey(4), createKey(4))
}

func NewPCCryptoSessionWithVector(clientVec, serverVec []uint8) CryptoSession {
	return &PCCryptSession{
		clientCrypt: newPCCipher(clientVec),
		serverCrypt: newPCCipher(serverVec),
	}
}

func (c *PCCryptSession) HeaderSize() uint16 {
	return PSOPCBlockSize
}
func (c *PCCryptSession) Encrypt(bytes []byte, length uint32) {
	c.serverCrypt.Encrypt(bytes, length)
}

func (c *PCCryptSession) DecryptServer(bytes []byte, length uint32) {
	c.serverCrypt.Decrypt(bytes, length)
}

func (c *PCCryptSession) Decrypt(bytes []byte, length uint32) {
	c.clientCrypt.Decrypt(bytes, length)
}

func (c *PCCryptSession) EncryptClient(bytes []byte, length uint32) {
	c.clientCrypt.Encrypt(bytes, length)
}

func (c *PCCryptSession) ClientVector() []byte {
	return c.clientCrypt.vector
}

func (c *PCCryptSession) ServerVector() []byte {
	return c.serverCrypt.vector
}
