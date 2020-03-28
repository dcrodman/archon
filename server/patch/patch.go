/*
* The PATCH and DATA server logic. Both are included here since they're
* neither are particularly complicated.
 */
package patch

import (
	"errors"
	"fmt"
	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/server"
	"hash/crc32"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/dcrodman/archon/util"
	crypto "github.com/dcrodman/archon/util/encryption"
)

// MaxFileChunkSize is the maximum number of bytes we can send of a file at a time.
const MaxFileChunkSize = 24576

// Copyright message expected by the client for the patch welcome.
var PatchCopyright = []byte("Patch Server. Copyright SonicTeam, LTD. 2001")

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
func NewPatchClient(conn *net.TCPConn) (*server.Client, error) {
	var err error
	cCrypt := crypto.NewPCCrypt()
	sCrypt := crypto.NewPCCrypt()
	pc := server.NewClient(conn, archon.PCHeaderSize, cCrypt, sCrypt)
	if SendPCWelcome(pc) != nil {
		err = errors.New("Error sending welcome packet to: " + pc.IPAddr())
		pc = nil
	}
	return pc, err
}

// Send the welcome packet to a client with the copyright message and encryption vectors.
func SendPCWelcome(client *server.Client) error {
	pkt := new(archon.PatchWelcomePkt)
	pkt.Header.Type = archon.PatchWelcomeType
	pkt.Header.Size = 0x4C
	copy(pkt.Copyright[:], PatchCopyright)
	copy(pkt.ClientVector[:], client.ClientVector())
	copy(pkt.ServerVector[:], client.ServerVector())

	data, size := util.BytesFromStruct(pkt)
	if archon.Config.DebugMode {
		fmt.Println("Sending Welcome Packet")
		util.PrintPayload(data, size)
		fmt.Println()
	}
	return client.SendRaw(data, size)
}

// PatchServer is the sub-server that acts as the first point of contact for a client. Its
// only real job is to send the client a welcome message and then send the address of DataServer.
type PatchServer struct {
	// Parsed representation of the login port.
	dataRedirectPort uint16
}

func NewServer() *PatchServer {
	return &PatchServer{}
}

func (server PatchServer) Name() string { return "PATCH" }

func (server PatchServer) Port() string { return archon.Config.PatchServer.PatchPort }

func (server *PatchServer) Init() error {
	// Convert the data port to a BE uint for the redirect packet.
	dataPort, _ := strconv.ParseUint(archon.Config.PatchServer.DataPort, 10, 16)
	server.dataRedirectPort = uint16((dataPort >> 8) | (dataPort << 8))
	return nil
}

func (server *PatchServer) NewClient(conn *net.TCPConn) (*server.Client, error) {
	return NewPatchClient(conn)
}

func (server *PatchServer) Handle(c *server.Client) error {
	var hdr archon.PCHeader
	util.StructFromBytes(c.Data()[:archon.PCHeaderSize], &hdr)

	var err error
	switch hdr.Type {
	case archon.PatchWelcomeType:
		err = server.sendWelcomeAck(c)
	case archon.PatchLoginType:
		if err := server.sendWelcomeMessage(c); err == nil {
			err = server.sendPatchRedirect(c)
		}
	default:
		archon.Log.Infof("Received unknown packet %2x from %s", hdr.Type, c.IPAddr())
	}
	return err
}

func (server *PatchServer) sendWelcomeAck(client *server.Client) error {
	// PatchLoginType is treated as an ack in this case.
	pkt := &archon.PCHeader{
		Size: 0x04,
		Type: archon.PatchLoginType,
	}

	archon.Log.Debug("Sending Welcome Ack")
	data, _ := util.BytesFromStruct(pkt)
	return client.SendEncrypted(data, 0x04)
}

// Message displayed on the patch download screen.
func (server *PatchServer) sendWelcomeMessage(client *server.Client) error {
	pkt := new(archon.PatchWelcomeMessage)
	pkt.Header = archon.PCHeader{Size: archon.PCHeaderSize + archon.MessageSize, Type: archon.PatchMessageType}
	pkt.Message = archon.MessageBytes

	archon.Log.Debug("Sending Welcome Message")
	return archon.EncryptAndSend(client, pkt)
}

// Send the redirect packet, providing the IP and port of the next server.
func (server *PatchServer) sendPatchRedirect(client *server.Client) error {
	pkt := new(archon.PatchRedirectPacket)
	pkt.Header.Type = archon.PatchRedirectType
	pkt.Port = server.dataRedirectPort
	hostnameBytes := archon.BroadcastIP()
	copy(pkt.IPAddr[:], hostnameBytes[:])

	archon.Log.Debug("Sending Patch Redirect")
	return archon.EncryptAndSend(client, pkt)
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

func NewDataServer() *DataServer {
	return &DataServer{}
}

func (server DataServer) Name() string { return "DATA" }

func (server DataServer) Port() string { return archon.Config.PatchServer.DataPort }

func (server *DataServer) Init() error {
	server.SkipPaths = []string{".", "..", ".DS_Store", ".rid"}

	wd, _ := os.Getwd()
	if err := os.Chdir(archon.Config.PatchServer.PatchDir); err != nil {
		return errors.New("Unable to cd to patches directory: " + err.Error())
	}

	// Construct our patch tree from the specified directory.
	fmt.Printf("Loading patches from %s...\n", archon.Config.PatchServer.PatchDir)
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
				relativePath: archon.Config.PatchServer.PatchDir + "/" + path + "/" + filename,
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

func (server DataServer) NewClient(conn *net.TCPConn) (*server.Client, error) {
	return NewPatchClient(conn)
}

func (server DataServer) Handle(c *server.Client) error {
	var hdr archon.PCHeader
	util.StructFromBytes(c.Data()[:archon.PCHeaderSize], &hdr)

	var err error
	switch hdr.Type {
	case archon.PatchWelcomeType:
		err = server.sendWelcomeAck(c)
	case archon.PatchLoginType:
		err = server.HandlePatchLogin(c)
	case archon.PatchFileStatusType:
		server.HandleFileStatus(c)
	case archon.PatchClientListDoneType:
		err = server.UpdateClientFiles(c)
	default:
		archon.Log.Infof("Received unknown packet %02x from %s", hdr.Type, c.IPAddr())
	}
	return err
}

// Simple acknowledgement to the welcome response.
func (server *DataServer) sendWelcomeAck(client *server.Client) error {
	// PatchLoginType is treated as an ack in this case.
	pkt := &archon.PCHeader{
		Size: 0x04,
		Type: archon.PatchLoginType,
	}
	archon.Log.Debug("Sending Welcome Ack")
	data, _ := util.BytesFromStruct(pkt)
	return client.SendEncrypted(data, 0x04)
}

// Once the client has authenticated, send them the list of files to update.
func (server *DataServer) HandlePatchLogin(c *server.Client) error {
	if err := server.sendDataAck(c); err != nil {
		return err
	}
	if err := server.sendFileList(c, &server.patchTree); err != nil {
		return err
	}
	return server.sendFileListDone(c)
}

// Acknowledgement sent after the DATA connection handshake.
func (server *DataServer) sendDataAck(client *server.Client) error {
	pkt := &archon.PCHeader{Type: archon.PatchDataAckType, Size: 0x04}
	archon.Log.Debug("Sending Data Ack")
	return archon.EncryptAndSend(client, pkt)
}

// Traverse the patch tree depth-first and send the check file requests.
func (server *DataServer) sendFileList(client *server.Client, node *PatchDir) error {
	// Step into the next directory.
	if err := server.sendChangeDir(client, node.dirname); err != nil {
		return err
	}

	for _, subdir := range node.subdirs {
		if err := server.sendFileList(client, subdir); err != nil {
			return err
		}
		// Move them back up each time we leave a directory.
		if err := server.sendDirAbove(client); err != nil {
			return err
		}
	}
	for _, patch := range node.patches {
		if err := server.sendCheckFile(client, patch.index, patch.filename); err != nil {
			return err
		}
	}
	return nil
}

// Tell the client to change to some directory within its file tree.
func (server *DataServer) sendChangeDir(client *server.Client, dir string) error {
	pkt := new(archon.ChangeDirPacket)
	pkt.Header.Type = archon.PatchChangeDirType
	copy(pkt.Dirname[:], dir)

	archon.Log.Debug("Sending Change Directory")
	return archon.EncryptAndSend(client, pkt)
}

// Tell the client to change to one directory above.
func (server *DataServer) sendDirAbove(client *server.Client) error {
	pkt := &archon.PCHeader{Type: archon.PatchDirAboveType, Size: 0x04}
	archon.Log.Debug("Sending Dir Above")
	return archon.EncryptAndSend(client, pkt)
}

// Inform the client that we've finished sending the patch list.
func (server *DataServer) sendFileListDone(client *server.Client) error {
	pkt := &archon.PCHeader{Type: archon.PatchFileListDoneType, Size: 0x04}
	archon.Log.Debug("Sending List Done")
	return archon.EncryptAndSend(client, pkt)
}

// Tell the client to check a file in its current working directory.
func (server *DataServer) sendCheckFile(client *server.Client, index uint32, filename string) error {
	pkt := new(archon.CheckFilePacket)
	pkt.Header.Type = archon.PatchCheckFileType
	pkt.PatchId = index
	copy(pkt.Filename[:], filename)

	archon.Log.Debug("Sending Check File")
	return archon.EncryptAndSend(client, pkt)
}

// The client sent us a checksum for one of the patch files. Compare it to what we
// have and add it to the list of files to update if there is any discrepancy.
func (server *DataServer) HandleFileStatus(client *server.Client) {
	var fileStatus archon.FileStatusPacket
	util.StructFromBytes(client.Data(), &fileStatus)

	patch := server.patchIndex[fileStatus.PatchId]
	if fileStatus.Checksum != patch.checksum || fileStatus.FileSize != patch.fileSize {
		client.updateList = append(client.updateList, patch)
	}
}

// The client finished sending all of the file check packets. If they have
// any files that need updating, now's the time to do it.
func (server *DataServer) UpdateClientFiles(client *server.Client) error {
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
				archon.Log.Error(err.Error())
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
	return server.sendUpdateComplete(client)
}

// Send the total number and cumulative size of files that need updating.
func (server *DataServer) sendUpdateFiles(client *server.Client, num, totalSize uint32) error {
	pkt := new(archon.UpdateFilesPacket)
	pkt.Header.Type = archon.PatchUpdateFilesType
	pkt.NumFiles = num
	pkt.TotalSize = totalSize

	archon.Log.Debug("Sending Update Files")
	return archon.EncryptAndSend(client, pkt)
}

// Send the header for a file we're about to update.
func (server *DataServer) sendFileHeader(client *server.Client, patch *PatchEntry) error {
	pkt := new(archon.FileHeaderPacket)
	pkt.Header.Type = archon.PatchFileHeaderType
	pkt.FileSize = patch.fileSize
	copy(pkt.Filename[:], patch.filename)

	archon.Log.Debug("Sending File Header")
	return archon.EncryptAndSend(client, pkt)
}

// Send a chunk of file data.
func (server *DataServer) sendFileChunk(client *server.Client, chunk, chksm, chunkSize uint32, fdata []byte) error {
	if chunkSize > MaxFileChunkSize {
		archon.Log.Errorf("Attempted to send %v byte chunk; max is %v",
			string(chunkSize), string(MaxFileChunkSize))
		panic(errors.New("File chunk size exceeds maximum"))
	}
	pkt := &archon.FileChunkPacket{
		Header:   archon.PCHeader{Type: archon.PatchFileChunkType},
		Chunk:    chunk,
		Checksum: chksm,
		Size:     chunkSize,
		Data:     fdata[:chunkSize],
	}

	archon.Log.Debug("Sending File Chunk")
	return archon.EncryptAndSend(client, pkt)
}

// Finished sending a particular file.
func (server *DataServer) sendFileComplete(client *server.Client) error {
	pkt := &archon.PCHeader{Type: archon.PatchFileCompleteType, Size: 0x04}
	archon.Log.Debug("Sending File Complete")
	return archon.EncryptAndSend(client, pkt)
}

// We've finished updating files.
func (server *DataServer) sendUpdateComplete(client *server.Client) error {
	pkt := &archon.PCHeader{Type: archon.PatchUpdateCompleteType, Size: 0x04}
	archon.Log.Debug("Sending File Update Done")
	return archon.EncryptAndSend(client, pkt)
}
