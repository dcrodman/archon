/*
* Archon PSO Server
* Copyright (C) 2014 Andrew Rodman
*
* This program is free software: you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License
* along with this program.  If not, see <http://www.gnu.org/licenses/>.
* ---------------------------------------------------------------------
*
* Blowfish implementation adapted to work with PSOBB's protocol.
 */
package encryption

import (
	"crypto/rand"
	"encoding/binary"
)

// Internal representation of a cipher capable of performing
// encryption and decryption on blocks.
type psoCipher interface {
	encrypt(data []byte)
	decrypt(data []byte)
	blockSize() int
}

// PSOCrypt object to be used per-client for crypto.
type PSOCrypt struct {
	cipher psoCipher
	Vector []uint8
}

// Generate a cryptographially secure random string of bytes.
func createKey(size int) []byte {
	key := make([]byte, size)
	for i := 0; i < size; i++ {
		binary.Read(rand.Reader, binary.LittleEndian, &key[i])
	}
	return key
}

// Condense four bytes into a LE 32-bit value.
func le(b []byte) uint32 {
	for i := 3; len(b) < 4; i-- {
		b = append(b, 0)
	}
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// Returns a newly allocated PSOCrypt with randomly generated, appropriately
// sized keys for encrypting packets over PSOPC connections.
func NewPCCrypt() *PSOCrypt {
	crypt := &PSOCrypt{Vector: createKey(4)}
	var err error
	if crypt.cipher, err = newPCCipher(crypt.Vector); err != nil {
		panic(err)
	}
	return crypt
}

// Returns a newly allocated PSOCrypt with randomly generated, appropriately
// sized keys for encrypting packets over PSOBB connections.
func NewBBCrypt() *PSOCrypt {
	crypt := &PSOCrypt{Vector: createKey(48)}
	var err error
	if crypt.cipher, err = newCipher(crypt.Vector); err != nil {
		panic(err)
	}
	return crypt
}

// Encrypt a block of data in place.
func (crypt *PSOCrypt) Encrypt(data []byte, size uint32) {
	blockSize := crypt.cipher.blockSize()
	for i := 0; i < int(size); i += blockSize {
		block := data[i : i+blockSize]
		crypt.cipher.encrypt(block)
	}
}

// Decrypt a block of data in place.
func (crypt *PSOCrypt) Decrypt(data []byte, size uint32) {
	blockSize := crypt.cipher.blockSize()
	for i := 0; i < int(size); i += blockSize {
		block := data[i : i+blockSize]
		crypt.cipher.decrypt(block)
	}
}
