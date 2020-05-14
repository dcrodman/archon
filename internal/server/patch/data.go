package patch

import (
	"errors"
	"github.com/dcrodman/archon"
	crypto "github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/dcrodman/archon/internal/server"
	"github.com/dcrodman/archon/internal/server/internal"
	"hash/crc32"
	"io"
	"os"
)

type DataServer struct {
	name string
	port string
}

func NewDataServer(name, port string) server.Server {
	return &DataServer{
		name: name,
		port: port,
	}
}

func (s DataServer) Name() string        { return s.name }
func (s DataServer) Port() string        { return s.port }
func (s *DataServer) HeaderSize() uint16 { return packets.PCHeaderSize }
func (s *DataServer) Init() error        { return initializePatchData() }

func (s DataServer) AcceptClient(cs *server.ConnectionState) (server.Client, error) {
	c := &client{
		cs:            cs,
		clientCrypt:   crypto.NewPCCrypt(),
		serverCrypt:   crypto.NewPCCrypt(),
		filesToUpdate: make(map[int]*fileEntry),
	}

	var err error
	if sendPCWelcome(c) != nil {
		err = errors.New("Error sending welcome packet to: " + c.ConnectionState().IPAddr())
		c = nil
	}
	return c, err
}

func (s DataServer) Handle(client server.Client) error {
	c := client.(*client)
	var hdr packets.PCHeader

	internal.StructFromBytes(c.ConnectionState().Data()[:packets.PCHeaderSize], &hdr)

	var err error
	switch hdr.Type {
	case packets.PatchWelcomeType:
		err = s.sendWelcomeAck(c)
	case packets.PatchHandshakeType:
		err = s.handlePatchLogin(c)
	case packets.PatchFileStatusType:
		s.handleFileStatus(c)
	case packets.PatchClientListDoneType:
		err = s.updateClientFiles(c)
	default:
		archon.Log.Infof("Received unknown packet %02x from %s", hdr.Type, c.ConnectionState().IPAddr())
	}
	return err
}

// Simple acknowledgement to the welcome response.
func (s *DataServer) sendWelcomeAck(client *client) error {
	return client.send(packets.PCHeader{
		Size: 0x04,
		Type: packets.PatchHandshakeType,
	})
}

// Once the client has authenticated, send them the list of files to update.
func (s *DataServer) handlePatchLogin(c *client) error {
	if err := s.sendDataAck(c); err != nil {
		return err
	}
	if err := s.sendFileList(c, rootNode); err != nil {
		return err
	}
	return s.sendFileListDone(c)
}

// Acknowledgement sent after the DATA connection handshake.
func (s *DataServer) sendDataAck(client *client) error {
	return client.send(packets.PCHeader{Type: packets.PatchDataAckType, Size: 0x04})
}

// Traverse the patch tree depth-first and send the check file requests.
func (s *DataServer) sendFileList(client *client, node *directoryNode) error {
	if err := s.sendChangeDir(client, node.clientPath); err != nil {
		return err
	}

	for _, patch := range node.patchFiles {
		if err := s.sendCheckFile(client, patch.index, patch.filename); err != nil {
			return err
		}
	}

	for _, subDirectory := range node.childNodes {
		if err := s.sendFileList(client, subDirectory); err != nil {
			return err
		}
		// Move them back up each time we leave a directory.
		if err := s.sendDirAbove(client); err != nil {
			return err
		}
	}

	return nil
}

// Tell the client to change to some directory within its file tree.
func (s *DataServer) sendChangeDir(client *client, dir string) error {
	pkt := packets.ChangeDir{
		Header:  packets.PCHeader{Type: packets.PatchChangeDirType},
		Dirname: [64]byte{},
	}
	copy(pkt.Dirname[:], dir)

	return client.send(pkt)
}

// Tell the client to check a file in its current working directory.
func (s *DataServer) sendCheckFile(client *client, index uint32, filename string) error {
	pkt := packets.CheckFile{
		Header:  packets.PCHeader{Type: packets.PatchCheckFileType},
		PatchId: index,
	}
	copy(pkt.Filename[:], filename)

	return client.send(pkt)
}

// Tell the client to change to one directory above.
func (s *DataServer) sendDirAbove(client *client) error {
	return client.send(packets.PCHeader{Type: packets.PatchDirAboveType, Size: 0x04})
}

// Tell the client that we've finished sending the patch list.
func (s *DataServer) sendFileListDone(client *client) error {
	return client.send(packets.PCHeader{Type: packets.PatchFileListDoneType, Size: 0x04})
}

// The client sent us a checksum for one of the patch files. Compare it to what we
// have and add it to the list of files to update if there is any discrepancy.
func (s *DataServer) handleFileStatus(client *client) {
	var fileStatus packets.FileStatus
	internal.StructFromBytes(client.ConnectionState().Data(), &fileStatus)

	patchFile := patchIndex[fileStatus.PatchId]

	if fileStatus.Checksum != patchFile.checksum || fileStatus.FileSize != patchFile.fileSize {
		client.filesToUpdate[int(fileStatus.PatchId)] = patchFile
	}
}

// The client finished sending all of the file check packets. If they have
// any files that need updating, now's the time to do it.
func (s *DataServer) updateClientFiles(client *client) error {
	var numFiles, totalSize uint32 = 0, 0

	for _, patch := range client.filesToUpdate {
		numFiles++
		totalSize += patch.fileSize
	}

	if numFiles > 0 {
		if err := s.sendStartFileUpdate(client, numFiles, totalSize); err != nil {
			return err
		}
		if err := s.traverseAndUpdate(client, rootNode); err != nil {
			return err
		}
	}

	return s.sendUpdateComplete(client)
}

// Send the total number and cumulative size of files that need updating.
func (s *DataServer) sendStartFileUpdate(client *client, num, totalSize uint32) error {
	return client.send(packets.StartFileUpdate{
		Header:    packets.PCHeader{Type: packets.PatchUpdateFilesType},
		NumFiles:  num,
		TotalSize: totalSize,
	})
}

// Recursively traverse our tree of patch files, sending the file data to the client
// when an out of date file is encountered.
func (s *DataServer) traverseAndUpdate(client *client, node *directoryNode) error {
	if err := s.sendChangeDir(client, node.name); err != nil {
		return err
	}

	for _, file := range node.patchFiles {
		if entry, ok := client.filesToUpdate[int(file.index)]; ok {
			if err := s.updateClientFile(client, entry); err != nil {
				return err
			}
		}

		for _, dir := range node.childNodes {
			if err := s.traverseAndUpdate(client, dir); err != nil {
				return err
			}
		}
	}

	// Don't ascend beyond the top level directory or the client will blow up.
	if node.path == "." {
		return nil
	}
	return s.sendDirAbove(client)
}

func (s *DataServer) updateClientFile(client *client, patch *fileEntry) error {
	if err := s.sendFileHeader(client, patch); err != nil {
		return nil
	}

	file, err := os.Open(patch.path)
	if err != nil {
		archon.Log.Error(err.Error())
		return err
	}

	// Divide the file into nChunks and send each one.
	nChunks := int((patch.fileSize / maxFileChunkSize) + 1)

	for i := 0; i < nChunks; i++ {
		chunkBuf := make([]byte, maxFileChunkSize)
		// Note: may need to cache these nChunks so that we don't risk having too many
		// file descriptors open given the dispatcher uses unbounded channels.
		bytes, err := file.ReadAt(chunkBuf, int64(maxFileChunkSize*i))
		if err != nil && err != io.EOF {
			return err
		}

		checksum := crc32.ChecksumIEEE(chunkBuf)
		if err := s.sendFileChunk(client, uint32(i), checksum, uint32(bytes), chunkBuf); err != nil {
			return err
		}
	}

	return s.sendFileComplete(client)
}

// send the header for a file we're about to update.
func (s *DataServer) sendFileHeader(client *client, patch *fileEntry) error {
	pkt := packets.FileHeader{
		Header:   packets.PCHeader{Type: packets.PatchFileHeaderType},
		FileSize: patch.fileSize,
		Filename: [48]byte{},
	}
	copy(pkt.Filename[:], patch.filename)

	return client.send(pkt)
}

// send a chunk of file data.
func (s *DataServer) sendFileChunk(client *client, chunk, chksm, chunkSize uint32, fdata []byte) error {
	if chunkSize > maxFileChunkSize {
		archon.Log.Errorf("Attempted to send %v byte chunk; max is %v",
			string(chunkSize), string(maxFileChunkSize))
		panic(errors.New("file chunk size exceeds maximum"))
	}

	return client.send(packets.FileChunk{
		Header:   packets.PCHeader{Type: packets.PatchFileChunkType},
		Chunk:    chunk,
		Checksum: chksm,
		Size:     chunkSize,
		Data:     fdata[:chunkSize],
	})
}

// Finished sending a particular file.
func (s *DataServer) sendFileComplete(client *client) error {
	return client.send(packets.PCHeader{Type: packets.PatchFileCompleteType, Size: 0x04})
}

// We've finished updating files.
func (s *DataServer) sendUpdateComplete(client *client) error {
	return client.send(packets.PCHeader{Type: packets.PatchUpdateCompleteType, Size: 0x04})
}
