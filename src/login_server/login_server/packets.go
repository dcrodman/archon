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
* Packet constants and structures.
 */
package login_server

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
	SetFlagType            = 0x00EC
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
	Security      [40]byte
}

// Represent the client's progression through the login process.
type ClientConfig struct {
	Magic        uint32 // Must be set to 0x48615467
	CharSelected uint8  // Has a character been selected?
	SlotNum      uint8  // Slot number of selected Character
	Flags        uint16
	Ports        [4]uint16
	Unused       [4]uint32
	Unused2      [2]uint32
}

// Security packet (0xE6) sent to the client to indicate the state of client login.
type SecurityPacket struct {
	Header       BBPktHeader
	ErrorCode    uint32
	PlayerTag    uint32
	Guildcard    uint32
	TeamId       uint32
	Config       *ClientConfig
	Capabilities uint32
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

type SetFlagPacket struct {
	Header BBPktHeader
	Flag   uint32
}

type CharPreviewPacket struct {
	Header    BBPktHeader
	Slot      uint32
	Character *CharacterPreview
}
