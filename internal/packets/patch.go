// Packets specific to the patch and data servers.
package packets

// Packet types handled by the patch and data servers.
const (
	PatchWelcomeType        = 0x02
	PatchHandshakeType      = 0x04
	PatchMessageType        = 0x13
	PatchRedirectType       = 0x14
	PatchDataAckType        = 0x0B
	PatchDirAboveType       = 0x0A
	PatchChangeDirType      = 0x09
	PatchCheckFileType      = 0x0C
	PatchFileListDoneType   = 0x0D
	PatchFileStatusType     = 0x0F
	PatchClientListDoneType = 0x10
	PatchUpdateFilesType    = 0x11
	PatchFileHeaderType     = 0x06
	PatchFileChunkType      = 0x07
	PatchFileCompleteType   = 0x08
	PatchUpdateCompleteType = 0x12
)

// Welcome packet with encryption vectors sent to the client upon initial connection.
type PatchWelcome struct {
	Header       PCHeader
	Copyright    [44]byte
	Padding      [20]byte
	ServerVector [4]byte
	ClientVector [4]byte
}

// Packet containing the patch server welcome message.
type PatchWelcomeMessage struct {
	Header  PCHeader
	Message []byte
}

// Redirect packet for patch to send character server IP.
type PatchRedirect struct {
	Header  PCHeader
	IPAddr  [4]uint8
	Port    uint16
	Padding uint16
}

// Instruct the client to chdir into Dirname (one level below).
type ChangeDir struct {
	Header  PCHeader
	Dirname [64]byte
}

// Request a check on a file in the client's working directory.
type CheckFile struct {
	Header   PCHeader
	PatchId  uint32
	Filename [32]byte
}

// Response to CheckFile from the client with the properties of a file.
type FileStatus struct {
	Header   PCHeader
	PatchId  uint32
	Checksum uint32
	FileSize uint32
}

// Size and number of files that need to be updated.
type StartFileUpdate struct {
	Header    PCHeader
	TotalSize uint32
	NumFiles  uint32
}

// File header for a series of file chunks.
type FileHeader struct {
	Header   PCHeader
	Padding  uint32
	FileSize uint32
	Filename [48]byte
}

// Chunk of data from a file.
type FileChunk struct {
	Header   PCHeader
	Chunk    uint32
	Checksum uint32
	Size     uint32
	Data     []byte
}
