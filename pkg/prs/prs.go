package prs

//#import "prs.h"
import "C"
import "unsafe"

// PRS compression/decompression library. Original underlying C implementation
// by Fuzziqer Software, wrapper written in Go for use with archon.

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
