// https://github.com/Sewer56/dlang-prs
package prs

type decompressor struct {
	// bitPos is the position we are reading from the controlByte.
	bitPos      int
	controlByte byte

	srcPos int
	src    []byte

	dst []byte

	// dstSize is incremented every time dst would be added to
	dstSize int

	// test flag - disable to not copy
	copy bool
}

// newDecompressor is a type built to support decompressing a PRS file.
// The controlByte starts at the first byte, and bitPos starts at 7,
// indicating we can shift the controlByte 8 times before we need a new one.
//
// The srcPos starts at 1 because we exclude the first control byte.
func newDecompressor(src []byte, size int, copy bool) *decompressor {
	return &decompressor{
		controlByte: src[0],
		bitPos:      8,
		src:         src,
		srcPos:      1,
		copy:        copy,
		dst:         make([]byte, 0, size),
	}
}

// decompress expands a PRS compressed file
func (d *decompressor) decompress() ([]byte, error) {
	for i := 0; ; i++ {
		if d.getNextBit() == 1 {
			d.copyCurrentByte()
			continue
		}
		if d.getNextBit() == 1 {
			offset := int(d.getNextByte()) | (int(d.getNextByte()) << 8)
			if offset == 0 {
				return d.dst, nil
			}

			length := (offset & 0b111) + 2

			offset = (offset >> 3) | -0x2000

			if length == 2 {
				length = int(d.getNextByte()) + 1
			}
			for i := 0; i < length; i++ {
				d.copyFromOffset(offset)
			}
		} else {
			// Length is encoded using 2 bits so the length will be between 0 and 3.
			// When it is encoded, 2 is subtracted from the length so the actual
			// length will be between 2 and 5 inclusive.
			length := int((d.getNextBit()<<1)|d.getNextBit()) + 2

			// The offset is encoded in the next byte, as 256 - positive offset.
			// ex: offset of 5
			// 256 - (-5 * -1) = 251
			// We'll decode that by:
			// 256 - 251 = 5
			// 5 * -1 = -5
			offset := int(d.getNextByte()) | -0x100
			for i := 0; i < length; i++ {
				d.copyFromOffset(offset)
			}
		}
	}
}

// getNextBit gets the next bit from the controlByte. If the controlByte has been
// exhausted (eg the bitPos is -1), then getNextBit will get the next controlByte
// from src before returning the next bit.
func (d *decompressor) getNextBit() byte {
	if d.bitPos == 0 {
		// read another byte
		d.controlByte = d.getNextByte()
		// max out the control byte position
		d.bitPos = 8
	}
	b := d.controlByte >> (8 - d.bitPos) & 1
	d.bitPos--
	return b
}

func (d *decompressor) getNextByte() byte {
	defer func() { d.srcPos++ }()
	return d.src[d.srcPos]
}

func (d *decompressor) copyCurrentByte() {
	if !d.copy {
		d.dstSize++
		return
	}
	d.dst = append(d.dst, d.src[d.srcPos])
	d.srcPos++
}

func (d *decompressor) copyFromOffset(offset int) {
	if !d.copy {
		d.dstSize++
		return
	}
	d.dst = append(d.dst, d.dst[len(d.dst)+offset])
}
