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
* Packet handlers. All functions return 0 on success, negative int on
* db error, and a positive int for any other errors.
 */
package login_server

import (
	"fmt"
	"libarchon/logger"
	"libarchon/util"
	"syscall"
	"time"
)

const TimeFmt = "2006:01:02: 15:05:05"

var copyrightBytes []byte = make([]byte, 96)

// Send the packet serialized (or otherwise contained) in pkt to a client.
// Note: Packets sent to BB Clients must have a length divisible by 8.
func SendPacket(client *LoginClient, pkt []byte, length uint16) int {
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
func SendSecurity(client *LoginClient, errorCode BBLoginError, guildcard uint32, teamId uint32) int {
	pkt := new(SecurityPacket)
	pkt.Header.Type = SecurityType

	// Constants set according to how Newserv does it.
	pkt.ErrorCode = uint32(errorCode)
	pkt.PlayerTag = 0x00010000
	pkt.Guildcard = guildcard
	pkt.TeamId = teamId
	pkt.Config = &client.config
	pkt.Capabilities = 0x00000102

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
func SendCharacterAck(client *LoginClient, slotNum uint32, flag uint32) int {
	pkt := new(CharAckPacket)
	pkt.Header.Type = CharAckType
	pkt.Slot = slotNum
	pkt.Flag = flag

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Character Ack Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the preview packet containing basic details about a character in
// the selected slot.
func SendCharacterPreview(client *LoginClient, charPreview *CharacterPreview) int {
	pkt := new(CharPreviewPacket)
	pkt.Header.Type = CharPreviewType
	pkt.Slot = 0
	pkt.Character = charPreview

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Character Preview Packet")
	}
	return SendEncrypted(client, data, uint16(size))
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

// Send an error message to the client, usually used before disconnecting.
func SendClientMessage(client *LoginClient, message string) int {
	pkt := new(ClientMessagePacket)
	pkt.Header.Type = ClientMessageType
	// English? Tethealla sets this.
	pkt.Language = 0x00450009
	pkt.Message = util.ConvertToUtf16(message)

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Client Message Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

func SendTimestamp(client *LoginClient) int {
	pkt := new(TimestampPacket)
	pkt.Header.Type = TimestampType

	var tv syscall.Timeval
	syscall.Gettimeofday(&tv)
	t := time.Now().Format(TimeFmt)
	stamp := fmt.Sprintf("%s.%03d", t, uint64(tv.Usec/1000))
	copy(pkt.Timestamp[:], stamp)

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Timestamp Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

func SendShipList(client *LoginClient) {
	// TODO
}

// Send whatever scrolling message was set in the config file and
// converted to UTF-16LE when the server started up.
func SendScrollMessage(client *LoginClient) int {
	pkt := new(ScrollMessagePacket)
	pkt.Header.Type = ScrollMessageType
	pkt.Message = GetConfig().cachedWelcomeMsg

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Scroll Message Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Pad the length of a packet to a multiple of 8 and set the first two
// bytes of the header.
func fixLength(data []byte, length uint16) uint16 {
	for length%BBHeaderSize != 0 {
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
