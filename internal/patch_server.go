package internal

import (
	"context"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"strconv"

	"github.com/dcrodman/archon/internal/commands"
	"github.com/dcrodman/archon/internal/encryption"
)

// PatchServer is responsible for exchanging file metadata with game clients
// in order to determine whether or not the client's files match the known patch
// files. If any of the patch file checksums do not equal the checksums of their
// corresponding client files (or do not exist), this server allows the client to
// download the correct file contents and forces a restart.
type PatchServer struct{}

func (s PatchServer) Identifier() string {
	return "PATCH:DATA"
}

func (s *PatchServer) Init(ctx context.Context) error {
	return InitializePatchData()
}

func (s *PatchServer) Handshake(ctx context.Context, c *Client) error {
	c.CryptoSession = encryption.NewPCCryptoSession()
	c.FilesToUpdate = make(map[int]interface{})

	// Send the welcome command to a client with the copyright message and encryption vectors.
	pkt := commands.PatchWelcome{
		Header: commands.PCHeader{Type: commands.PatchWelcomeType, Size: 0x4C},
	}
	copy(pkt.Copyright[:], PatchCopyright)
	copy(pkt.ServerVector[:], c.CryptoSession.ServerVector())
	copy(pkt.ClientVector[:], c.CryptoSession.ClientVector())

	return c.SendRaw(ctx, pkt)
}

func (s *PatchServer) Handle(ctx context.Context, c *Client, data []byte) error {
	var hdr commands.PCHeader

	UnmarshalStruct(data[:commands.PCHeaderSize], &hdr)

	var err error
	switch hdr.Type {
	case commands.PatchWelcomeType:
		err = SendPatchWelcomeAck(c)
	case commands.PatchHandshakeType:
		err = s.handlePatchLogin(c)
	case commands.PatchFileStatusType:
		var fileStatus commands.PatchFileStatus
		UnmarshalStruct(data, &fileStatus)
		s.handleFileStatus(c, &fileStatus)
	case commands.PatchClientListDoneType:
		err = updateClientFiles(c)
	default:
		Logger.Infof("received unknown command %02x from %s", hdr.Type, c.IPAddr)
	}
	return err
}

// Once the client has authenticated, send them the list of files to update.
func (s *PatchServer) handlePatchLogin(c *Client) error {
	if err := SendPatchDataAck(c); err != nil {
		return err
	}
	if err := SendPatchFileList(c, rootNode); err != nil {
		return err
	}
	return SendPatchFileListDone(c)
}

// Acknowledgement sent after the DATA connection handshake.
func SendPatchDataAck(c *Client) error {
	return c.Send(commands.PCHeader{Type: commands.PatchDataAckType, Size: 0x04})
}

// Traverse the patch tree depth-first and send the check file requests.
func SendPatchFileList(c *Client, node *directoryNode) error {
	if err := SendPatchChangeDir(c, node.clientPath); err != nil {
		return err
	}

	for _, patch := range node.patchFiles {
		if err := SendPatchCheckFile(c, patch.index, patch.filename); err != nil {
			return err
		}
	}

	for _, subDirectory := range node.childNodes {
		if err := SendPatchFileList(c, subDirectory); err != nil {
			return err
		}
		// Move them back up each time we leave a directory.
		if err := SendPatchDirAbove(c); err != nil {
			return err
		}
	}

	return nil
}

// Tell the client to change to some directory within its file tree.
func SendPatchChangeDir(c *Client, dir string) error {
	pkt := commands.ChangeDir{
		Header:  commands.PCHeader{Type: commands.PatchChangeDirType},
		Dirname: [64]byte{},
	}
	copy(pkt.Dirname[:], dir)

	return c.Send(pkt)
}

// Tell the client to check a file in its current working directory.
func SendPatchCheckFile(c *Client, index uint32, filename string) error {
	pkt := commands.PatchCheckFile{
		Header:  commands.PCHeader{Type: commands.PatchCheckFileType},
		PatchID: index,
	}
	copy(pkt.Filename[:], filename)

	return c.Send(pkt)
}

// Tell the client to change to one directory above.
func SendPatchDirAbove(c *Client) error {
	return c.Send(commands.PCHeader{Type: commands.PatchDirAboveType, Size: 0x04})
}

// Tell the client that we've finished sending the patch list.
func SendPatchFileListDone(c *Client) error {
	return c.Send(commands.PCHeader{Type: commands.PatchFileListDoneType, Size: 0x04})
}

// The client sent us a checksum for one of the patch files. Compare it to what we
// have and add it to the list of files to update if there is any discrepancy.
func (s *PatchServer) handleFileStatus(c *Client, fileStatus *commands.PatchFileStatus) {
	patchFile := patchIndex[fileStatus.PatchID]

	if fileStatus.Checksum != patchFile.checksum || fileStatus.FileSize != patchFile.fileSize {
		c.FilesToUpdate[int(fileStatus.PatchID)] = patchFile
	}
}

// The client finished sending all of the file check commands. If they have
// any files that need updating, now's the time to do it.
func updateClientFiles(c *Client) error {
	var numFiles, totalSize uint32 = 0, 0

	for _, patch := range c.FilesToUpdate {
		numFiles++
		totalSize += patch.(*fileEntry).fileSize
	}

	if numFiles > 0 {
		if err := SendPatchStartFileUpdate(c, numFiles, totalSize); err != nil {
			return err
		}
		if err := traverseAndUpdate(c, rootNode); err != nil {
			return err
		}
	}

	return SendPatchUpdateComplete(c)
}

// Send the total number and cumulative size of files that need updating.
func SendPatchStartFileUpdate(c *Client, num, totalSize uint32) error {
	return c.Send(commands.PatchStartFileUpdate{
		Header:    commands.PCHeader{Type: commands.PatchStartFileUpdateType},
		NumFiles:  num,
		TotalSize: totalSize,
	})
}

// Recursively traverse our tree of patch files, sending the file data to the client
// when an out of date file is encountered.
func traverseAndUpdate(c *Client, node *directoryNode) error {
	if err := SendPatchChangeDir(c, node.name); err != nil {
		return err
	}

	for _, file := range node.patchFiles {
		if entry, ok := c.FilesToUpdate[int(file.index)]; ok {
			if err := updateClientFile(c, entry.(*fileEntry)); err != nil {
				return err
			}
		}

		for _, dir := range node.childNodes {
			if err := traverseAndUpdate(c, dir); err != nil {
				return err
			}
		}
	}

	// Don't ascend beyond the top level directory or the client will blow up.
	if node.path == "." {
		return nil
	}
	return SendPatchDirAbove(c)
}

func updateClientFile(c *Client, patch *fileEntry) error {
	if err := SendPatchFileHeader(c, patch); err != nil {
		return nil
	}

	file, err := os.Open(patch.path)
	if err != nil {
		Logger.Error(err.Error())
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
		if err := SendPatchFileChunk(c, uint32(i), checksum, uint32(bytes), chunkBuf); err != nil {
			return err
		}
	}

	return SendPatchFileComplete(c)
}

// send the header for a file we're about to update.
func SendPatchFileHeader(c *Client, patch *fileEntry) error {
	pkt := commands.PatchFileHeader{
		Header:   commands.PCHeader{Type: commands.PatchFileHeaderType},
		FileSize: patch.fileSize,
		Filename: [48]byte{},
	}
	copy(pkt.Filename[:], patch.filename)

	return c.Send(pkt)
}

// send a chunk of file data.
func SendPatchFileChunk(c *Client, chunk, chksm, chunkSize uint32, fdata []byte) error {
	if chunkSize > maxFileChunkSize {
		Logger.Errorf("Attempted to send %v byte chunk; max is %v",
			strconv.Itoa(int(chunkSize)), string(rune(maxFileChunkSize)))
		panic(errors.New("file chunk size exceeds maximum"))
	}

	return c.Send(commands.PatchFileChunk{
		Header:   commands.PCHeader{Type: commands.PatchFileChunkType},
		Chunk:    chunk,
		Checksum: chksm,
		Size:     chunkSize,
		Data:     fdata[:chunkSize],
	})
}

// Finished sending a particular file.
func SendPatchFileComplete(c *Client) error {
	return c.Send(commands.PCHeader{Type: commands.PatchFileCompleteType, Size: 0x04})
}

// We've finished updating files.
func SendPatchUpdateComplete(c *Client) error {
	return c.Send(commands.PCHeader{Type: commands.PatchUpdateCompleteType, Size: 0x04})
}
