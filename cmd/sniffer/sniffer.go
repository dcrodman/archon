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

type sniffer struct {
	Writer *bufio.Writer

	patchClientCrypt *encryption.PSOCrypt
	patchServerCrypt *encryption.PSOCrypt

	clientCrypt *encryption.PSOCrypt
	serverCrypt *encryption.PSOCrypt
}

func (s *sniffer) startIngesting(packetChan chan gopacket.Packet) {
	// for packet := range packetChan {
	for i := 0; i < 10; i++ {
		packet := <-packetChan

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
	fmt.Printf("server: %v\n", server)
	decrypt := true
	// Make sure we've initialized our encryption vectors.
	if (server == PATCH_SERVER || server == DATA_SERVER) && s.patchClientCrypt == nil {
		var welcomePacket packets.PatchWelcome
		bytes.StructFromBytes(data, &welcomePacket)
		s.patchClientCrypt = encryption.NewPCCryptWithVector(welcomePacket.ClientVector[:])
		s.patchServerCrypt = encryption.NewPCCryptWithVector(welcomePacket.ServerVector[:])
		decrypt = false
	} else if (server != PATCH_SERVER && server != DATA_SERVER) && s.clientCrypt == nil {
		var welcomePacket packets.Welcome
		bytes.StructFromBytes(data, &welcomePacket)
		s.clientCrypt = encryption.NewBBCryptWithVector(welcomePacket.ClientVector[:])
		s.serverCrypt = encryption.NewBBCryptWithVector(welcomePacket.ServerVector[:])
		decrypt = false
	}

	if decrypt {
		s.decryptData(server, clientPacket, data)
	}
	// Extract the header so that we know what type and how large it is.
	var header packets.PCHeader
	bytes.StructFromBytes(data[:packets.PCHeaderSize], &header)

	// Write a header line for each packet with some metadata.
	var headerLine strings.Builder
	headerLine.WriteString(fmt.Sprintf("[%s] 0x%02x ", string(server), header.Type))
	headerLine.WriteString(fmt.Sprintf("(?) "))
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

func (s *sniffer) decryptData(server ServerType, clientPacket bool, data []byte) {
	if server == PATCH_SERVER || server == DATA_SERVER {
		if clientPacket {
			// Decrypt the packet with our client cipher.
			s.patchClientCrypt.Decrypt(data, uint32(len(data)))
		} else {
			// Decrypt the packet with our server cipher.
			s.patchServerCrypt.Decrypt(data, uint32(len(data)))
		}
	} else {
		if clientPacket {
			// Decrypt the packet with our client cipher.
			s.clientCrypt.Decrypt(data, uint32(len(data)))
		} else {
			// Decrypt the packet with our server cipher.
			s.serverCrypt.Decrypt(data, uint32(len(data)))
		}
	}
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
