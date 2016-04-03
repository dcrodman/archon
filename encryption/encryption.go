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
* Wrapper library for Fuzzier's encryption lib.
 */

import (
	"crypto/rand"
	"encoding/binary"
	"github.com/dcrodman/archon/blowfish"
	"unsafe"
)

type PSOCrypt struct {
	cryptSetup C.CRYPT_SETUP
	cipher     *blowfish.Cipher
	Vector     []uint8
}

// Returns a newly allocated and zeroed PSOCrypt.
func NewCrypt(vectorSize int) *PSOCrypt {
	crypt := new(PSOCrypt)
	crypt.Vector = make([]uint8, vectorSize)
	for i := 0; i < vectorSize; i++ {
		binary.Read(rand.Reader, binary.LittleEndian, &(crypt.Vector[i]))
	}
	if vectorSize == 4 {
		C.CRYPT_CreateKeys(&crypt.cryptSetup, unsafe.Pointer(&crypt.Vector[0]), C.CRYPT_PC)
	} else {
		var err error
		if crypt.cipher, err = blowfish.NewCipher(crypt.Vector); err != nil {
			panic(err)
		}
	}
	return crypt
}

// Convenience wrapper for CryptData with encrypting = 1.
func (crypt *PSOCrypt) Encrypt(data []byte, size uint32) {
	if crypt.cipher != nil {
		crypt.cipher.Encrypt(data, data)
	} else {
		CryptData(crypt, data, size, C.int(1))
	}
}

// Convenience wrapper for CryptData with encrypting = 0.
func (crypt *PSOCrypt) Decrypt(data []byte, size uint32) {
	if crypt.cipher != nil {
		crypt.cipher.Decrypt(data, data)
	} else {
		CryptData(crypt, data, size, C.int(0))
	}
}

func CryptData(crypt *PSOCrypt, data []byte, size uint32, encrypting C.int) int {
	return int(C.CRYPT_CryptData(&crypt.cryptSetup,
		unsafe.Pointer(&data[0]), C.ulong(size), encrypting))
}
