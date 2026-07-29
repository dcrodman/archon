package prs

func Decompress(src []byte, size int) ([]byte, error) {
	d := newDecompressor(src, size, true)
	return d.decompress()
}

func DecompressSize(src []byte) (int, error) {
	d := newDecompressor(src, 0, false)
	if _, err := d.decompress(); err != nil {
		return 0, err
	}
	return d.dstSize, nil
}
