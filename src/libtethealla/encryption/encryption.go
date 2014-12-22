package encryption

//#include "encryption.h"
import "C"

/* Wrapper library for Fuzzier's encryption lib. */

import (
	"crypto/rand"
	"encoding/binary"
	"unsafe"
)

type PSOCrypt struct {
	cryptSetup C.CRYPT_SETUP
	Vector     []uint8
}

// Initializes a CRYPT_SETUP with a 48-byte key.
func (crypt *PSOCrypt) CreateKeys() int {
	crypt.Vector = make([]uint8, 48)
	for i := 0; i < 48; i++ {
		binary.Read(rand.Reader, binary.LittleEndian, &(crypt.Vector[i]))
	}
	return int(C.CRYPT_CreateKeys(&crypt.cryptSetup, unsafe.Pointer(&crypt.Vector[0]), C.CRYPT_BLUEBURST))
}

// Convenience wrapper for CryptData with encrypting = 1.
func (crypt *PSOCrypt) Encrypt(data []byte, size uint32) {
	CryptData(crypt, data, size, C.int(1))
}

// Convenience wrapper for CryptData with encrypting = 0.
func (crypt *PSOCrypt) Decrypt(data []byte, size uint32) {
	CryptData(crypt, data, size, C.int(0))
}

// Returns a newly allocated and zeroed PSOCrypt.
func NewCrypt() *PSOCrypt {
	return new(PSOCrypt)
}

// Encrypt or decrypt the packet pointed to by data in-place using crypt_setup.
func CryptData(crypt *PSOCrypt, data []byte, size uint32, encrypting C.int) int {
	return int(C.CRYPT_CryptData(&crypt.cryptSetup, unsafe.Pointer(&data[0]), C.ulong(size), encrypting))
}
