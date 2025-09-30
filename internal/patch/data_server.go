package patch

import (
	"context"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"strconv"

	"github.com/dcrodman/archon/internal/commands"
	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/client"
	"github.com/dcrodman/archon/internal/core/debug"
	"go.uber.org/zap"
)

// PatchDataServer is responsible for exchanging file metadata with game clients
// in order to determine whether or not the client's files match the known patch
// files. If any of the patch file checksums do not equal the checksums of their
// corresponding client files (or do not exist), this server allows the client to
// download the correct file contents and forces a restart.
type PatchDataServer struct {
	Config *core.Config
	Logger *zap.SugaredLogger
}

func (s PatchDataServer) Identifier() string {
	return "PATCH:DATA"
}

func (s *PatchDataServer) Init(ctx context.Context) error {
	return initializePatchData(s.Logger, s.Config)
}

func (s *PatchDataServer) SetUpClient(c *client.Client) {
	c.CryptoSession = client.NewPCCryptoSession()
	c.FilesToUpdate = make(map[int]interface{})
	c.DebugTags[debug.SERVER_TYPE] = debug.DATA_SERVER
}

func (s *PatchDataServer) Handshake(c *client.Client) error {
	// Send the welcome command to a client with the copyright message and encryption vectors.
	pkt := commands.PatchWelcome{
		Header: commands.PCHeader{Type: commands.PatchWelcomeType, Size: 0x4C},
	}
	copy(pkt.Copyright[:], copyright)
	copy(pkt.ServerVector[:], c.CryptoSession.ServerVector())
	copy(pkt.ClientVector[:], c.CryptoSession.ClientVector())

	return c.SendRaw(pkt)
}

func (s *PatchDataServer) Handle(ctx context.Context, c *client.Client, data []byte) error {
	var hdr commands.PCHeader

	bytes.StructFromBytes(data[:commands.PCHeaderSize], &hdr)

	var err error
	switch hdr.Type {
	case commands.PatchWelcomeType:
		err = s.sendWelcomeAck(c)
	case commands.PatchHandshakeType:
		err = s.handlePatchLogin(c)
	case commands.PatchFileStatusType:
		var fileStatus commands.FileStatus
		bytes.StructFromBytes(data, &fileStatus)
		s.handleFileStatus(c, &fileStatus)
	case commands.PatchClientListDoneType:
		err = s.updateClientFiles(c)
	default:
		s.Logger.Infof("received unknown command %02x from %s", hdr.Type, c.IPAddr())
	}
	return err
}

// Simple acknowledgement to the welcome response.
func (s *PatchDataServer) sendWelcomeAck(c *client.Client) error {
	return c.Send(commands.PCHeader{
		Size: 0x04,
		Type: commands.PatchHandshakeType,
	})
}

// Once the client has authenticated, send them the list of files to update.
func (s *PatchDataServer) handlePatchLogin(c *client.Client) error {
	if err := s.sendDataAck(c); err != nil {
		return err
	}
	if err := s.sendFileList(c, rootNode); err != nil {
		return err
	}
	return s.sendFileListDone(c)
}

// Acknowledgement sent after the DATA connection handshake.
func (s *PatchDataServer) sendDataAck(c *client.Client) error {
	return c.Send(commands.PCHeader{Type: commands.PatchDataAckType, Size: 0x04})
}

// Traverse the patch tree depth-first and send the check file requests.
func (s *PatchDataServer) sendFileList(c *client.Client, node *directoryNode) error {
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
func (s *PatchDataServer) sendChangeDir(c *client.Client, dir string) error {
	pkt := commands.ChangeDir{
		Header:  commands.PCHeader{Type: commands.PatchChangeDirType},
		Dirname: [64]byte{},
	}
	copy(pkt.Dirname[:], dir)

	return c.Send(pkt)
}

// Tell the client to check a file in its current working directory.
func (s *PatchDataServer) sendCheckFile(c *client.Client, index uint32, filename string) error {
	pkt := commands.CheckFile{
		Header:  commands.PCHeader{Type: commands.PatchCheckFileType},
		PatchID: index,
	}
	copy(pkt.Filename[:], filename)

	return c.Send(pkt)
}

// Tell the client to change to one directory above.
func (s *PatchDataServer) sendDirAbove(c *client.Client) error {
	return c.Send(commands.PCHeader{Type: commands.PatchDirAboveType, Size: 0x04})
}

// Tell the client that we've finished sending the patch list.
func (s *PatchDataServer) sendFileListDone(c *client.Client) error {
	return c.Send(commands.PCHeader{Type: commands.PatchFileListDoneType, Size: 0x04})
}

// The client sent us a checksum for one of the patch files. Compare it to what we
// have and add it to the list of files to update if there is any discrepancy.
func (s *PatchDataServer) handleFileStatus(c *client.Client, fileStatus *commands.FileStatus) {
	patchFile := patchIndex[fileStatus.PatchID]

	if fileStatus.Checksum != patchFile.checksum || fileStatus.FileSize != patchFile.fileSize {
		c.FilesToUpdate[int(fileStatus.PatchID)] = patchFile
	}
}

// The client finished sending all of the file check commands. If they have
// any files that need updating, now's the time to do it.
func (s *PatchDataServer) updateClientFiles(c *client.Client) error {
	var numFiles, totalSize uint32 = 0, 0

	for _, patch := range c.FilesToUpdate {
		numFiles++
		totalSize += patch.(*fileEntry).fileSize
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
func (s *PatchDataServer) sendStartFileUpdate(c *client.Client, num, totalSize uint32) error {
	return c.Send(commands.StartFileUpdate{
		Header:    commands.PCHeader{Type: commands.PatchUpdateFilesType},
		NumFiles:  num,
		TotalSize: totalSize,
	})
}

// Recursively traverse our tree of patch files, sending the file data to the client
// when an out of date file is encountered.
func (s *PatchDataServer) traverseAndUpdate(c *client.Client, node *directoryNode) error {
	if err := s.sendChangeDir(c, node.name); err != nil {
		return err
	}

	for _, file := range node.patchFiles {
		if entry, ok := c.FilesToUpdate[int(file.index)]; ok {
			if err := s.updateClientFile(c, entry.(*fileEntry)); err != nil {
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

func (s *PatchDataServer) updateClientFile(c *client.Client, patch *fileEntry) error {
	if err := s.sendFileHeader(c, patch); err != nil {
		return nil
	}

	file, err := os.Open(patch.path)
	if err != nil {
		s.Logger.Error(err.Error())
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
func (s *PatchDataServer) sendFileHeader(c *client.Client, patch *fileEntry) error {
	pkt := commands.FileHeader{
		Header:   commands.PCHeader{Type: commands.PatchFileHeaderType},
		FileSize: patch.fileSize,
		Filename: [48]byte{},
	}
	copy(pkt.Filename[:], patch.filename)

	return c.Send(pkt)
}

// send a chunk of file data.
func (s *PatchDataServer) sendFileChunk(c *client.Client, chunk, chksm, chunkSize uint32, fdata []byte) error {
	if chunkSize > maxFileChunkSize {
		s.Logger.Errorf("Attempted to send %v byte chunk; max is %v",
			strconv.Itoa(int(chunkSize)), string(rune(maxFileChunkSize)))
		panic(errors.New("file chunk size exceeds maximum"))
	}

	return c.Send(commands.FileChunk{
		Header:   commands.PCHeader{Type: commands.PatchFileChunkType},
		Chunk:    chunk,
		Checksum: chksm,
		Size:     chunkSize,
		Data:     fdata[:chunkSize],
	})
}

// Finished sending a particular file.
func (s *PatchDataServer) sendFileComplete(c *client.Client) error {
	return c.Send(commands.PCHeader{Type: commands.PatchFileCompleteType, Size: 0x04})
}

// We've finished updating files.
func (s *PatchDataServer) sendUpdateComplete(c *client.Client) error {
	return c.Send(commands.PCHeader{Type: commands.PatchUpdateCompleteType, Size: 0x04})
}
