package encryption

//#include "encryption.h"
import "C"

/* Wrapper library for Fuzzier's encryption lib. */
import "unsafe"

// Returns a newly allocated C CRYPT_SETUP
func NewCrypt() *C.CRYPT_SETUP {
	return new(C.CRYPT_SETUP)
}

// Initializes a CRYPT_SETUP with a 48-byte key.
func CreateKeys(crypt_setup *C.CRYPT_SETUP, keys []byte) int {
	return int(C.CRYPT_CreateKeys(crypt_setup, unsafe.Pointer(&keys[0]), C.CRYPT_BLUEBURST))
}

// Encrypt or decrypt the packet pointed to by data in-place using crypt_setup.
func CryptData(crypt_setup *C.CRYPT_SETUP, data []byte, size uint32, encrypting bool) int {
	var encr C.int
	if encrypting {
		encr = 1
	} else {
		encr = 0
	}
	return int(C.CRYPT_CryptData(crypt_setup, unsafe.Pointer(&data[0]), C.ulong(size), encr))
}
