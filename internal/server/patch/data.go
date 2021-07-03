package patch

import (
	"context"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"strconv"

	"github.com/dcrodman/archon"
	crypto "github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/dcrodman/archon/internal/server"
	"github.com/dcrodman/archon/internal/server/internal"
)

// DataServer is responsible for exchanging file metadata with game clients
// in order to determine whether or not the client's files match the known patch
// files. If any of the patch file checksums do not equal the checksums of their
// corresponding client files (or do not exist), this server allows the client to
// download the correct file contents and forces a restart.
type DataServer struct {
	name string
}

func NewDataServer(name string) *DataServer {
	return &DataServer{name: name}
}

func (s DataServer) Name() string { return s.name }

func (s *DataServer) Init(ctx context.Context) error {
	return initializePatchData()
}

func (s *DataServer) CreateExtension() server.ClientExtension {
	return &patchClientExtension{
		clientCrypt:   crypto.NewPCCrypt(),
		serverCrypt:   crypto.NewPCCrypt(),
		filesToUpdate: make(map[int]*fileEntry),
	}
}

func (s *DataServer) StartSession(c *server.Client) error {
	ext := c.Extension.(*patchClientExtension)

	// Send the welcome packet to a client with the copyright message and encryption vectors.
	pkt := packets.PatchWelcome{
		Header: packets.PCHeader{Type: packets.PatchWelcomeType, Size: 0x4C},
	}
	copy(pkt.Copyright[:], copyright)
	copy(pkt.ServerVector[:], ext.serverCrypt.Vector)
	copy(pkt.ClientVector[:], ext.clientCrypt.Vector)

	return c.SendRaw(pkt)
}

func (s *DataServer) Handle(ctx context.Context, c *server.Client, data []byte) error {
	var hdr packets.PCHeader

	internal.StructFromBytes(data[:packets.PCHeaderSize], &hdr)

	var err error
	switch hdr.Type {
	case packets.PatchWelcomeType:
		err = s.sendWelcomeAck(c)
	case packets.PatchHandshakeType:
		err = s.handlePatchLogin(c)
	case packets.PatchFileStatusType:
		var fileStatus packets.FileStatus
		internal.StructFromBytes(data, &fileStatus)
		s.handleFileStatus(c, &fileStatus)
	case packets.PatchClientListDoneType:
		err = s.updateClientFiles(c)
	default:
		archon.Log.Infof("Received unknown packet %02x from %s", hdr.Type, c.IPAddr())
	}
	return err
}

// Simple acknowledgement to the welcome response.
func (s *DataServer) sendWelcomeAck(c *server.Client) error {
	return c.Send(packets.PCHeader{
		Size: 0x04,
		Type: packets.PatchHandshakeType,
	})
}

// Once the client has authenticated, send them the list of files to update.
func (s *DataServer) handlePatchLogin(c *server.Client) error {
	if err := s.sendDataAck(c); err != nil {
		return err
	}
	if err := s.sendFileList(c, rootNode); err != nil {
		return err
	}
	return s.sendFileListDone(c)
}

// Acknowledgement sent after the DATA connection handshake.
func (s *DataServer) sendDataAck(c *server.Client) error {
	return c.Send(packets.PCHeader{Type: packets.PatchDataAckType, Size: 0x04})
}

// Traverse the patch tree depth-first and send the check file requests.
func (s *DataServer) sendFileList(c *server.Client, node *directoryNode) error {
	if err := s.sendChangeDir(c, node.clientPath); err != nil {
		return err
	}

	for _, patch := range node.patchFiles {
		if err := s.sendCheckFile(c, patch.index, patch.filename); err != nil {
			return err
		}
	}

	for _, subDirectory := range node.childNodes {
		if err := s.sendFileList(c, subDirectory); err != nil {
			return err
		}
		// Move them back up each time we leave a directory.
		if err := s.sendDirAbove(c); err != nil {
			return err
		}
	}

	return nil
}

// Tell the client to change to some directory within its file tree.
func (s *DataServer) sendChangeDir(c *server.Client, dir string) error {
	pkt := packets.ChangeDir{
		Header:  packets.PCHeader{Type: packets.PatchChangeDirType},
		Dirname: [64]byte{},
	}
	copy(pkt.Dirname[:], dir)

	return c.Send(pkt)
}

// Tell the client to check a file in its current working directory.
func (s *DataServer) sendCheckFile(c *server.Client, index uint32, filename string) error {
	pkt := packets.CheckFile{
		Header:  packets.PCHeader{Type: packets.PatchCheckFileType},
		PatchID: index,
	}
	copy(pkt.Filename[:], filename)

	return c.Send(pkt)
}

// Tell the client to change to one directory above.
func (s *DataServer) sendDirAbove(c *server.Client) error {
	return c.Send(packets.PCHeader{Type: packets.PatchDirAboveType, Size: 0x04})
}

// Tell the client that we've finished sending the patch list.
func (s *DataServer) sendFileListDone(c *server.Client) error {
	return c.Send(packets.PCHeader{Type: packets.PatchFileListDoneType, Size: 0x04})
}

// The client sent us a checksum for one of the patch files. Compare it to what we
// have and add it to the list of files to update if there is any discrepancy.
func (s *DataServer) handleFileStatus(c *server.Client, fileStatus *packets.FileStatus) {
	patchFile := patchIndex[fileStatus.PatchID]

	if fileStatus.Checksum != patchFile.checksum || fileStatus.FileSize != patchFile.fileSize {
		ext := c.Extension.(*patchClientExtension)
		ext.filesToUpdate[int(fileStatus.PatchID)] = patchFile
	}
}

// The client finished sending all of the file check packets. If they have
// any files that need updating, now's the time to do it.
func (s *DataServer) updateClientFiles(c *server.Client) error {
	var numFiles, totalSize uint32 = 0, 0

	for _, patch := range c.Extension.(*patchClientExtension).filesToUpdate {
		numFiles++
		totalSize += patch.fileSize
	}

	if numFiles > 0 {
		if err := s.sendStartFileUpdate(c, numFiles, totalSize); err != nil {
			return err
		}
		if err := s.traverseAndUpdate(c, rootNode); err != nil {
			return err
		}
	}

	return s.sendUpdateComplete(c)
}

// Send the total number and cumulative size of files that need updating.
func (s *DataServer) sendStartFileUpdate(c *server.Client, num, totalSize uint32) error {
	return c.Send(packets.StartFileUpdate{
		Header:    packets.PCHeader{Type: packets.PatchUpdateFilesType},
		NumFiles:  num,
		TotalSize: totalSize,
	})
}

// Recursively traverse our tree of patch files, sending the file data to the client
// when an out of date file is encountered.
func (s *DataServer) traverseAndUpdate(c *server.Client, node *directoryNode) error {
	if err := s.sendChangeDir(c, node.name); err != nil {
		return err
	}

	ext := c.Extension.(*patchClientExtension)

	for _, file := range node.patchFiles {
		if entry, ok := ext.filesToUpdate[int(file.index)]; ok {
			if err := s.updateClientFile(c, entry); err != nil {
				return err
			}
		}

		for _, dir := range node.childNodes {
			if err := s.traverseAndUpdate(c, dir); err != nil {
				return err
			}
		}
	}

	// Don't ascend beyond the top level directory or the client will blow up.
	if node.path == "." {
		return nil
	}
	return s.sendDirAbove(c)
}

func (s *DataServer) updateClientFile(c *server.Client, patch *fileEntry) error {
	if err := s.sendFileHeader(c, patch); err != nil {
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
		if err := s.sendFileChunk(c, uint32(i), checksum, uint32(bytes), chunkBuf); err != nil {
			return err
		}
	}

	return s.sendFileComplete(c)
}

// send the header for a file we're about to update.
func (s *DataServer) sendFileHeader(c *server.Client, patch *fileEntry) error {
	pkt := packets.FileHeader{
		Header:   packets.PCHeader{Type: packets.PatchFileHeaderType},
		FileSize: patch.fileSize,
		Filename: [48]byte{},
	}
	copy(pkt.Filename[:], patch.filename)

	return c.Send(pkt)
}

// send a chunk of file data.
func (s *DataServer) sendFileChunk(c *server.Client, chunk, chksm, chunkSize uint32, fdata []byte) error {
	if chunkSize > maxFileChunkSize {
		archon.Log.Errorf("Attempted to send %v byte chunk; max is %v",
			strconv.Itoa(int(chunkSize)), string(rune(maxFileChunkSize)))
		panic(errors.New("file chunk size exceeds maximum"))
	}

	return c.Send(packets.FileChunk{
		Header:   packets.PCHeader{Type: packets.PatchFileChunkType},
		Chunk:    chunk,
		Checksum: chksm,
		Size:     chunkSize,
		Data:     fdata[:chunkSize],
	})
}

// Finished sending a particular file.
func (s *DataServer) sendFileComplete(c *server.Client) error {
	return c.Send(packets.PCHeader{Type: packets.PatchFileCompleteType, Size: 0x04})
}

// We've finished updating files.
func (s *DataServer) sendUpdateComplete(c *server.Client) error {
	return c.Send(packets.PCHeader{Type: packets.PatchUpdateCompleteType, Size: 0x04})
}
