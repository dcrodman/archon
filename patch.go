/*
* Archon PSO Server
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
* The PATCH and DATA server logic. Both are included here since they're
* neither are particularly complicated.
 */
package main

import (
	"errors"
	"fmt"
	crypto "github.com/dcrodman/archon/encryption"
	"github.com/dcrodman/archon/util"
	"hash/crc32"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
)

// MaxFileChunkSize is the maximum number of bytes we can send of a file at a time.
const MaxFileChunkSize = 24576

// PatchEntry instances contain metadata about each of the files in the patches directory.
type PatchEntry struct {
	filename string
	// Path relative to the patch dir for convenience.
	relativePath string
	pathDirs     []string
	index        uint32
	checksum     uint32
	fileSize     uint32
}

// PatchDir is a tree structure for holding patch data that more closely represents
// a file hierarchy and makes it easier to handle the client working dir. Patch files and
// subdirectories are represented as lists in order to make a breadth-first search easier
// and the order predictable.
type PatchDir struct {
	dirname string
	patches []*PatchEntry
	subdirs []*PatchDir
}

// Create and initialize a new Patch client so long as we're able
// to send the welcome packet to begin encryption.
func NewPatchClient(conn *net.TCPConn) (*Client, error) {
	var err error
	cCrypt := crypto.NewPCCrypt()
	sCrypt := crypto.NewPCCrypt()
	pc := NewClient(conn, PCHeaderSize, cCrypt, sCrypt)
	if pc.SendPCWelcome() != 0 {
		err = errors.New("Error sending welcome packet to: " + pc.IPAddr())
		pc = nil
	}
	return pc, err
}

// PatchServer is the sub-server that acts as the first point of contact for a client. Its
// only real job is to send the client a welcome message and then send the address of DataServer.
type PatchServer struct {
	// Parsed representation of the login port.
	dataRedirectPort uint16
}

func (server PatchServer) Name() string { return "PATCH" }

func (server PatchServer) Port() string { return config.PatchPort }

func (server *PatchServer) Init() error {
	// Convert the data port to a BE uint for the redirect packet.
	dataPort, _ := strconv.ParseUint(config.DataPort, 10, 16)
	server.dataRedirectPort = uint16((dataPort >> 8) | (dataPort << 8))
	return nil
}

func (server *PatchServer) NewClient(conn *net.TCPConn) (*Client, error) {
	return NewPatchClient(conn)
}

func (server *PatchServer) Handle(c *Client) error {
	var hdr PCHeader
	util.StructFromBytes(c.Data()[:PCHeaderSize], &hdr)

	var err error
	switch hdr.Type {
	case PatchWelcomeType:
		err = server.sendWelcomeAck(c)
	case PatchLoginType:
		if err := server.sendWelcomeMessage(c); err == nil {
			err = server.sendPatchRedirect(c)
		}
	default:
		log.Infof("Received unknown packet %2x from %s", hdr.Type, c.IPAddr())
	}
	return err
}

func (server *PatchServer) sendWelcomeAck(client *Client) error {
	// PatchLoginType is treated as an ack in this case.
	pkt := &PCHeader{
		Size: 0x04,
		Type: PatchLoginType,
	}
	data, _ := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Welcome Ack")
	}
	return client.SendEncrypted(data, 0x04)
}

// Message displayed on the patch download screen.
func (server *PatchServer) sendWelcomeMessage(client *Client) error {
	pkt := new(PatchWelcomeMessage)
	pkt.Header = PCHeader{Size: PCHeaderSize + config.MessageSize, Type: PatchMessageType}
	pkt.Message = config.MessageBytes

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Welcome Message")
	}
	return client.SendEncrypted(data, uint16(size))
}

// Send the redirect packet, providing the IP and port of the next server.
func (server *PatchServer) sendPatchRedirect(client *Client) error {
	pkt := new(PatchRedirectPacket)
	pkt.Header.Type = PatchRedirectType
	pkt.Port = server.dataRedirectPort

	hostnameBytes := config.HostnameBytes()
	copy(pkt.IPAddr[:], hostnameBytes[:])

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Patch Redirect")
	}
	return client.SendEncrypted(data, uint16(size))
}

// Data sub-server definition.
type DataServer struct {
	// File names that should be ignored when searching for patch files.
	SkipPaths []string

	// Each index corresponds to a patch file. This is constructed in the order
	// that the patch tree will be traversed and makes it faster to locate a
	// patch entry when the client sends us an index in the FileStatusPacket.
	patchTree  PatchDir
	patchIndex []*PatchEntry
}

func (server DataServer) Name() string { return "DATA" }

func (server DataServer) Port() string { return config.DataPort }

func (server *DataServer) Init() error {
	server.SkipPaths = []string{".", "..", ".DS_Store", ".rid"}

	wd, _ := os.Getwd()
	if err := os.Chdir(config.PatchDir); err != nil {
		return errors.New("Unable to cd to patches directory: " + err.Error())
	}

	// Construct our patch tree from the specified directory.
	fmt.Printf("Loading patches from %s...\n", config.PatchDir)
	if err := server.loadPatches(&server.patchTree, "."); err != nil {
		return errors.New("Failed to load patches: " + err.Error())
	}
	server.buildPatchIndex(&server.patchTree)
	if len(server.patchIndex) < 1 {
		return errors.New("Failed: At least one patch file must be present.")
	}
	os.Chdir(wd)

	fmt.Println()
	return nil
}

// Recursively build the list of patch files present in the patch directory
// to sync with the client. Files are represented in a tree, directories act
// as nodes (PatchDir) and each keeps a list of patches/subdirectories.
func (server *DataServer) loadPatches(node *PatchDir, path string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Printf("Couldn't parse %s\n", path)
		return err
	}
	dirs := strings.Split(path, "/")
	node.dirname = dirs[len(dirs)-1]

	for _, file := range files {
		filename := file.Name()
		skip := false
		for _, path := range server.SkipPaths {
			if filename == path {
				skip = true
				break
			}
		}

		if skip {
			continue
		} else if file.IsDir() {
			subdir := new(PatchDir)
			node.subdirs = append(node.subdirs, subdir)
			server.loadPatches(subdir, path+"/"+filename)
		} else {
			data, err := ioutil.ReadFile(path + "/" + filename)
			if err != nil {
				return err
			}
			patch := &PatchEntry{
				filename:     filename,
				relativePath: config.PatchDir + "/" + path + "/" + filename,
				pathDirs:     dirs,
				fileSize:     uint32(file.Size()),
				checksum:     crc32.ChecksumIEEE(data),
			}

			node.patches = append(node.patches, patch)
			fmt.Printf("%s (%d bytes, checksum: %v)\n",
				path+"/"+filename, patch.fileSize, patch.checksum)
		}
	}
	return nil
}

// Build the patch index, performing a depth-first search and mapping
// each patch entry to an array so that they're quickly indexable when
// we need to look up the patch data.
func (server *DataServer) buildPatchIndex(node *PatchDir) {
	for _, dir := range node.subdirs {
		server.buildPatchIndex(dir)
	}
	for _, patch := range node.patches {
		server.patchIndex = append(server.patchIndex, patch)
		patch.index = uint32(len(server.patchIndex) - 1)
	}
}

func (server DataServer) NewClient(conn *net.TCPConn) (*Client, error) {
	return NewPatchClient(conn)
}

func (server DataServer) Handle(c *Client) error {
	var hdr PCHeader
	util.StructFromBytes(c.Data()[:PCHeaderSize], &hdr)

	var err error
	switch hdr.Type {
	case PatchWelcomeType:
		err = server.sendWelcomeAck(c)
	case PatchLoginType:
		err = server.HandlePatchLogin(c)
	case PatchFileStatusType:
		server.HandleFileStatus(c)
	case PatchClientListDoneType:
		err = server.UpdateClientFiles(c)
	default:
		log.Infof("Received unknown packet %02x from %s", hdr.Type, c.IPAddr())
	}
	return err
}

func (server *DataServer) sendWelcomeAck(client *Client) error {
	// PatchLoginType is treated as an ack in this case.
	pkt := &PCHeader{
		Size: 0x04,
		Type: PatchLoginType,
	}
	data, _ := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Welcome Ack")
	}
	return client.SendEncrypted(data, 0x04)
}

// Once the client has authenticated, send them the list of files to update.
func (server *DataServer) HandlePatchLogin(c *Client) error {
	server.sendDataAck(c)
	server.sendFileList(c, &server.patchTree)
	server.sendFileListDone(c)
	return nil
}

// Acknowledgement sent after the DATA connection handshake.
func (server *DataServer) sendDataAck(client *Client) int {
	pkt := &PCHeader{Type: PatchDataAckType, Size: 0x04}
	data, size := util.BytesFromStruct(pkt)

	if config.DebugMode {
		fmt.Println("Sending Data Ack")
	}
	return sendEncrypted(client, data, uint16(size))
}

// Traverse the patch tree depth-first and send the check file requests.
func (server *DataServer) sendFileList(client *Client, node *PatchDir) error {
	// Step into the next directory.
	server.sendChangeDir(client, node.dirname)
	for _, subdir := range node.subdirs {
		server.sendFileList(client, subdir)
		// Move them back up each time we leave a directory.
		server.sendDirAbove(client)
	}
	for _, patch := range node.patches {
		server.sendCheckFile(client, patch.index, patch.filename)
	}
	return nil
}

// Tell the client to change to some directory within its file tree.
func (server *DataServer) sendChangeDir(client *Client, dir string) int {
	pkt := new(ChangeDirPacket)
	pkt.Header.Type = PatchChangeDirType
	copy(pkt.Dirname[:], dir)

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Change Directory")
	}
	return sendEncrypted(client, data, uint16(size))
}

// Tell the client to change to one directory above.
func (server *DataServer) sendDirAbove(client *Client) int {
	pkt := &PCHeader{Type: PatchDirAboveType, Size: 0x04}
	data, size := util.BytesFromStruct(pkt)

	if config.DebugMode {
		fmt.Println("Sending Dir Above")
	}
	return sendEncrypted(client, data, uint16(size))
}

// Inform the client that we've finished sending the patch list.
func (server *DataServer) sendFileListDone(client *Client) int {
	pkt := &PCHeader{Type: PatchFileListDoneType, Size: 0x04}
	data, size := util.BytesFromStruct(pkt)

	if config.DebugMode {
		fmt.Println("Sending List Done")
	}
	return sendEncrypted(client, data, uint16(size))
}

// Tell the client to check a file in its current working directory.
func (server *DataServer) sendCheckFile(client *Client, index uint32, filename string) int {
	pkt := new(CheckFilePacket)
	pkt.Header.Type = PatchCheckFileType
	pkt.PatchId = index
	copy(pkt.Filename[:], filename)

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Check File")
	}
	return sendEncrypted(client, data, uint16(size))
}

// The client sent us a checksum for one of the patch files. Compare it to what we
// have and add it to the list of files to update if there is any discrepancy.
func (server *DataServer) HandleFileStatus(client *Client) {
	var fileStatus FileStatusPacket
	util.StructFromBytes(client.Data(), &fileStatus)

	patch := server.patchIndex[fileStatus.PatchId]
	if fileStatus.Checksum != patch.checksum || fileStatus.FileSize != patch.fileSize {
		client.updateList = append(client.updateList, patch)
	}
}

// The client finished sending all of the file check packets. If they have
// any files that need updating, now's the time to do it.
func (server *DataServer) UpdateClientFiles(client *Client) error {
	var numFiles, totalSize uint32 = 0, 0
	for _, patch := range client.updateList {
		numFiles++
		totalSize += patch.fileSize
	}

	// Send files, if we have any.
	if numFiles > 0 {
		server.sendUpdateFiles(client, numFiles, totalSize)
		server.sendChangeDir(client, ".")
		chunkBuf := make([]byte, MaxFileChunkSize)

		for _, patch := range client.updateList {
			// Descend into the correct directory if needed.
			ascendCtr := 0
			for i := 1; i < len(patch.pathDirs); i++ {
				ascendCtr++
				server.sendChangeDir(client, patch.pathDirs[i])
			}
			server.sendFileHeader(client, patch)

			// Divide the file into chunks and send each one.
			chunks := int((patch.fileSize / MaxFileChunkSize) + 1)
			file, err := os.Open(patch.relativePath)
			if err != nil {
				// Critical since this is most likely a filesystem error.
				log.Error(err.Error())
				return err
			}
			for i := 0; i < chunks; i++ {
				bytes, err := file.ReadAt(chunkBuf, int64(MaxFileChunkSize*i))
				if err != nil && err != io.EOF {
					return err
				}
				chksm := crc32.ChecksumIEEE(chunkBuf)
				server.sendFileChunk(client, uint32(i), chksm, uint32(bytes), chunkBuf)
			}

			server.sendFileComplete(client)
			// Change back to the top level directory.
			for ascendCtr > 0 {
				ascendCtr--
				server.sendDirAbove(client)
			}
		}
	}
	server.sendUpdateComplete(client)
	return nil
}

// Send the total number and cumulative size of files that need updating.
func (server *DataServer) sendUpdateFiles(client *Client, num, totalSize uint32) int {
	pkt := new(UpdateFilesPacket)
	pkt.Header.Type = PatchUpdateFilesType
	pkt.NumFiles = num
	pkt.TotalSize = totalSize

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Update Files")
	}
	return sendEncrypted(client, data, uint16(size))
}

// Send the header for a file we're about to update.
func (server *DataServer) sendFileHeader(client *Client, patch *PatchEntry) int {
	pkt := new(FileHeaderPacket)
	pkt.Header.Type = PatchFileHeaderType
	pkt.FileSize = patch.fileSize
	copy(pkt.Filename[:], patch.filename)

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending File Header")
	}
	return sendEncrypted(client, data, uint16(size))
}

// Send a chunk of file data.
func (server *DataServer) sendFileChunk(client *Client, chunk, chksm, chunkSize uint32, fdata []byte) int {
	if chunkSize > MaxFileChunkSize {
		log.Error("Attempted to send %v byte chunk; max is %v",
			string(chunkSize), string(MaxFileChunkSize))
		panic(errors.New("File chunk size exceeds maximum"))
	}
	pkt := &FileChunkPacket{
		Header:   PCHeader{Type: PatchFileChunkType},
		Chunk:    chunk,
		Checksum: chksm,
		Size:     chunkSize,
		Data:     fdata[:chunkSize],
	}

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending File Chunk")
	}
	return sendEncrypted(client, data, uint16(size))
}

// Finished sending a particular file.
func (server *DataServer) sendFileComplete(client *Client) error {
	pkt := &PCHeader{Type: PatchFileCompleteType, Size: 0x04}
	data, size := util.BytesFromStruct(pkt)

	if config.DebugMode {
		fmt.Println("Sending File Complete")
	}
	sendEncrypted(client, data, uint16(size))
	return nil
}

// We've finished updating files.
func (server *DataServer) sendUpdateComplete(client *Client) int {
	pkt := &PCHeader{Type: PatchUpdateCompleteType, Size: 0x04}
	data, size := util.BytesFromStruct(pkt)

	if config.DebugMode {
		fmt.Println("Sending File Update Done")
	}
	return sendEncrypted(client, data, uint16(size))
}
