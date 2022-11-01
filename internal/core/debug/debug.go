package debug

import (
	"bufio"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"strings"

	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/sirupsen/logrus"
)

// This function starts the default pprof HTTP server that can be accessed via localhost
// to get runtime information about archon. See https://golang.org/pkg/net/http/pprof/
func StartPprofServer(logger *logrus.Logger, pprofPort int) {
	listenerAddr := fmt.Sprintf("localhost:%d", pprofPort)
	logger.Infof("starting pprof server on %s", listenerAddr)

	go func() {
		if err := http.ListenAndServe(listenerAddr, nil); err != nil {
			logger.Infof("error starting pprof server: %s", err)
		}
	}()
}

// Used with Clients to attach debugging information.
type Tag string

var (
	SERVER_TYPE = "server_type"
)

type ServerType string

const (
	UNKNOWN          = "?"
	PATCH_SERVER     = "PATCH"
	DATA_SERVER      = "DATA"
	LOGIN_SERVER     = "LOGIN"
	CHARACTER_SERVER = "CHARACTER"
	SHIP_SERVER      = "SHIP"
	BLOCK_SERVER     = "BLOCK"
)

type PrintPacketParams struct {
	Writer       *bufio.Writer
	ServerType   ServerType
	ClientPacket bool
	Data         []byte
}

// PrintPacket prints the contents of a packet to a specified writer along with some
// inferred metadata about the packet itself.
func PrintPacket(params PrintPacketParams) {
	var header packets.PCHeader
	bytes.StructFromBytes(params.Data[:packets.PCHeaderSize], &header)

	// Write a header line for each packet with some metadata.
	var headerLine strings.Builder
	headerLine.WriteString(fmt.Sprintf("[%s] 0x%02X ", params.ServerType, header.Type))
	headerLine.WriteString(fmt.Sprintf("(%s) ", getPacketName(params.ServerType, header.Type)))
	if params.ClientPacket {
		headerLine.WriteString("| client->server ")
	} else {
		headerLine.WriteString("| server->client ")
	}
	headerLine.WriteString(fmt.Sprintf("(%d bytes)\n", header.Size))

	// Write out the contents of the actual packet.
	if _, err := params.Writer.WriteString(headerLine.String()); err != nil {
		fmt.Printf("error writing packet header: %v\n", err)
		return
	}
	if err := writePacketBodyToFile(params.Writer, params.Data); err != nil {
		fmt.Printf("error writing packet body: %v\n", err)
		return
	}
	_, _ = params.Writer.WriteString("\n")
	params.Writer.Flush()
}

const PacketLineLength = 16

func writePacketBodyToFile(writer *bufio.Writer, data []byte) error {
	pktLen := len(data)
	for rem, offset := pktLen, 0; rem > 0; rem -= PacketLineLength {
		var line string

		if rem < PacketLineLength {
			line = buildPacketLine(data[(pktLen-rem):pktLen], rem, offset)
		} else {
			line = buildPacketLine(data[offset:offset+PacketLineLength], PacketLineLength, offset)
		}
		offset += PacketLineLength

		if _, err := writer.WriteString(line); err != nil {
			return err
		}
	}
	return nil
}

// Build one line of formatted packet data.
func buildPacketLine(data []uint8, length int, offset int) string {
	var line strings.Builder

	line.WriteString(fmt.Sprintf("(%04X) ", offset))

	for i, j := 0, 0; i < length; i++ {
		if j == 8 {
			// Visual aid - spacing between groups of 8 bytes.
			j = 0
			// line.WriteString("  ")
		}
		line.WriteString(fmt.Sprintf("%02x ", data[i]))
		j++
	}

	// Fill in rest of the line gap if we don't have enough bytes.
	for i := length; i < PacketLineLength; i++ {
		if i == 8 {
			line.WriteString("  ")
		}
		line.WriteString("   ")
	}
	line.WriteString("    ")

	// Display the print characters as-is, others as periods.
	for i := 0; i < length; i++ {
		c := data[i]

		if strconv.IsPrint(rune(c)) {
			line.WriteString(string(data[i]))
		} else {
			line.WriteString(".")
		}
	}

	line.WriteString("\n")
	return line.String()
}
