// Most of this code is copied from the golang.org/x/crypto/blowfish package.
//
// Source modified to work with the customized PSOBB Blowfish implementation,
// which uses fewer rounds. Work based off of the encryption library written
// by Fuzziqer Software (http://www.fuzziqersoftware.com/).
package encryption

import "fmt"

// The Blowfish block size in bytes.
const bbBlockSize = 8

// A blowfishCipher is an instance of Blowfish encryption using a particular key.
type blowfishCipher struct {
	p              [18]uint32
	s0, s1, s2, s3 [256]uint32
}

// NewCipher creates and returns a blowfishCipher. The key argument should be
// the Blowfish key such that 1 <= len(k) <= 56 bytes.
func newCipher(key []byte) (psoCipher, error) {
	var result blowfishCipher
	if k := len(key); k < 1 || k > 56 {
		return nil, fmt.Errorf("crypto/blowfish: invalid key size %d", k)
	}

	initCipher(&result)
	expandKey(key, &result)
	return &result, nil
}

func (c *blowfishCipher) blockSize() int {
	return bbBlockSize
}

// Encrypt encrypts the 8-byte buffer src using the key k
// and stores the result in dst.
// Note that for amounts of data larger than a block,
// it is not safe to just call Encrypt on successive blocks;
// instead, use an encryption mode like CBC (see crypto/cipher/cbc.go).
func (c *blowfishCipher) encrypt(src []byte) {
	l := toLittleEndian(src[0:4])
	r := toLittleEndian(src[4:8])
	l, r = encryptData(l, r, c)
	src[0], src[1], src[2], src[3] = byte(l), byte(l>>8), byte(l>>16), byte(l>>24)
	src[4], src[5], src[6], src[7] = byte(r), byte(r>>8), byte(r>>16), byte(r>>24)
}

// Decrypt decrypts the 8-byte buffer src using the key k
// and stores the result in dst.
func (c *blowfishCipher) decrypt(src []byte) {
	l := toLittleEndian(src[0:4])
	r := toLittleEndian(src[4:8])
	l, r = decryptData(l, r, c)
	src[0], src[1], src[2], src[3] = byte(l), byte(l>>8), byte(l>>16), byte(l>>24)
	src[4], src[5], src[6], src[7] = byte(r), byte(r>>8), byte(r>>16), byte(r>>24)
}
