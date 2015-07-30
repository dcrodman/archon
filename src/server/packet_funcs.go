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
* Packet types, defintitions, and sending functions.
 */
package main

import (
	"errors"
	"fmt"
	"server/client"
	"server/util"
	"syscall"
	"time"
)

const (
	// Copyright messages the client expects.
	patchCopyright = "Patch Server. Copyright SonicTeam, LTD. 2001"
	loginCopyright = "Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM."
	// Format for the timestamp sent to the client.
	timeFmt = "2006:01:02: 15:05:05"
)

var (
	patchCopyrightBytes []byte
	loginCopyrightBytes []byte
	serverName          = util.ConvertToUtf16("Archon")
)

// Send the packet serialized (or otherwise contained) in pkt to a client.
// Note: Packets sent to BB Clients must have a length divisible by 8.
func SendPacket(c client.Client, pkt []byte, length uint16) int {
	if err := c.Send(pkt[:length]); err != nil {
		log.Info("Error sending to client %v: %s", c.IPAddr(), err.Error())
		return -1
	}
	return 0
}

// Send data to client after padding it to a length disible by 8 and
// encrypting it with the client's server ciper.
func SendEncrypted(cw client.ClientWrapper, data []byte, length uint16) int {
	c := cw.Client()
	length = fixLength(data, length)
	if config.DebugMode {
		util.PrintPayload(data, int(length))
		fmt.Println()
	}
	c.Encrypt(data, uint32(length))
	return SendPacket(c, data, length)
}

// Pad the length of a packet to a multiple of 8 and set the first two
// bytes of the header.
func fixLength(data []byte, length uint16) uint16 {
	for length%PCHeaderSize != 0 {
		length++
		_ = append(data, 0)
	}
	data[0] = byte(length & 0xFF)
	data[1] = byte((length & 0xFF00) >> 8)
	return length
}

// Send a simple 4-byte header packet.
func SendPCHeader(client *PatchClient, pktType uint16) int {
	pkt := &PCPktHeader{
		Type: pktType,
		Size: 0x04,
	}
	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		util.PrintPayload(data, size)
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the welcome packet to a client with the copyright message and encryption vectors.
func SendPatchWelcome(client *PatchClient) int {
	pkt := new(PatchWelcomePkt)
	pkt.Header.Type = PatchWelcomeType
	pkt.Header.Size = 0x4C
	copy(pkt.Copyright[:], patchCopyrightBytes)
	copy(pkt.ClientVector[:], client.c.ClientVector())
	copy(pkt.ServerVector[:], client.c.ServerVector())

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Welcome Packet")
		util.PrintPayload(data, size)
		fmt.Println()
	}
	return SendPacket(client.c, data, uint16(size))
}

func SendWelcomeAck(client *PatchClient) int {
	pkt := &PCPktHeader{
		Size: 0x04,
		Type: PatchLoginType, // treated as an ack
	}
	data, _ := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Welcome Ack")
	}
	return SendEncrypted(client, data, 0x0004)
}

func SendWelcomeMessage(client *PatchClient) int {
	cfg := config
	pkt := new(WelcomeMessage)
	pkt.Header.Type = PatchMessageType
	pkt.Header.Size = PCHeaderSize + cfg.MessageSize
	pkt.Message = cfg.MessageBytes

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Welcome Message")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the redirect packet, providing the IP and port of the next server.
func SendPatchRedirect(client *PatchClient, port uint16, ipAddr [4]byte) int {
	pkt := new(PatchRedirectPacket)
	pkt.Header.Type = PatchRedirectType
	copy(pkt.IPAddr[:], ipAddr[:])
	pkt.Port = port

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Redirect")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Acknowledgement sent after the DATA connection handshake.
func SendDataAck(client *PatchClient) int {
	if config.DebugMode {
		fmt.Println("Sending Data Ack")
	}
	return SendPCHeader(client, PatchDataAckType)
}

// Tell the client to change to one directory above.
func SendDirAbove(client *PatchClient) int {
	if config.DebugMode {
		fmt.Println("Sending Dir Above")
	}
	return SendPCHeader(client, PatchDirAboveType)
}

// Tell the client to change to some directory within its file tree.
func SendChangeDir(client *PatchClient, dir string) int {
	pkt := new(ChangeDirPacket)
	pkt.Header.Type = PatchChangeDirType
	copy(pkt.Dirname[:], dir)

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Change Directory")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Tell the client to check a file in its current working directory.
func SendCheckFile(client *PatchClient, index uint32, filename string) int {
	pkt := new(CheckFilePacket)
	pkt.Header.Type = PatchCheckFileType
	pkt.PatchId = index
	copy(pkt.Filename[:], filename)

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Check File")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Inform the client that we've finished sending the patch list.
func SendFileListDone(client *PatchClient) int {
	if config.DebugMode {
		fmt.Println("Sending List Done")
	}
	return SendPCHeader(client, PatchFileListDoneType)
}

// Send the total number and cumulative size of files that need updating.
func SendUpdateFiles(client *PatchClient, num, totalSize uint32) int {
	pkt := new(UpdateFilesPacket)
	pkt.Header.Type = PatchUpdateFilesType
	pkt.NumFiles = num
	pkt.TotalSize = totalSize

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Update Files")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the header for a file we're about to update.
func SendFileHeader(client *PatchClient, patch *PatchEntry) int {
	pkt := new(FileHeaderPacket)
	pkt.Header.Type = PatchFileHeaderType
	pkt.FileSize = patch.fileSize
	copy(pkt.Filename[:], patch.filename)

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending File Header")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send a chunk of file data.
func SendFileChunk(client *PatchClient, chunk, chksm, chunkSize uint32, fdata []byte) int {
	if chunkSize > MaxFileChunkSize {
		log.Error("Attempted to send %v byte chunk; max is %v",
			string(chunkSize), string(MaxFileChunkSize))
		panic(errors.New("File chunk size exceeds maximum"))
	}
	pkt := new(FileChunkPacket)
	pkt.Header.Type = PatchFileChunkType
	pkt.Chunk = chunk
	pkt.Checksum = chksm
	pkt.Size = chunkSize
	pkt.Data = fdata[:chunkSize]

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending File Chunk")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Finished sending a particular file.
func SendFileComplete(client *PatchClient) int {
	if config.DebugMode {
		fmt.Println("Sending File Complete")
	}
	return SendPCHeader(client, PatchFileCompleteType)
}

// We've finished updating files.
func SendUpdateComplete(client *PatchClient) int {
	if config.DebugMode {
		fmt.Println("Sending File Update Done")
	}
	return SendPCHeader(client, PatchUpdateCompleteType)
}

// Send the welcome packet to a client with the copyright message and encryption vectors.
func SendLoginWelcome(client *LoginClient) int {
	pkt := new(WelcomePkt)
	pkt.Header.Type = LoginWelcomeType
	pkt.Header.Size = 0xC8
	copy(pkt.Copyright[:], loginCopyrightBytes)
	copy(pkt.ClientVector[:], client.c.ClientVector())
	copy(pkt.ServerVector[:], client.c.ServerVector())

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Welcome Packet")
		util.PrintPayload(data, size)
		fmt.Println()
	}
	return SendPacket(client.c, data, uint16(size))
}

// Send the security initialization packet with information about the user's
// authentication status.
func SendSecurity(client *LoginClient, errorCode BBLoginError, guildcard uint32, teamId uint32) int {
	pkt := new(SecurityPacket)
	pkt.Header.Type = LoginSecurityType

	// Constants set according to how Newserv does it.
	pkt.ErrorCode = uint32(errorCode)
	pkt.PlayerTag = 0x00010000
	pkt.Guildcard = guildcard
	pkt.TeamId = teamId
	pkt.Config = &client.config
	pkt.Capabilities = 0x00000102

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Security Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the redirect packet, providing the IP and port of the next server.
func SendRedirect(client *LoginClient, port uint16, ipAddr [4]byte) int {
	pkt := new(RedirectPacket)
	pkt.Header.Type = LoginRedirectType
	copy(pkt.IPAddr[:], ipAddr[:])
	pkt.Port = port

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
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
	pkt.Header.Type = LoginOptionsType

	pkt.PlayerKeyConfig.Guildcard = client.guildcard
	copy(pkt.PlayerKeyConfig.KeyConfig[:], keyConfig[:0x16C])
	copy(pkt.PlayerKeyConfig.JoystickConfig[:], keyConfig[0x16C:])

	// Sylverant sets these to enable all team rewards? Not sure what this means yet.
	pkt.PlayerKeyConfig.TeamRewards[0] = 0xFFFFFFFF
	pkt.PlayerKeyConfig.TeamRewards[1] = 0xFFFFFFFF

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Key Config Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the character acknowledgement packet. 0 indicates a creation ack, 1 is
// ack'ing a selected character, and 2 indicates that a character doesn't exist
// in the slot requested via preview request.
func SendCharacterAck(client *LoginClient, slotNum uint32, flag uint32) int {
	pkt := new(CharAckPacket)
	pkt.Header.Type = LoginCharAckType
	pkt.Slot = slotNum
	pkt.Flag = flag

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Character Ack Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the preview packet containing basic details about a character in
// the selected slot.
func SendCharacterPreview(client *LoginClient, charPreview *CharacterPreview) int {
	pkt := new(CharPreviewPacket)
	pkt.Header.Type = LoginCharPreviewType
	pkt.Slot = 0
	pkt.Character = charPreview

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Character Preview Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Acknowledge the checksum the client sent us. We don't actually do
// anything with it but the client won't proceed otherwise.
func SendChecksumAck(client *LoginClient, ack uint32) int {
	pkt := new(ChecksumAckPacket)
	pkt.Header.Type = LoginChecksumAckType
	pkt.Ack = ack

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Checksum Ack Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the guildcard chunk header.
func SendGuildcardHeader(client *LoginClient, checksum uint32, dataLen uint16) int {
	pkt := new(GuildcardHeaderPacket)
	pkt.Header.Type = LoginGuildcardHeaderType
	pkt.Unknown = 0x00000001
	pkt.Length = dataLen
	pkt.Checksum = checksum

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Guildcard Header Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the next chunk of guildcard data.
func SendGuildcardChunk(client *LoginClient, chunkNum uint32) int {
	pkt := new(GuildcardChunkPacket)
	pkt.Header.Type = LoginGuildcardChunkType
	pkt.Chunk = chunkNum

	// The client will only accept 0x6800 bytes of a chunk per packet.
	offset := uint16(chunkNum) * MaxChunkSize
	remaining := client.gcDataSize - offset
	if remaining > MaxChunkSize {
		pkt.Data = client.gcData[offset : offset+MaxChunkSize]
	} else {
		pkt.Data = client.gcData[offset:]
	}

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Guildcard Chunk Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the header for the parameter files we're about to start sending.
func SendParameterHeader(client *LoginClient, numEntries uint32, entries []byte) int {
	pkt := new(ParameterHeaderPacket)
	pkt.Header.Type = LoginParameterHeaderType
	pkt.Header.Flags = numEntries
	pkt.Entries = entries

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Parameter Header Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Index into chunkData and send the specified chunk of parameter data.
func SendParameterChunk(client *LoginClient, chunkData []byte, chunk uint32) int {
	pkt := new(ParameterChunkPacket)
	pkt.Header.Type = LoginParameterChunkType
	pkt.Chunk = chunk

	pkt.Data = chunkData

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Parameter Chunk Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send an error message to the client, usually used before disconnecting.
func SendClientMessage(client *LoginClient, message string) int {
	pkt := new(ClientMessagePacket)
	pkt.Header.Type = LoginClientMessageType
	// English? Tethealla sets this.
	pkt.Language = 0x00450009
	pkt.Message = util.ConvertToUtf16(message)

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Client Message Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send a timestamp packet in order to indicate the server's current time.
func SendTimestamp(client *LoginClient) int {
	pkt := new(TimestampPacket)
	pkt.Header.Type = LoginTimestampType

	var tv syscall.Timeval
	syscall.Gettimeofday(&tv)
	t := time.Now().Format(timeFmt)
	stamp := fmt.Sprintf("%s.%03d", t, uint64(tv.Usec/1000))
	copy(pkt.Timestamp[:], stamp)

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Timestamp Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the menu items for the ship select screen. ships must always
// contain at least one entry, the default being "No Ships".
func SendShipList(client *LoginClient, ships []ShipEntry) int {
	pkt := new(ShipListPacket)
	pkt.Header.Type = LoginShipListType

	pkt.Header.Flags = 0x01
	pkt.Unknown = 0x02
	pkt.Unknown2 = 0xFFFFFFF4
	pkt.Unknown3 = 0x04
	copy(pkt.ServerName[:], serverName)
	// Global mutex, what could possibly go wrong?
	// shipListMutex.RLock()
	pkt.ShipEntries = ships
	// shipListMutex.RUnlock()

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Ship List Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send whatever scrolling message was set in the config file and
// converted to UTF-16LE when the server started up.
func SendScrollMessage(client *LoginClient) int {
	pkt := new(ScrollMessagePacket)
	pkt.Header.Type = LoginScrollMessageType
	pkt.Message = config.ScrollMessageBytes()

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Scroll Message Packet")
	}
	return SendEncrypted(client, data, uint16(size))
}

func init() {
	patchCopyrightBytes = []byte(patchCopyright)
	loginCopyrightBytes = []byte(loginCopyright)
}
