/*
* Archon Login Server
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
*
* Packet constants and handlers. All handlers return 0 on success, negative int on
* db error, and a positive int for any other errors.
 */
package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"libarchon/util"
)

const BBHeaderSize = 0x08
const bbCopyright = "Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM."

// Packet types for packets sent to and from the login and character servers.
const (
	WelcomeType    = 0x03
	DisconnectType = 0x05
	LoginType      = 0x93
	SecurityType   = 0xE6
)

// Sizes of packets sent to the client. Not a 1-1 mapping with the types above, these are
// helpful because it's difficult to get the full packet size and update the packet header.
const (
	WelcomeSize  = 0xC8
	SecuritySize = 0x44
	MessageSize  = 0x12
)

// Error code types used for packet E6.
type BBLoginError uint32

const (
	BBLoginErrorNone         = 0x0
	BBLoginErrorUnknown      = 0x1
	BBLoginErrorPassword     = 0x2
	BBLoginErrorPassword2    = 0x3 // Same as password
	BBLoginErrorMaintenance  = 0x4
	BBLoginErrorUserInUse    = 0x5
	BBLoginErrorBanned       = 0x6
	BBLoginErrorBanned2      = 0x7 // Same as banned
	BBLoginErrorUnregistered = 0x8
	BBLoginErrorExpiredSub   = 0x9
	BBLoginErrorLocked       = 0xA
	BBLoginErrorPatch        = 0xB
	BBLoginErrorDisconnect   = 0xC
)

var copyrightBytes []byte = make([]byte, 96)

// Packet header for every packet sent between the server and BlueBurst clients.
type BBPktHeader struct {
	Size    uint16
	Type    uint16
	Padding uint32
}

// Welcome packet with encryption vectors sent to the client upon initial connection.
type WelcomePkt struct {
	Header       BBPktHeader
	Copyright    [96]byte
	ServerVector [48]byte
	ClientVector [48]byte
}

// Login Packet (0x93) sent to both the login and character servers.
type LoginPkt struct {
	Header        BBPktHeader
	Unknown       [8]byte
	ClientVersion uint16
	Unknown2      [6]byte
	TeamId        uint32
	Username      [16]byte
	Padding       [32]byte
	Password      [16]byte
	Unknown3      [40]byte
	HardwareInfo  [8]byte
	Version       [40]byte
}

// Not entirely sure what this is for.
type BBClientConfig struct {
	Magic       uint32 // Must be set to 0x48615467
	BBGameState uint8  // Status of connected client
	BBPlayerNum uint8  // Selected Character
	Flags       uint16
	Ports       [4]uint16
	Unused      [4]uint32
	Unused2     [2]uint32
}

// Security packet (0xE6) sent to the client to indicate the state of client login.
type SecurityPacket struct {
	Header       BBPktHeader
	ErrorCode    uint32
	PlayerTag    uint32
	Guildcard    uint32
	TeamId       uint32
	Config       BBClientConfig
	Capabilities uint32
	Padding      uint32
}


// Send the packet serialized (or otherwise contained) in pkt to a client.
// Note: Packets sent to BB Clients must have a length divisible by 8.
func SendPacket(client *LoginClient, pkt []byte, length int) int {
	_, err := client.conn.Write(pkt[:length])
	if err != nil {
		LogMsg("Error sending to client "+client.ipAddr+": "+err.Error(),
			LogTypeInfo, LogPriorityMedium)
		return -1
	}
	return 0
}

// Send data to client after padding it to a length disible by 8 and
// encrypting it with the client's server ciper.
func SendEncrypted(client *LoginClient, data []byte, length int) int {
	for length%8 != 0 {
		length++
		data = append(data, 0)
	}
	client.serverCrypt.Encrypt(data, uint32(length))
	return SendPacket(client, data, length)
}

// Send the welcome packet to a client with the copyright message and encryption vectors.
func SendWelcome(client *LoginClient) int {
	pkt := new(WelcomePkt)
	pkt.Header.Size = WelcomeSize
	pkt.Header.Type = WelcomeType
	copy(pkt.Copyright[:], copyrightBytes)
	copy(pkt.ClientVector[:], client.clientCrypt.Vector)
	copy(pkt.ServerVector[:], client.serverCrypt.Vector)

	data := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Welcome Packet")
		util.PrintPayload(data, WelcomeSize)
	}
	return SendPacket(client, data, WelcomeSize)
}

func SendSecurity(client *LoginClient, errorCode BBLoginError, teamId uint32) int {
	pkt := new(SecurityPacket)
	pkt.Header.Type = SecurityType
	pkt.Header.Size = SecuritySize

	// Constants set according to how Newserv does it.
	pkt.ErrorCode = uint32(errorCode)
	pkt.PlayerTag = 0x00010000
	pkt.Guildcard = client.guildcard
	pkt.TeamId = teamId
	pkt.Capabilities = 0x00000102
	pkt.Config.Magic = 0x48615467

	data := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Security Packet")
		util.PrintPayload(data, SecuritySize)
	}
	return SendEncrypted(client, data, SecuritySize)
}

func init() {
	copy(copyrightBytes, bbCopyright)
}
