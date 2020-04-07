package relay

import (
	"fmt"
	"github.com/dcrodman/archon/server"
	"github.com/dcrodman/archon/server/internal"
)

// send converts a packet struct to bytes and encrypts it before  using the
// server's session key before sending the data to the client.
func SendPacket(c server.Client2, packet interface{}, lenDivisor uint16) error {
	data, size := internal.BytesFromStruct(packet)
	bytes, n := adjustPacketLength(data, uint16(size), lenDivisor)

	c.Encrypt(bytes, uint32(n))
	return SendRaw(c, bytes, n)
}

// adjustPacketLength pads the length of a packet to a multiple of the header
// length and adjusts first two bytes of the header to the corrected size
// (may be a no-op). Note: PSOBB clients will reject packets that are not
// padded in this manner.
func adjustPacketLength(data []byte, length uint16, divisor uint16) ([]byte, uint16) {
	for length%divisor != 0 {
		length++
		data = append(data, 0)
	}

	data[0] = byte(length & 0xFF)
	data[1] = byte((length & 0xFF00) >> 8)

	return data, length
}

// sendRaw writes all data contained in the slice to the client as-is.
func SendRaw(c server.Client2, data []byte, length uint16) error {
	bytesSent := 0

	for bytesSent < int(length) {
		b, err := c.ConnectionState().WriteBytes(data[:length])
		if err != nil {
			return fmt.Errorf("failed to send to client %v: %s", c.ConnectionState().IPAddr(), err.Error())
		}
		bytesSent += b
	}

	return nil
}
