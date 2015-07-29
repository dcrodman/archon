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
	"server/util"
)

const (
	// Copyright messages the client expects.
	patchCopyright = "Patch Server. Copyright SonicTeam, LTD. 2001"
	loginCopyright = "Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM."
	// Maximum size of a block of guildcard data.
	MaxGCChunkSize = 0x6800
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
func SendPacket(client *PatchClient, pkt []byte, length uint16) int {
	if err := client.c.Send(pkt[:length]); err != nil {
		log.Info("Error sending to client %v: %s", client.IPAddr(), err.Error())
		return -1
	}
	return 0
}

// Send data to client after padding it to a length disible by 8 and
// encrypting it with the client's server ciper.
func SendEncrypted(client *PatchClient, data []byte, length uint16) int {
	length = fixLength(data, length)
	if config.DebugMode {
		util.PrintPayload(data, int(length))
		fmt.Println()
	}
	client.c.Encrypt(data, uint32(length))
	return SendPacket(client, data, length)
}

// Send a simple 4-byte header packet.
func SendPCHeader(client *PatchClient, pktType uint16) int {
	pkt := &PCPktHeader{
		Type: pktType,
		Size: 0x04,
	}
	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		util.PrintPayload()
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the welcome packet to a client with the copyright message and encryption vectors.
func SendPatchWelcome(client *PatchClient) int {
	pkt := new(WelcomePkt)
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
	return SendPacket(client, data, uint16(size))
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
func SendRedirect(client *PatchClient, port uint16, ipAddr [4]byte) int {
	pkt := new(RedirectPacket)
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
	return SendHeader(client, PatchDataAckType)
}

// Tell the client to change to one directory above.
func SendDirAbove(client *PatchClient) int {
	if config.DebugMode {
		fmt.Println("Sending Dir Above")
	}
	return SendHeader(client, PatchDirAboveType)
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
	return SendHeader(client, PatchFileListDoneType)
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
	if chunkSize > MaxChunkSize {
		log.Error("Attempted to send %v byte chunk; max is %v",
			string(chunkSize), string(MaxChunkSize))
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
	return SendHeader(client, PatchFileCompleteType)
}

// We've finished updating files.
func SendUpdateComplete(client *PatchClient) int {
	if config.DebugMode {
		fmt.Println("Sending File Update Done")
	}
	return SendHeader(client, UpdateCompleteType)
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

func init() {
	patchCopyrightBytes = []byte(patchCopyright)
	loginCopyrightBytes = []byte(loginCopyright)
}
