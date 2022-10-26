package main

import (
	"encoding/binary"

	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/encryption"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/google/gopacket"
)

var (
	knownPatchPorts = []uint16{
		11000, 11001,
	}
	knownPorts = []uint16{
		12000, 12001, 5278, 5279, 5280,
	}
)

type sniffer struct {
	patchClientCrypt *encryption.PSOCrypt
	patchServerCrypt *encryption.PSOCrypt

	clientCrypt *encryption.PSOCrypt
	serverCrypt *encryption.PSOCrypt
}

func (s *sniffer) startIngesting(packetChan chan gopacket.Packet) {
	knownPorts = append(knownPorts, knownPatchPorts...)
	knownPortSet := make(map[uint16]struct{})
	for _, port := range knownPorts {
		knownPortSet[port] = struct{}{}
	}

	for packet := range packetChan {
		flow := packet.TransportLayer().TransportFlow()
		srcPort := binary.BigEndian.Uint16(flow.Src().Raw())
		dstPort := binary.BigEndian.Uint16(flow.Dst().Raw())

		var patchServerPacket bool
		for _, port := range knownPatchPorts {
			if srcPort == port || dstPort == port {
				patchServerPacket = true
				break
			}
		}
		data := packet.ApplicationLayer().Payload()

		// Make sure we've initialized our encryption vectors or the data will be gibberish.
		var header packets.PCHeader
		bytes.StructFromBytes(data[:packets.PCHeaderSize], &header)
		if s.patchClientCrypt == nil && patchServerPacket {
			if header.Type != packets.PatchWelcomeType {
				panic("should have been patch welcome type")
			}
			var welcomePacket packets.PatchWelcome
			bytes.StructFromBytes(data, &welcomePacket)
			s.patchClientCrypt = encryption.NewPCCryptWithVector(welcomePacket.ClientVector[:])
			s.patchServerCrypt = encryption.NewPCCryptWithVector(welcomePacket.ServerVector[:])
		} else if s.clientCrypt == nil && !patchServerPacket {
			if header.Type != packets.LoginWelcomeType {
				panic("should have been login welcome type")
			}
			var welcomePacket packets.Welcome
			bytes.StructFromBytes(data, &welcomePacket)
			s.clientCrypt = encryption.NewBBCryptWithVector(welcomePacket.ClientVector[:])
			s.serverCrypt = encryption.NewBBCryptWithVector(welcomePacket.ServerVector[:])
		}

		// Determine whether this packet is coming from the client or the server.
		// var clientPacket bool
		// if _, ok := knownPortSet[srcPort]; ok {
		// 	clientPacket = true
		// } else if _, ok := knownPortSet[dstPort]; !ok {
		// 	fmt.Printf("received packet to/from unrecognized ports; source=%v dest=%v\n", srcPort, dstPort)
		// 	continue
		// }
	}
}
