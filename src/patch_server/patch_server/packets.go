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
package patch_server

import (
	"errors"
	"fmt"
	"libarchon/logger"
	"libarchon/util"
)

const PCHeaderSize = 0x04
const bbCopyright = "Patch Server. Copyright SonicTeam, LTD. 2001"

var copyrightBytes []byte

// Packet types for packets sent to and from the login and character servers.
const (
	WelcomeType        = 0x02
	WelcomeAckType     = 0x04 // sent
	LoginType          = 0x04 // received
	MessageType        = 0x13
	RedirectType       = 0x14
	DataAckType        = 0x0B
	SetDirAboveType    = 0x0A
	ChangeDirType      = 0x09
	CheckFileType      = 0x0C
	FileListDoneType   = 0x0D
	FileStatusType     = 0x0F
	ClientListDoneType = 0x10
	UpdateFilesType    = 0x11
	FileHeaderType     = 0x06
	FileChunkType      = 0x07
	FileCompleteType   = 0x08
	UpdateCompleteType = 0x12
)

// Blueburst, PC, and Gamecube clients all use a 4 byte header to communicate with the
// patch server instead of the 8 byte one used by Blueburst for the other servers.
type PCPktHeader struct {
	Size uint16
	Type uint16
}

// Welcome packet with encryption vectors sent to the client upon initial connection.
type WelcomePkt struct {
	Header       PCPktHeader
	Copyright    [44]byte
	Padding      [20]byte
	ServerVector [4]byte
	ClientVector [4]byte
}

// Packet containing the patch server welcome message.
type WelcomeMessage struct {
	Header  PCPktHeader
	Message []byte
}

// The address of the next server; in this case, the character server.
type RedirectPacket struct {
	Header  PCPktHeader
	IPAddr  [4]uint8
	Port    uint16
	Padding uint16
}

// Instruct the client to chdir into Dirname (one level below).
type ChangeDirPacket struct {
	Header  PCPktHeader
	Dirname [64]byte
}

// Request a check on a file in the client's working directory.
type CheckFilePacket struct {
	Header   PCPktHeader
	PatchId  uint32
	Filename [32]byte
}

// Response to CheckFilePacket from the client with the properties of a file.
type FileStatusPacket struct {
	Header   PCPktHeader
	PatchId  uint32
	Checksum uint32
	FileSize uint32
}

// Size and number of files that need to be updated.
type UpdateFilesPacket struct {
	Header    PCPktHeader
	TotalSize uint32
	NumFiles  uint32
}

// File header for a series of file chunks.
type FileHeaderPacket struct {
	Header   PCPktHeader
	Padding  uint32
	FileSize uint32
	Filename [48]byte
}

// Chunk of data from a file.
type FileChunkPacket struct {
	Header   PCPktHeader
	Chunk    uint32
	Checksum uint32
	Size     uint32
	Data     []byte
}

// Send the packet serialized (or otherwise contained) in pkt to a client.
// Note: Packets sent to BB Clients must have a length divisible by 8.
func SendPacket(client *PatchClient, pkt []byte, length uint16) int {
	if err := client.c.Send(pkt[:length]); err != nil {
		log.Info("Error sending to client "+client.IPAddr()+": "+err.Error(),
			logger.MediumPriority)
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
	client.c.Encrypt(data, uint32(length))
	return SendPacket(client, data, length)
}

// Send a simple 4-byte header packet.
func SendHeader(client *PatchClient, pktType uint16) int {
	pkt := new(PCPktHeader)
	pkt.Type = pktType
	pkt.Size = 0x04
	data, size := util.BytesFromStruct(pkt)
	return SendEncrypted(client, data, uint16(size))
}

// Send the welcome packet to a client with the copyright message and encryption vectors.
func SendWelcome(client *PatchClient) int {
	pkt := new(WelcomePkt)
	pkt.Header.Type = WelcomeType
	pkt.Header.Size = 0x4C
	copy(pkt.Copyright[:], copyrightBytes)
	copy(pkt.ClientVector[:], client.c.ClientVector())
	copy(pkt.ServerVector[:], client.c.ServerVector())

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Welcome Packet")
		util.PrintPayload(data, size)
		fmt.Println()
	}
	return SendPacket(client, data, uint16(size))
}

func SendWelcomeAck(client *PatchClient) int {
	pkt := new(PCPktHeader)
	pkt.Size = 0x04
	pkt.Type = WelcomeAckType
	data, _ := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Welcome Ack")
	}
	return SendEncrypted(client, data, 0x0004)
}

func SendWelcomeMessage(client *PatchClient) int {
	cfg := GetConfig()
	pkt := new(WelcomeMessage)
	pkt.Header.Type = MessageType
	pkt.Header.Size = PCHeaderSize + cfg.MessageSize
	pkt.Message = cfg.MessageBytes

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Welcome Message")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the redirect packet, providing the IP and port of the next server.
func SendRedirect(client *PatchClient, port uint16, ipAddr [4]byte) int {
	pkt := new(RedirectPacket)
	pkt.Header.Type = RedirectType
	copy(pkt.IPAddr[:], ipAddr[:])
	pkt.Port = port

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Redirect")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Acknowledgement sent after the DATA connection handshake.
func SendDataAck(client *PatchClient) int {
	if GetConfig().DebugMode {
		fmt.Println("Sending Data Ack")
	}
	return SendHeader(client, DataAckType)
}

// Tell the client to change to one directory above.
func SendDirAbove(client *PatchClient) int {
	if GetConfig().DebugMode {
		fmt.Println("Sending Dir Above")
	}
	return SendHeader(client, SetDirAboveType)
}

// Tell the client to change to some directory within its file tree.
func SendChangeDir(client *PatchClient, dir string) int {
	pkt := new(ChangeDirPacket)
	pkt.Header.Type = ChangeDirType
	copy(pkt.Dirname[:], dir)

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Change Directory")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Tell the client to check a file in its current working directory.
func SendCheckFile(client *PatchClient, index uint32, filename string) int {
	pkt := new(CheckFilePacket)
	pkt.Header.Type = CheckFileType
	pkt.PatchId = index
	copy(pkt.Filename[:], filename)

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Check File")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Inform the client that we've finished sending the patch list.
func SendFileListDone(client *PatchClient) int {
	if GetConfig().DebugMode {
		fmt.Println("Sending List Done")
	}
	return SendHeader(client, FileListDoneType)
}

// Send the total number and cumulative size of files that need updating.
func SendUpdateFiles(client *PatchClient, num, totalSize uint32) int {
	pkt := new(UpdateFilesPacket)
	pkt.Header.Type = UpdateFilesType
	pkt.NumFiles = num
	pkt.TotalSize = totalSize

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending Update Files")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send the header for a file we're about to update.
func SendFileHeader(client *PatchClient, patch *PatchEntry) int {
	pkt := new(FileHeaderPacket)
	pkt.Header.Type = FileHeaderType
	pkt.FileSize = patch.fileSize
	copy(pkt.Filename[:], patch.filename)

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending File Header")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Send a chunk of file data.
func SendFileChunk(client *PatchClient, chunk, chksm, chunkSize uint32, fdata []byte) int {
	if chunkSize > MaxChunkSize {
		panic(errors.New("File chunk size exceeds maximum"))
		log.Error("Attempted to send "+string(chunkSize)+
			" byte chunk; max is "+string(MaxChunkSize),
			logger.CriticalPriority)
	}
	pkt := new(FileChunkPacket)
	pkt.Header.Type = FileChunkType
	pkt.Chunk = chunk
	pkt.Checksum = chksm
	pkt.Size = chunkSize
	pkt.Data = fdata[:chunkSize]

	data, size := util.BytesFromStruct(pkt)
	if GetConfig().DebugMode {
		fmt.Println("Sending File Chunk")
	}
	return SendEncrypted(client, data, uint16(size))
}

// Finished sending a particular file.
func SendFileComplete(client *PatchClient) int {
	if GetConfig().DebugMode {
		fmt.Println("Sending File Complete")
	}
	return SendHeader(client, FileCompleteType)
}

// We've finished updating files.
func SendUpdateComplete(client *PatchClient) int {
	if GetConfig().DebugMode {
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
	copyrightBytes = []byte(bbCopyright)
}
