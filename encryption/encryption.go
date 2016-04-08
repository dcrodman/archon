package encryption

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

type PSOCrypt struct {
	cipher *Cipher
	Vector []uint8
}

// Returns a newly allocated and zeroed PSOCrypt.
func NewCrypt(key []byte) *PSOCrypt {
	crypt := new(PSOCrypt)
	crypt.Vector = key
	var err error
	if crypt.cipher, err = NewCipher(crypt.Vector); err != nil {
		panic(err)
	}
	return crypt
}

// Convenience wrapper for CryptData with encrypting = 1.
func (crypt *PSOCrypt) Encrypt(data []byte, size uint32) []byte {
	dst := make([]byte, size)
	crypt.cipher.Encrypt(dst, data)
	return dst
}

// Convenience wrapper for CryptData with encrypting = 0.
func (crypt *PSOCrypt) Decrypt(data []byte, size uint32) []byte {
	dst := make([]byte, size)
	crypt.cipher.Decrypt(dst, data)
	return dst
}
