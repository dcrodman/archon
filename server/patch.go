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
package server

import (
	"errors"
	"fmt"
	"github.com/dcrodman/archon/server/util"
	"hash/crc32"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
)

var (
	// Parsed representation of the login port.
	dataRedirectPort uint16

	// File names that should be ignored when searching for patch files.
	SkipPaths = []string{".", "..", ".DS_Store", ".rid"}

	// Each index corresponds to a patch file. This is constructed in the order
	// that the patch tree will be traversed and makes it faster to locate a
	// patch entry when the client sends us an index in the FileStatusPacket.
	patchTree  PatchDir
	patchIndex []*PatchEntry
)

const MaxFileChunkSize = 24576

// Data for one patch file.
type PatchEntry struct {
	filename string
	// Path relative to the patch dir for convenience.
	relativePath string
	pathDirs     []string
	index        uint32
	checksum     uint32
	fileSize     uint32
}

// Basic tree structure for holding patch data that more closely represents
// a file hierarchy and makes it easier to handle the client working dir.
// Patch files and subdirectories are represented as lists in order to make
// a breadth-first search easier and the order predictable.
type PatchDir struct {
	dirname string
	patches []*PatchEntry
	subdirs []*PatchDir
}

// Traverse the patch tree depth-first and send the check file requests.
func sendFileList(client *Client, node *PatchDir) {
	// Step into the next directory.
	client.SendChangeDir(node.dirname)
	for _, subdir := range node.subdirs {
		sendFileList(client, subdir)
		// Move them back up each time we leave a directory.
		client.SendDirAbove()
	}
	for _, patch := range node.patches {
		client.SendCheckFile(patch.index, patch.filename)
	}
}

// The client sent us a checksum for one of the patch files. Compare it
// to what we have and add it to the list of files to update if there
// is any discrepancy.
func handleFileStatus(client *Client) {
	var fileStatus FileStatusPacket
	util.StructFromBytes(client.Data(), &fileStatus)

	patch := patchIndex[fileStatus.PatchId]
	if fileStatus.Checksum != patch.checksum || fileStatus.FileSize != patch.fileSize {
		client.updateList = append(client.updateList, patch)
	}
}

// The client finished sending all of the file check packets. If they have
// any files that need updating, now's the time to do it.
func updateClientFiles(client *Client) error {
	var numFiles, totalSize uint32 = 0, 0
	for _, patch := range client.updateList {
		numFiles++
		totalSize += patch.fileSize
	}

	// Send files, if we have any.
	if numFiles > 0 {
		client.SendUpdateFiles(numFiles, totalSize)
		client.SendChangeDir(".")
		chunkBuf := make([]byte, MaxFileChunkSize)

		for _, patch := range client.updateList {
			// Descend into the correct directory if needed.
			ascendCtr := 0
			for i := 1; i < len(patch.pathDirs); i++ {
				ascendCtr++
				client.SendChangeDir(patch.pathDirs[i])
			}
			client.SendFileHeader(patch)

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
				client.SendFileChunk(uint32(i), chksm, uint32(bytes), chunkBuf)
			}

			client.SendFileComplete()
			// Change back to the top level directory.
			for ascendCtr > 0 {
				ascendCtr--
				client.SendDirAbove()
			}
		}
	}
	client.SendUpdateComplete()
	return nil
}

// Create and initialize a new struct to hold client information.
func NewPatchClient(conn *net.TCPConn) (*Client, error) {
	var err error
	pc := NewClient(conn, PCHeaderSize)
	if pc.SendPCWelcome() != 0 {
		err = errors.New("Error sending welcome packet to: " + pc.IPAddr())
		pc = nil
	}
	return pc, err
}

// Handle communication with a PATCH client until the connection
// is closed or an error is encountered.
func PatchHandler(pc *Client) {
	var pktHeader PCHeader
	for {
		err := pc.Process()
		if err == io.EOF {
			break
		} else if err != nil {
			// Error communicating with the client.
			log.Warn(err.Error())
			break
		}

		util.StructFromBytes(pc.Data()[:PCHeaderSize], &pktHeader)
		if config.DebugMode {
			fmt.Printf("PATCH: Got %v bytes from client:\n", pktHeader.Size)
			util.PrintPayload(pc.Data(), int(pktHeader.Size))
			fmt.Println()
		}

		switch pktHeader.Type {
		case PatchWelcomeType:
			pc.SendWelcomeAck()
		case PatchLoginType:
			if pc.SendWelcomeMessage() == 0 {
				pc.SendPatchRedirect(dataRedirectPort, config.HostnameBytes())
			}
		default:
			log.Info("Received unknown packet %2x from %s", pktHeader.Type, pc.IPAddr())
		}

		if err != nil {
			log.Warn("Error in client communication: " + err.Error())
			return
		}
	}
}

// Handle communication with a DATA client until the connection
// is closed or an error is encountered.
func DataHandler(pc *Client) {
	var pktHeader PCHeader
	for {
		err := pc.Process()
		if err == io.EOF {
			break
		} else if err != nil {
			// Error communicating with the client.
			log.Warn(err.Error())
			break
		}

		util.StructFromBytes(pc.Data()[:PCHeaderSize], &pktHeader)
		if config.DebugMode {
			fmt.Printf("DATA: Got %v bytes from client:\n", pktHeader.Size)
			util.PrintPayload(pc.Data(), int(pktHeader.Size))
			fmt.Println()
		}

		switch pktHeader.Type {
		case PatchWelcomeType:
			pc.SendWelcomeAck()
		case PatchLoginType:
			pc.SendDataAck()
			sendFileList(pc, &patchTree)
			pc.SendFileListDone()
		case PatchFileStatusType:
			handleFileStatus(pc)
		case PatchClientListDoneType:
			if updateClientFiles(pc) != nil {
				return
			}
		default:
			log.Info("Received unknown packet %02x from %s", pktHeader.Type, pc.IPAddr())
		}
	}
}

// Recursively build the list of patch files present in the patch directory
// to sync with the client. Files are represented in a tree, directories act
// as nodes (PatchDir) and each keeps a list of patches/subdirectories.
func loadPatches(node *PatchDir, path string) error {
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
		for _, path := range SkipPaths {
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
			loadPatches(subdir, path+"/"+filename)
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
func buildPatchIndex(node *PatchDir) {
	for _, dir := range node.subdirs {
		buildPatchIndex(dir)
	}
	for _, patch := range node.patches {
		patchIndex = append(patchIndex, patch)
		patch.index = uint32(len(patchIndex) - 1)
	}
}

func InitPatch() {
	wd, _ := os.Getwd()
	os.Chdir(config.PatchDir)

	// Construct our patch tree from the specified directory.
	fmt.Printf("Loading patches from %s...\n", config.PatchDir)
	if err := loadPatches(&patchTree, "."); err != nil {
		fmt.Printf("Failed to load patches: %s\n", err.Error())
		os.Exit(1)
	}
	buildPatchIndex(&patchTree)
	if len(patchIndex) < 1 {
		fmt.Println("Failed: At least one patch file must be present.")
		os.Exit(1)
	}
	os.Chdir(wd)

	// Convert the data port to a BE uint for the redirect packet.
	dataPort, _ := strconv.ParseUint(config.DataPort, 10, 16)
	dataRedirectPort = uint16((dataPort >> 8) | (dataPort << 8))
	fmt.Println()
}
