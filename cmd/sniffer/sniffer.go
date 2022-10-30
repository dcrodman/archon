package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/encryption"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/google/gopacket"
)

const PacketLineLength = 16

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

// Best effort guess as to what ports correspond to which servers based on
// the defaults for the servers this tool will likely be used with.
var serverPorts = map[uint16]ServerType{
	11000: PATCH_SERVER,
	11001: DATA_SERVER,
	12000: LOGIN_SERVER,
	12001: CHARACTER_SERVER,
	// Archon's ports.
	15000: SHIP_SERVER,
	15001: BLOCK_SERVER,
	// Tethealla's ports.
	5278: SHIP_SERVER,
	5279: BLOCK_SERVER,
	5280: BLOCK_SERVER,
}

type CipherPair struct {
	clientCrypt *encryption.PSOCrypt
	serverCrypt *encryption.PSOCrypt
}

type sniffer struct {
	Writer *bufio.Writer

	ciphers map[ServerType]CipherPair
}

func (s *sniffer) startReading(packetChan chan gopacket.Packet) {
	s.ciphers = make(map[ServerType]CipherPair)

	for packet := range packetChan {
		flow := packet.TransportLayer().TransportFlow()
		srcPort := binary.BigEndian.Uint16(flow.Src().Raw())
		dstPort := binary.BigEndian.Uint16(flow.Dst().Raw())
		data := packet.ApplicationLayer().Payload()

		clientPacket, server := getPacketType(srcPort, dstPort)
		s.handlePacket(server, clientPacket, data)
	}
}

// Guesses the server type based on the sender of the packet and what ports the
// packet was sent between. Also returns whether or not this packet was sent
// by the client.
func getPacketType(srcPort, dstPort uint16) (bool, ServerType) {
	if server, ok := serverPorts[dstPort]; ok {
		return true, server
	}
	if server, ok := serverPorts[srcPort]; ok {
		return false, server
	}
	return false, UNKNOWN
}

func (s *sniffer) handlePacket(server ServerType, clientPacket bool, data []byte) {
	// Peek at the header
	var header packets.PCHeader
	bytes.StructFromBytes(data[:packets.PCHeaderSize], &header)

	// Any time we see a welcome packet, initialize a new set of ciphers for the corresponding server.
	switch header.Type {
	case packets.PatchWelcomeType:
		var welcomePacket packets.PatchWelcome
		bytes.StructFromBytes(data, &welcomePacket)
		s.ciphers[server] = CipherPair{
			clientCrypt: encryption.NewPCCryptWithVector(welcomePacket.ClientVector[:]),
			serverCrypt: encryption.NewPCCryptWithVector(welcomePacket.ServerVector[:]),
		}
	case packets.LoginWelcomeType:
		var welcomePacket packets.Welcome
		bytes.StructFromBytes(data, &welcomePacket)
		s.ciphers[server] = CipherPair{
			clientCrypt: encryption.NewBBCryptWithVector(welcomePacket.ClientVector[:]),
			serverCrypt: encryption.NewBBCryptWithVector(welcomePacket.ServerVector[:]),
		}
	default:
		// Anything else is meaningless since it's encrypted, so decrypt it
		// and read the hader again.
		if clientPacket {
			// Decrypt the packet with our client cipher.
			s.ciphers[server].clientCrypt.Decrypt(data, uint32(len(data)))
		} else {
			// Decrypt the packet with our server cipher.
			s.ciphers[server].serverCrypt.Decrypt(data, uint32(len(data)))
		}
		bytes.StructFromBytes(data[:packets.PCHeaderSize], &header)
	}

	// Write a header line for each packet with some metadata.
	var headerLine strings.Builder
	headerLine.WriteString(fmt.Sprintf("[%s] 0x%02X ", string(server), header.Type))
	headerLine.WriteString(fmt.Sprintf("(%s) ", getPacketName(server, header.Type)))
	if clientPacket {
		headerLine.WriteString("| client->server ")
	} else {
		headerLine.WriteString("| server->client ")
	}
	headerLine.WriteString(fmt.Sprintf("(%d bytes)\n", header.Size))

	// Write out the contents of the actual packet.
	if _, err := s.Writer.WriteString(headerLine.String()); err != nil {
		fmt.Printf("error writing packet header to file: %v\n", err)
		return
	}
	if err := s.writePacketBodyToFile(data); err != nil {
		fmt.Printf("error writing packet body to file: %v\n", err)
		return
	}
	s.Writer.WriteString("\n")
	s.Writer.Flush()
}

func (s *sniffer) writePacketBodyToFile(data []byte) error {
	pktLen := len(data)
	for rem, offset := pktLen, 0; rem > 0; rem -= PacketLineLength {
		var line string

		if rem < PacketLineLength {
			line = buildPacketLine(data[(pktLen-rem):pktLen], rem, offset)
		} else {
			line = buildPacketLine(data[offset:offset+PacketLineLength], PacketLineLength, offset)
		}
		offset += PacketLineLength

		if _, err := s.Writer.WriteString(line); err != nil {
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
