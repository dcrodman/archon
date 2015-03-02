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
	"fmt"
	"libarchon/util"
)

const BBHeaderSize = 0x08
const bbCopyright = "Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM."

// Packet types for packets sent to and from the login and character servers.
const (
	WelcomeType            = 0x03
	DisconnectType         = 0x05
	LoginType              = 0x93
	SecurityType           = 0xE6
	RedirectType           = 0x19
	OptionsRequestType     = 0xE0
	OptionsType            = 0xE2
	CharPreviewReqType     = 0xE3
	CharPreviewNoneType    = 0xE4
	CharPreviewType        = 0xE5
	ChecksumType           = 0x01E8
	ChecksumAckType        = 0x02E8
	GuildcardReqType       = 0x03E8
	GuildcardHeaderType    = 0x01DC
	GuildcardChunkType     = 0x02DC
	GuildcardChunkReqType  = 0x03DC
	ParameterHeaderType    = 0x01EB
	ParameterChunkType     = 0x02EB
	ParameterChunkReqType  = 0x03EB
	ParameterHeaderReqType = 0x04EB
)

const MAX_CHUNK_SIZE = 0x6800

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
	Size  uint16
	Type  uint16
	Flags uint32
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

// The address of the next server; in this case, the character server.
type RedirectPacket struct {
	Header  BBPktHeader
	IPAddr  [4]uint8
	Port    uint16
	Padding uint16
}

// Based on the key config structure from sylverant and newserv. KeyConfig
// and JoystickConfig are saved in the database.
type KeyTeamConfig struct {
	Unknown            [0x114]uint8
	KeyConfig          [0x16C]uint8
	JoystickConfig     [0x38]uint8
	Guildcard          uint32
	TeamId             uint32
	TeamInfo           [2]uint32
	TeamPrivilegeLevel uint16
	Reserved           uint16
	Teamname           [0x10]uint16
	TeamFlag           [0x0800]uint8
	TeamRewards        [2]uint32
}

// Option packet containing keyboard and joystick config, team options, etc.
type OptionsPacket struct {
	Header          BBPktHeader
	PlayerKeyConfig KeyTeamConfig
}

type CharPreviewRequestPacket struct {
	Header  BBPktHeader
	Slot    uint32
	Padding uint32
}

type CharPreviewNonePacket struct {
	Header BBPktHeader
	Slot   uint32
	Error  uint32
}

// Sent in response to 0x01E8 to acknowledge a checksum (really it's just ignored).
type ChecksumAckPacket struct {
	Header BBPktHeader
	Ack    uint32
}

// Chunk header with info about the guildcard data we're about to send.
type GuildcardHeaderPacket struct {
	Header   BBPktHeader
	Unknown  uint32
	Length   uint16
	Padding  uint16
	Checksum uint32
}

// Received from the client to request a guildcard data chunk.
type GuildcardChunkReqPacket struct {
	Header         BBPktHeader
	Unknown        uint32
	ChunkRequested uint32
	Continue       uint32
}

type GuildcardChunkPacket struct {
	Header  BBPktHeader
	Unknown uint32
	Chunk   uint32
	Data    []uint8
}

// Parameter header containing details about the param files we're about to send.
type ParameterHeaderPacket struct {
	Header  BBPktHeader
	Entries []byte
}

type ParameterChunkPacket struct {
	Header BBPktHeader
	Chunk  uint32
	Data   []byte
}

// Send the packet serialized (or otherwise contained) in pkt to a client.
// Note: Packets sent to BB Clients must have a length divisible by 8.
func SendPacket(client *LoginClient, pkt []byte, length uint16) int {
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
func SendEncrypted(client *LoginClient, data []byte, length uint16) int {
	length = fixLength(data, length)
	if GetConfig().DebugMode {
		util.PrintPayload(data, int(length))
		fmt.Println()
	}
	client.serverCrypt.Encrypt(data, uint32(length))
	return SendPacket(client, data, length)
}

// Send the welcome packet to a client with the copyright message and encryption vectors.
func SendWelcome(client *LoginClient) int {
	pkt := new(WelcomePkt)
	pkt.Header.Type = WelcomeType
	pkt.Header.Size = 0xC8
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

// Send the security initialization packet with information about the user's
// authentication status.
func SendSecurity(client *LoginClient, errorCode BBLoginError, teamId uint32) int {
	pkt := new(SecurityPacket)
	pkt.Header.Type = SecurityType

	// Constants set according to how Newserv does it.
	pkt.ErrorCode = uint32(errorCode)
	pkt.PlayerTag = 0x00010000
	pkt.Guildcard = client.guildcard
	pkt.TeamId = teamId
	pkt.Capabilities = 0x00000102
	pkt.Config.Magic = 0x48615467

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Security Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the redirect packet, providing the IP and port of the next server.
func SendRedirect(client *LoginClient, port uint16, ipAddr [4]byte) int {
	pkt := new(RedirectPacket)
	pkt.Header.Type = RedirectType
	copy(pkt.IPAddr[:], ipAddr[:])
	pkt.Port = port

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Redirect Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the client's configuration options. keyConfig should be 420 bytes long and either
// point to the default keys array or loaded from the database.
func SendOptions(client *LoginClient, keyConfig []byte) int {
	if len(keyConfig) != 420 {
		panic("Received keyConfig of length " + string(len(keyConfig)) + "; should be 420")
	}
	pkt := new(OptionsPacket)
	pkt.Header.Type = OptionsType

	pkt.PlayerKeyConfig.Guildcard = client.guildcard
	// TODO: What to do about team stuff?
	copy(pkt.PlayerKeyConfig.KeyConfig[:], keyConfig[:0x16C])
	copy(pkt.PlayerKeyConfig.JoystickConfig[:], keyConfig[0x16C:])

	// Sylverant sets these to enable all team rewards? Not sure what this means yet.
	pkt.PlayerKeyConfig.TeamRewards[0] = 0xFFFFFFFF
	pkt.PlayerKeyConfig.TeamRewards[1] = 0xFFFFFFFF

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Key Config Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the character preview acknowledgement packet to tell them that we don't
// have any data for that slot.
func SendCharPreviewNone(client *LoginClient, slotNum uint32) int {
	pkt := new(CharPreviewNonePacket)
	pkt.Header.Type = CharPreviewNoneType
	pkt.Slot = slotNum
	pkt.Error = 0x02

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Character Ack Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

func SendCharPreview() {
	// TODO
}

// Acknowledge the checksum the client sent us.
func SendChecksumAck(client *LoginClient, ack uint32) int {
	pkt := new(ChecksumAckPacket)
	pkt.Header.Type = ChecksumAckType
	pkt.Ack = ack

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Checksum Ack Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the guildcard chunk header.
func SendGuildcardHeader(client *LoginClient, checksum uint32, dataLen uint16) int {
	pkt := new(GuildcardHeaderPacket)
	pkt.Header.Type = GuildcardHeaderType
	pkt.Unknown = 0x00000001
	pkt.Length = dataLen
	pkt.Checksum = checksum

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Guildcard Header Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the next chunk of guildcard data.
func SendGuildcardChunk(client *LoginClient, chunkNum uint32) int {
	pkt := new(GuildcardChunkPacket)
	pkt.Header.Type = GuildcardChunkType
	pkt.Chunk = chunkNum

	// The client will only accept 0x6800 bytes of a chunk per packet.
	offset := uint16(chunkNum) * MAX_CHUNK_SIZE
	remaining := client.gcDataSize - offset
	if remaining > MAX_CHUNK_SIZE {
		pkt.Data = client.gcData[offset : offset+MAX_CHUNK_SIZE]
	} else {
		pkt.Data = client.gcData[offset:]
	}

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Guildcard Chunk Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the header for the parameter files we're about to start sending.
func SendParameterHeader(client *LoginClient, numEntries uint32, entries []byte) int {
	pkt := new(ParameterHeaderPacket)
	pkt.Header.Type = ParameterHeaderType
	pkt.Header.Flags = numEntries
	pkt.Entries = entries

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Parameter Header Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Index into chunkData and send the specified chunk of parameter data.
func SendParameterChunk(client *LoginClient, chunkData []byte, chunk uint32) int {
	pkt := new(ParameterChunkPacket)
	pkt.Header.Type = ParameterChunkType
	pkt.Chunk = chunk

	pkt.Data = chunkData

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Parameter Chunk Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Pad the length of a packet to a multiple of 8 and set the first two
// bytes of the header.
func fixLength(data []byte, length uint16) uint16 {
	for length%8 != 0 {
		length++
		_ = append(data, 0)
	}
	// TODO: This might cause problems on big endian machines.
	data[0] = byte(length & 0xFF)
	data[1] = byte((length & 0xFF00) >> 8)
	return length
}

func init() {
	copy(copyrightBytes, bbCopyright)
}
