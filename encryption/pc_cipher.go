// PSOPC encryption algorithm. Implementation based on (and in some cases basically
// copied from) the encryption library included with Fuzziqer Software's newserv code.
//
// Note: The cipher used PC/Gamecrube is not symmetrical in that decrypting a block
// previously encrypted with this cipher will not yield the same result due (I think)
// to the key rotations.
package encryption

import "strconv"

const PCBlockSize = 4

type PCCrypt struct {
	seed     uint32
	position uint32
	keys     []uint32
}

type PCKeySizeError int

func (k PCKeySizeError) Error() string {
	return "encryption/pccrypt: invalid key size " + strconv.Itoa(int(k))
}

func newPCCipher(key []byte) (psoCipher, error) {
	if len(key) > 4 {
		return nil, PCKeySizeError(1)
	}
	// Key is expected to be in little endian.
	crypt := &PCCrypt{seed: le(key), position: 0, keys: make([]uint32, 57)}
	crypt.createKeys()
	return crypt, nil
}

func (crypt *PCCrypt) blockSize() int { return PCBlockSize }

// Initialize the cipher.
func (crypt *PCCrypt) createKeys() {
	x := uint32(1)
	key := crypt.seed
	crypt.keys[56], crypt.keys[55] = key, key

	for i := 0x15; i <= 0x46E; i += 0x15 {
		j := i % 55
		key -= x
		crypt.keys[j] = x
		x = key
		key = crypt.keys[j]
	}

	for i := 0; i < 4; i++ {
		crypt.mixKeys()
	}

	crypt.position = 56
}

func (crypt *PCCrypt) mixKeys() {
	initial := 1
	for i := 0x18; i > 0; i-- {
		x := crypt.keys[initial+0x1F]
		y := crypt.keys[initial]
		y -= x
		crypt.keys[initial] = y
		initial++
	}
	initial = 0x19
	for i := 0x1F; i > 0; i-- {
		x := crypt.keys[initial-0x18]
		y := crypt.keys[initial]
		y -= x
		crypt.keys[initial] = y
		initial++
	}
}

func (crypt *PCCrypt) getNextKey() uint32 {
	var re uint32
	if crypt.position == 56 {
		crypt.mixKeys()
		crypt.position = 1
	}
	re = crypt.keys[crypt.position]
	crypt.position++
	return re
}

func (crypt *PCCrypt) encrypt(src []byte) {
	crypt.process(src, len(src))
}

func (crypt *PCCrypt) decrypt(src []byte) {
	crypt.process(src, len(src))
}

// Perform the actual encryption/decryption. The operation is
// symmetrical, so the same algorithm can be applied for both.
func (crypt *PCCrypt) process(data []byte, size int) {
	for x := 0; x < size; x += 4 {
		tmp := le(data[x : x+4])
		tmp ^= crypt.getNextKey()
		// Stick the data back in LE order.
		data[x] = byte(tmp)
		data[x+1] = byte(tmp >> 8)
		data[x+2] = byte(tmp >> 16)
		data[x+3] = byte(tmp >> 24)
	}
}
