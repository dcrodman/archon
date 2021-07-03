package prs
//
//type compressor struct {
//	// bitPos is the position we are reading from the controlByte.
//	bitPos           int
//	controlByte      byte
//	controlByteIndex int
//
//	srcPos int
//	src    []byte
//
//	dstPos int
//	dst    []byte
//}
//
//func newCompressor(src []byte) *compressor {
//	return &compressor{
//		bitPos:           0,
//		controlByte:      0,
//		controlByteIndex: 0,
//		srcPos:           0,
//		src:              src,
//		dstPos:           0,
//		dst:              make([]byte, len(src)*2),
//	}
//}
//
//func (c *compressor) compress() ([]byte, error) {
//}
//
//func (c *compressor) setControlBit(bit int) {
//	if c.bitPos == 0 {
//		c.dst[c.controlByteIndex] = c.controlByte
//		c.controlByteIndex = c.dstPos
//		c.dstPos++
//
//		c.controlByte = 0
//		c.bitPos = 7
//	}
//	c.bitPos--
//	c.controlByte >>= 1
//	if bit != 0 {
//		// set the most significant bit, we'll right shift later if we need to.
//		c.controlByte |= 128
//	}
//}
//
//func (c *compressor) copyLiteral() {
//	c.dst[c.dstPos] = c.src[c.srcPos]
//	c.dstPos++
//	c.srcPos++
//}
//
//func (c *compressor) shortCopy(offset, length int) {
//	// Offset the size as required for this mode (pack 2-5 as 0-3)
//	length -= 2
//
//	// Write opcode 00.
//	c.setControlBit(0)
//	c.setControlBit(0)
//
//	// Pack the size with the second byte first.
//	c.setControlBit(length & 2)
//	c.setControlBit(length & 1)
//	// Write the offset as 256 - (offset * - 1) as required by the format.
//	c.writeLiteral(byte(offset & 0xFF))
//}
//
//func (c *compressor) longCopySmall(offset, length int) {
//	// Offset the size as required for this mode (pack 3-9 as 1-7)
//	length -= 2
//
//	// Write opcode 01.
//	c.setControlBit(0)
//	c.setControlBit(1)
//
//	short := offset<<3&0xFFF8 | length
//	c.writeLiteral(byte(short))
//	c.writeLiteral(byte(short >> 8))
//}
//
//func (c *compressor) longCopyLarge(offset, length int) {
//	// Offset the size as required for this mode.
//	length -= 1
//
//	// Write opcode 01.
//	c.setControlBit(0)
//	c.setControlBit(1)
//
//	short := offset << 3 & 0xFFF8
//
//	// Write the packed size and offset in Big Endian
//	c.writeLiteral(byte(short))
//	c.writeLiteral(byte(short >> 8))
//
//	// Write the offset.
//	c.writeLiteral(byte(length))
//}
//
//func (c *compressor) writeLiteral(value byte) {
//	c.dst[c.dstPos] = value
//	c.dstPos++
//}
//
//func (c *compressor) writeFinalFlags() {
//	c.controlByte >>= c.bitPos
//	c.dst[c.controlByteIndex] = c.controlByte
//}
//
//func (c *compressor) WriteEOF() {
//	c.setControlBit(0)
//	c.setControlBit(1)
//	c.writeFinalFlags()
//	c.writeLiteral(0)
//	c.writeLiteral(0)
//}
