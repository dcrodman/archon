package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/google/gopacket"

	packets "github.com/dcrodman/archon/internal/commands"
	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/debug"
	"github.com/dcrodman/archon/internal/core/encryption"
)

// Best effort guess as to what ports correspond to which servers based on
// the defaults for the servers this tool will likely be used with.
var serverPorts = map[uint16]string{
	11000: "PATCH:AUTH",
	11001: "PATCH:DATA}",
	12000: "CHARACTER:AUTH",
	12001: "CHARACTER:DATA",
	// Archon's ports.
	15000: "SHIP:AUTH",
	15001: "SHIP:GAME",
	// Tethealla's ports.
	5278: "SHIP",
	5279: "BLOCK01",
	5280: "BLOCK01",
}

type CipherPair struct {
	clientCrypt *encryption.PSOCrypt
	serverCrypt *encryption.PSOCrypt
}

type sniffer struct {
	Writer *bufio.Writer

	ciphers           map[string]CipherPair
	currentPacketSize uint16
	bufferBytesRead   uint16
	buffer            []byte
}

func (s *sniffer) startReading(packetChan chan gopacket.Packet) {
	s.ciphers = make(map[string]CipherPair)
	s.buffer = make([]byte, 100000)

	for packet := range packetChan {
		flow := packet.TransportLayer().TransportFlow()
		srcPort := binary.BigEndian.Uint16(flow.Src().Raw())
		dstPort := binary.BigEndian.Uint16(flow.Dst().Raw())
		data := packet.ApplicationLayer().Payload()

		clientPacket, server := getServerType(srcPort, dstPort)
		s.handlePacket(server, clientPacket, data)
	}
}

// Guesses the server type based on the sender of the packet and what ports the
// packet was sent between. Also returns whether or not this packet was sent
// by the client.
func getServerType(srcPort, dstPort uint16) (bool, string) {
	if server, ok := serverPorts[dstPort]; ok {
		return true, server
	}
	if server, ok := serverPorts[srcPort]; ok {
		return false, server
	}
	return false, "???"
}

func (s *sniffer) handlePacket(server string, clientPacket bool, data []byte) {
	emitPacket := true

	// Copy the data we just got into the working slice for the current packet.
	s.bufferBytesRead += uint16(copy(s.buffer[s.bufferBytesRead:], data))

	// Peek at the header.
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
		s.currentPacketSize = header.Size
	case packets.LoginWelcomeType:
		var welcomePacket packets.Welcome
		bytes.StructFromBytes(data, &welcomePacket)
		s.ciphers[server] = CipherPair{
			clientCrypt: encryption.NewBBCryptWithVector(welcomePacket.ClientVector[:]),
			serverCrypt: encryption.NewBBCryptWithVector(welcomePacket.ServerVector[:]),
		}
		s.currentPacketSize = header.Size
	default:
		var expectedHeaderSize uint16 = packets.BBHeaderSize
		if server == "PATCH:AUTH" || server == "PATCH:DATA" {
			expectedHeaderSize = packets.PCHeaderSize
		}

		if s.ciphers[server].serverCrypt == nil {
			fmt.Println("exiting: encryption vectors are not initialized")
			fmt.Println("sniffer must be started before starting the game session")
			os.Exit(1)
		}

		// If we're expecting a new packet, read it in and decrypt it.
		if s.currentPacketSize == 0 {
			if clientPacket {
				s.ciphers[server].clientCrypt.Decrypt(s.buffer, uint32(expectedHeaderSize))
			} else {
				s.ciphers[server].serverCrypt.Decrypt(s.buffer, uint32(expectedHeaderSize))
			}
			bytes.StructFromBytes(s.buffer[:expectedHeaderSize], &header)
			s.currentPacketSize = header.Size
			// Like we do elsewhere in the server, make sure we're reading packet lengths that are
			// multiples of the header size. Sometimes the client messes up the size.
			if s.currentPacketSize%expectedHeaderSize != 0 {
				s.currentPacketSize += expectedHeaderSize - (s.currentPacketSize % expectedHeaderSize)
			}
		}

		// Once have the entire packet, decrypt and print it out .
		if s.bufferBytesRead >= s.currentPacketSize {
			if clientPacket {
				s.ciphers[server].clientCrypt.Decrypt(s.buffer[expectedHeaderSize:], uint32(s.currentPacketSize-expectedHeaderSize))
			} else {
				s.ciphers[server].serverCrypt.Decrypt(s.buffer[expectedHeaderSize:], uint32(s.currentPacketSize-expectedHeaderSize))
			}
		} else {
			emitPacket = false
		}
	}

	if emitPacket {
		params := debug.PrintPacketParams{
			Writer:        s.Writer,
			ServerType:    server,
			ClientCommand: clientPacket,
			Data:          s.buffer[:s.currentPacketSize],
		}
		if *truncate {
			params.TruncateThreshold = truncatePacketLimit
		}
		if *interpret {
			params.Interpret = *interpret
		}
		debug.PrintPacket(params)

		// Sometimes multiple payloads might be sent as part of the same pocket. To account
		// for this, recursively call handlePacket with the remaining bytes we read and
		// process it as if it were a new block of data.
		packetSize := s.currentPacketSize
		bufferLength := s.bufferBytesRead
		s.currentPacketSize = 0
		s.bufferBytesRead = 0

		if bufferLength > packetSize {
			s.handlePacket(server, clientPacket, s.buffer[packetSize:bufferLength])
		}
	}
}
