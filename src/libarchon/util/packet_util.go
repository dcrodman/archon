// Packet operations common among the various servers.
package util

import (
	"bytes"
	"encoding/binary"
)

// Extract the packet length from the first two bytes of data.
func GetPacketSize(data []byte) (uint16, error) {
	if len(data) < 2 {
		return 0, &ServerError{message: "getSize(): data must be at least two bytes."}
	}
	var size uint16
	reader := bytes.NewReader(data)
	binary.Read(reader, binary.LittleEndian, &size)
	return size, nil
}
