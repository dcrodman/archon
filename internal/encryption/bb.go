package encryption

type BlueBurstCryptSession struct {
	clientCrypt *BlowfishCipher
	serverCrypt *BlowfishCipher
}

// NewBlueBurstCryptoSession returns a CryptoSession with newly initialized
// ciphers that can be used to communicate with a PSO Blue Burst client.
func NewBlueBurstCryptoSession() CryptoSession {
	return NewBlueBurstCryptoSessionWithVector(createKey(48), createKey(48))
}

// NewBlueBurstCryptoSessionWithVector returns a CryptoSession with the specified encryption
// vectors that can be used to communicate with a PSO Blue Burst client.
func NewBlueBurstCryptoSessionWithVector(clientVec, serverVec []uint8) CryptoSession {
	return &BlueBurstCryptSession{
		clientCrypt: newBlowfishCipher(clientVec),
		serverCrypt: newBlowfishCipher(serverVec),
	}
}

func (c *BlueBurstCryptSession) HeaderSize() uint16 {
	return BlowfishBlockSize
}

func (c *BlueBurstCryptSession) Encrypt(bytes []byte, length uint32) {
	c.serverCrypt.Encrypt(bytes, length)
}

func (c *BlueBurstCryptSession) DecryptServer(bytes []byte, length uint32) {
	c.serverCrypt.Decrypt(bytes, length)
}

func (c *BlueBurstCryptSession) Decrypt(bytes []byte, length uint32) {
	c.clientCrypt.Decrypt(bytes, length)
}

func (c *BlueBurstCryptSession) EncryptClient(bytes []byte, length uint32) {
	c.clientCrypt.Encrypt(bytes, length)
}

func (c *BlueBurstCryptSession) ClientVector() []byte {
	return c.clientCrypt.vector
}

func (c *BlueBurstCryptSession) ServerVector() []byte {
	return c.serverCrypt.vector
}
