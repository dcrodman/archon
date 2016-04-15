package encryption

//#include "encryption.h"
import "C"

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

import (
	"crypto/rand"
	"encoding/binary"
	"unsafe"
)

type PSOCrypt struct {
	cipher     *Cipher
	cryptSetup C.CRYPT_SETUP
	Vector     []uint8
}

func createKey(size int) []byte {
	key := make([]byte, size)
	for i := 0; i < size; i++ {
		binary.Read(rand.Reader, binary.LittleEndian, &key[i])
	}
	return key
}

// Returns a newly allocated PSOCrypt with randomly generated, appropriately
// sized keys for encrypting packets over PSOPC connections.
func NewPCCrypt() *PSOCrypt {
	crypt := &PSOCrypt{Vector: createKey(4)}
	C.CRYPT_CreateKeys(&crypt.cryptSetup, unsafe.Pointer(&crypt.Vector[0]), C.CRYPT_PC)
	return crypt
}

// Returns a newly allocated PSOCrypt with randomly generated, appropriately
// sized keys for encrypting packets over PSOBB connections.
func NewBBCrypt() *PSOCrypt {
	crypt := &PSOCrypt{Vector: createKey(48)}
	var err error
	if crypt.cipher, err = NewCipher(crypt.Vector); err != nil {
		panic(err)
	}
	C.CRYPT_CreateKeys(&crypt.cryptSetup, unsafe.Pointer(&crypt.Vector[0]), C.CRYPT_BLUEBURST)
	return crypt
}

// Encrypt a block of data in place.
func (crypt *PSOCrypt) Encrypt(data []byte, size uint32) {
	if crypt.cipher != nil {
		for i := 0; i < int(size); i += BlockSize {
			block := data[i : i+BlockSize]
			crypt.cipher.Encrypt(block, block)
		}
	} else {
		C.CRYPT_CryptData(&crypt.cryptSetup, unsafe.Pointer(&data[0]), C.ulong(size), C.int(1))
	}
}

// Decrypt a block of data in place.
func (crypt *PSOCrypt) Decrypt(data []byte, size uint32) {
	if crypt.cipher != nil {
		for i := 0; i < int(size); i += BlockSize {
			block := data[i : i+BlockSize]
			crypt.cipher.Decrypt(block, block)
		}
	} else {
		C.CRYPT_CryptData(&crypt.cryptSetup, unsafe.Pointer(&data[0]), C.ulong(size), C.int(0))
	}
}
