package prs

//#import "prs.h"
import "C"
import "unsafe"

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
* PRS compression/decompression library. Original underlying C implementation
* by Fuzziqer Software, wrapper written in Go for use with archon.
 */

func Compress(src, dest []byte, size uint32) uint32 {
	return uint32(C.prs_compress(
		unsafe.Pointer(&src[0]), unsafe.Pointer(&dest[0]), C.u32(size)))
}

func Decompress(src, dest []byte) uint32 {
	return uint32(C.prs_decompress(
		unsafe.Pointer(&src[0]), unsafe.Pointer(&dest[0])))
}

func DecompressSize(src []byte) uint32 {
	return uint32(C.prs_decompress_size(unsafe.Pointer(&src[0])))
}
