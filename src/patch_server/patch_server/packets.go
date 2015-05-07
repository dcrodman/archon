/*
* Archon Patch Server
* Copyright (C) 2014 Andrew Rodman
*
* This program is free software: you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License
* along with this program.  If not, see <http://www.gnu.org/licenses/>.
* ---------------------------------------------------------------------
 */
package patch_server

import (
	"fmt"
	"libarchon/logger"
	"libarchon/util"
)

const BBHeaderSize = 0x04
const bbCopyright = "Patch Server. Copyright SonicTeam, LTD. 2001"

var copyrightBytes []byte = make([]byte, 96)

// Packet types for packets sent to and from the login and character servers.
const (
	WelcomeType    = 0x02
	WelcomeAckType = 0x04 // sent
	LoginType      = 0x04 // received
)

// Packet header for every packet sent between the server and BlueBurst clients.
type BBPktHeader struct {
	Size uint16
	Type uint16
}

// Welcome packet with encryption vectors sent to the client upon initial connection.
type WelcomePkt struct {
	Header       BBPktHeader
	Copyright    [44]byte
	Padding      [20]byte
	ServerVector [4]byte
	ClientVector [4]byte
}

// Send the packet serialized (or otherwise contained) in pkt to a client.
// Note: Packets sent to BB Clients must have a length divisible by 8.
func SendPacket(client *PatchClient, pkt []byte, length uint16) int {
	_, err := client.conn.Write(pkt[:length])
	if err != nil {
		log.Info("Error sending to client "+client.ipAddr+": "+err.Error(),
			logger.LogPriorityMedium)
		return -1
	}
	return 0
}

// Send data to client after padding it to a length disible by 8 and
// encrypting it with the client's server ciper.
func SendEncrypted(client *PatchClient, data []byte, length uint16) int {
	length = fixLength(data, length)
	if GetConfig().DebugMode {
		util.PrintPayload(data, int(length))
		fmt.Println()
	}
	client.serverCrypt.Encrypt(data, uint32(length))
	return SendPacket(client, data, length)
}

// Send the welcome packet to a client with the copyright message and encryption vectors.
func SendWelcome(client *PatchClient) int {
	pkt := new(WelcomePkt)
	pkt.Header.Type = WelcomeType
	pkt.Header.Size = 0x4C
	copy(pkt.Copyright[:], copyrightBytes)
	copy(pkt.ClientVector[:], client.clientCrypt.Vector)
	copy(pkt.ServerVector[:], client.serverCrypt.Vector)

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Welcome Packet")
		util.PrintPayload(data, size)
		fmt.Println()
	}
	return SendPacket(client, data, uint16(size))
}

func SendWelcomeAck(client *PatchClient) int {
	pkt := new(BBPktHeader)
	pkt.Size = 0x04
	pkt.Type = WelcomeAckType
	data, _ := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Welcome Ack")
	}
	return SendEncrypted(client, data, 0x0004)
}

// Pad the length of a packet to a multiple of 8 and set the first two
// bytes of the header.
func fixLength(data []byte, length uint16) uint16 {
	for length%4 != 0 {
		length++
		_ = append(data, 0)
	}
	data[0] = byte(length & 0xFF)
	data[1] = byte((length & 0xFF00) >> 8)
	return length
}

func init() {
	copy(copyrightBytes, bbCopyright)
}
