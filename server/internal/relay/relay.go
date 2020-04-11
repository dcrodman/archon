package relay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/debug"
	"github.com/dcrodman/archon/server"
	"github.com/dcrodman/archon/server/internal"
	"github.com/spf13/viper"
	"net/http"
)

// send converts a packet struct to bytes and encrypts it before  using the
// server's session key before sending the data to the client.
func SendPacket(c server.Client2, packet interface{}, lenDivisor uint16) error {
	data, size := internal.BytesFromStruct(packet)
	b, n := adjustPacketLength(data, uint16(size), lenDivisor)

	sendPacketToAnalyzer(b)

	c.Encrypt(b, uint32(n))
	return SendRaw(c, b, n)
}

// Ship the packet we're about to send over to the packet_analyzer server
// provided the server is in debug mode.
func sendPacketToAnalyzer(packetBytes []byte) {
	if !(debug.Enabled() && viper.IsSet("packet_analyzer_address")) {
		return
	}

	cbytes := make([]int, len(packetBytes))
	for i := 0; i < len(packetBytes); i++ {
		cbytes[i] = int(packetBytes[i])
	}

	packet := struct {
		ServerName  string
		SessionID   string
		Source      string
		Destination string
		Contents    []int
	}{
		"Archon", "", "server", "client", cbytes,
	}

	reqBytes, _ := json.Marshal(&packet)

	// We don't care if the packets don't get through.
	_, err := http.Post(
		"http://"+viper.GetString("packet_analyzer_address"),
		"application/json",
		bytes.NewBuffer(reqBytes),
	)

	if err != nil {
		archon.Log.Warn("failed to send packet to analyzer:", err)
	}
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

// SendRaw writes all data contained in the slice to the client
// as-is (e.g. without encrypting it first).
func SendRaw(c server.Client2, data []byte, length uint16) error {
	sendPacketToAnalyzer(data)
	return transmit(c, data, length)
}

func transmit(c server.Client2, data []byte, length uint16) error {
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
