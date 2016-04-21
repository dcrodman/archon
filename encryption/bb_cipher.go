/* Copyright 2010 The Go Authors. All rights reserved.
* Use of this source code is governed by a BSD-style
* license that can be found in the LICENSE file.
*
* The code is a port of Bruce Schneier's C implementation.
* See http://www.schneier.com/blowfish.html.
*
* Source modified by Andrew Rodman to work with the customized
* PSOBB Blowfish implementation. Work based off of the encryption
* library written by Fuzziqer Software.
 */

package encryption

import "strconv"

// The Blowfish block size in bytes.
const BlockSize = 8

// A Cipher is an instance of Blowfish encryption using a particular key.
type Cipher struct {
	p              [18]uint32
	s0, s1, s2, s3 [256]uint32
}

type KeySizeError int

func (k KeySizeError) Error() string {
	return "crypto/blowfish: invalid key size " + strconv.Itoa(int(k))
}

// NewCipher creates and returns a Cipher.
// The key argument should be the Blowfish key, from 1 to 56 bytes.
func newCipher(key []byte) (psoCipher, error) {
	var result Cipher
	if k := len(key); k < 1 || k > 56 {
		return nil, KeySizeError(k)
	}
	initCipher(&result)
	expandKey(key, &result)
	return &result, nil
}

// BlockSize returns the Blowfish block size, 8 bytes.
// It is necessary to satisfy the Block interface in the
// package "crypto/cipher".
func (c *Cipher) blockSize() int { return BlockSize }

// Encrypt encrypts the 8-byte buffer src using the key k
// and stores the result in dst.
// Note that for amounts of data larger than a block,
// it is not safe to just call Encrypt on successive blocks;
// instead, use an encryption mode like CBC (see crypto/cipher/cbc.go).
func (c *Cipher) encrypt(src []byte) {
	l := le(src[0:4])
	r := le(src[4:8])
	l, r = encryptData(l, r, c)
	src[0], src[1], src[2], src[3] = byte(l), byte(l>>8), byte(l>>16), byte(l>>24)
	src[4], src[5], src[6], src[7] = byte(r), byte(r>>8), byte(r>>16), byte(r>>24)
}

// Decrypt decrypts the 8-byte buffer src using the key k
// and stores the result in dst.
func (c *Cipher) decrypt(src []byte) {
	l := le(src[0:4])
	r := le(src[4:8])
	l, r = decryptData(l, r, c)
	src[0], src[1], src[2], src[3] = byte(l), byte(l>>8), byte(l>>16), byte(l>>24)
	src[4], src[5], src[6], src[7] = byte(r), byte(r>>8), byte(r>>16), byte(r>>24)
}

func initCipher(c *Cipher) {
	copy(c.p[0:], p[0:])
	copy(c.s0[0:], s0[0:])
	copy(c.s1[0:], s1[0:])
	copy(c.s2[0:], s2[0:])
	copy(c.s3[0:], s3[0:])
}
