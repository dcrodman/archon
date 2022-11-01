package main

import (
	"bufio"
	"encoding/binary"

	"github.com/google/gopacket"

	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/debug"
	"github.com/dcrodman/archon/internal/core/encryption"
	"github.com/dcrodman/archon/internal/packets"
)

// Best effort guess as to what ports correspond to which servers based on
// the defaults for the servers this tool will likely be used with.
var serverPorts = map[uint16]debug.ServerType{
	11000: debug.PATCH_SERVER,
	11001: debug.DATA_SERVER,
	12000: debug.LOGIN_SERVER,
	12001: debug.CHARACTER_SERVER,
	// Archon's ports.
	15000: debug.SHIP_SERVER,
	15001: debug.BLOCK_SERVER,
	// Tethealla's ports.
	5278: debug.SHIP_SERVER,
	5279: debug.BLOCK_SERVER,
	5280: debug.BLOCK_SERVER,
}

type CipherPair struct {
	clientCrypt *encryption.PSOCrypt
	serverCrypt *encryption.PSOCrypt
}

type sniffer struct {
	Writer *bufio.Writer

	ciphers map[debug.ServerType]CipherPair
}

func (s *sniffer) startReading(packetChan chan gopacket.Packet) {
	s.ciphers = make(map[debug.ServerType]CipherPair)

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
func getServerType(srcPort, dstPort uint16) (bool, debug.ServerType) {
	if server, ok := serverPorts[dstPort]; ok {
		return true, server
	}
	if server, ok := serverPorts[srcPort]; ok {
		return false, server
	}
	return false, debug.UNKNOWN
}

func (s *sniffer) handlePacket(server debug.ServerType, clientPacket bool, data []byte) {
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
	case packets.LoginWelcomeType:
		var welcomePacket packets.Welcome
		bytes.StructFromBytes(data, &welcomePacket)
		s.ciphers[server] = CipherPair{
			clientCrypt: encryption.NewBBCryptWithVector(welcomePacket.ClientVector[:]),
			serverCrypt: encryption.NewBBCryptWithVector(welcomePacket.ServerVector[:]),
		}
	default:
		// Anything else is meaningless since it's encrypted, so decrypt it
		// and read the hader again. Choose which cipher to use depending on
		// which side sent it since they're different vectors.
		if clientPacket {
			s.ciphers[server].clientCrypt.Decrypt(data, uint32(len(data)))
		} else {
			s.ciphers[server].serverCrypt.Decrypt(data, uint32(len(data)))
		}
		bytes.StructFromBytes(data[:packets.PCHeaderSize], &header)
	}

	debug.PrintPacket(debug.PrintPacketParams{
		Writer:       s.Writer,
		ServerType:   server,
		ClientPacket: clientPacket,
		Data:         data,
	})
}
