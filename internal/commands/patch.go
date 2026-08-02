// Commands specific to the patch and data servers.
package commands

// Welcome command with encryption vectors sent to the client upon initial connection.
const PatchWelcomeType = 0x02

type PatchWelcome struct {
	Header       PCHeader
	Copyright    [44]byte
	Padding      [20]byte
	ServerVector [4]byte
	ClientVector [4]byte
}

const PatchHandshakeType = 0x04

// File header for a series of file chunks.
const PatchFileHeaderType = 0x06

type PatchFileHeader struct {
	Header   PCHeader
	Padding  uint32
	FileSize uint32
	Filename [48]byte
}

// Chunk of data from a file.
const PatchFileChunkType = 0x07

type PatchFileChunk struct {
	Header   PCHeader
	Chunk    uint32
	Checksum uint32
	Size     uint32
	Data     []byte
}

const PatchFileCompleteType = 0x08

// Instruct the client to chdir into Dirname (one level below).
const PatchChangeDirType = 0x09

type ChangeDir struct {
	Header  PCHeader
	Dirname [64]byte
}

const PatchDirAboveType = 0x0A

const PatchDataAckType = 0x0B

// Request a check on a file in the client's working directory.
const PatchCheckFileType = 0x0C

type PatchCheckFile struct {
	Header   PCHeader
	PatchID  uint32
	Filename [32]byte
}

const PatchFileListDoneType = 0x0D

// Client response to PatchCheckFileType containing the metadata of their copy of the requested file.
const PatchFileStatusType = 0x0F

type PatchFileStatus struct {
	Header   PCHeader
	PatchID  uint32
	Checksum uint32
	FileSize uint32
}

const PatchClientListDoneType = 0x10

// Size and number of files that need to be updated.
const PatchStartFileUpdateType = 0x11

type PatchStartFileUpdate struct {
	Header    PCHeader
	TotalSize uint32
	NumFiles  uint32
}

const PatchUpdateCompleteType = 0x12

// Command containing the patch server text area welcome message.
const PatchMessageType = 0x13

type PatchWelcomeMessage struct {
	Header  PCHeader
	Message []byte
}

// Redirect command for patch to send character server IP.
const PatchRedirectType = 0x14

type PatchRedirect struct {
	Header  PCHeader
	IPAddr  [4]uint8
	Port    uint16
	Padding uint16
}
