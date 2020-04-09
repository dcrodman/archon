/*
* Packet constants and structures. All functions return 0 on success,
* negative int on db error, and a positive int for any other errors.
 */
package archon

const (
	PCHeaderSize = 0x04
	BBHeaderSize = 0x08
)

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

// Packet types for packets sent to and from the login and character servers.
const (
	LoginWelcomeType            = 0x03
	LoginType                   = 0x93
	LoginSecurityType           = 0xE6
	LoginClientMessageType      = 0x1A
	LoginOptionsRequestType     = 0xE0
	LoginOptionsType            = 0xE2
	LoginCharPreviewReqType     = 0xE3
	LoginCharAckType            = 0xE4
	LoginCharPreviewType        = 0xE5
	LoginChecksumType           = 0x01E8
	LoginChecksumAckType        = 0x02E8
	LoginGuildcardReqType       = 0x03E8
	LoginGuildcardHeaderType    = 0x01DC
	LoginGuildcardChunkType     = 0x02DC
	LoginGuildcardChunkReqType  = 0x03DC
	LoginParameterHeaderType    = 0x01EB
	LoginParameterChunkType     = 0x02EB
	LoginParameterChunkReqType  = 0x03EB
	LoginParameterHeaderReqType = 0x04EB
	LoginSetFlagType            = 0xEC
	LoginTimestampType          = 0xB1
	LoginShipListType           = 0xA0
	LoginScrollMessageType      = 0xEE
)

// Packet types for packets sent to and from the ship and block servers.
const (
	BlockListType = 0x07
	LobbyListType = 0x83
)

// Packet types common to multiple servers.
const (
	DisconnectType = 0x05
	RedirectType   = 0x19
	MenuSelectType = 0x10
)

// Error code types used for packet E6.
type BBLoginError uint32

const (
	BBLoginErrorNone         = 0x0
	BBLoginErrorUnknown      = 0x1
	BBLoginErrorPassword     = 0x2
	BBLoginErrorPassword2    = 0x3 // Same as password
	BBLoginErrorMaintenance  = 0x4
	BBLoginErrorUserInUse    = 0x5
	BBLoginErrorBanned       = 0x6
	BBLoginErrorBanned2      = 0x7 // Same as banned
	BBLoginErrorUnregistered = 0x8
	BBLoginErrorExpiredSub   = 0x9
	BBLoginErrorLocked       = 0xA
	BBLoginErrorPatch        = 0xB
	BBLoginErrorDisconnect   = 0xC
)

// Blueburst, PC, and Gamecube clients all use a 4 byte header to
// communicate with the patch server instead of the 8 byte one used
// by Blueburst for the other servers.
type PCHeader struct {
	Size uint16
	Type uint16
}

// Packet header for every packet sent between the server and BlueBurst clients.
type BBHeader struct {
	Size  uint16
	Type  uint16
	Flags uint32
}

// Welcome packet with encryption vectors sent to the client upon initial connection.
type PatchWelcomePkt struct {
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
type PatchRedirectPacket struct {
	Header  PCHeader
	IPAddr  [4]uint8
	Port    uint16
	Padding uint16
}

// Instruct the client to chdir into Dirname (one level below).
type ChangeDirPacket struct {
	Header  PCHeader
	Dirname [64]byte
}

// Request a check on a file in the client's working directory.
type CheckFilePacket struct {
	Header   PCHeader
	PatchId  uint32
	Filename [32]byte
}

// Response to CheckFilePacket from the client with the properties of a file.
type FileStatusPacket struct {
	Header   PCHeader
	PatchId  uint32
	Checksum uint32
	FileSize uint32
}

// Size and number of files that need to be updated.
type StartFileUpdatePacket struct {
	Header    PCHeader
	TotalSize uint32
	NumFiles  uint32
}

// File header for a series of file chunks.
type FileHeaderPacket struct {
	Header   PCHeader
	Padding  uint32
	FileSize uint32
	Filename [48]byte
}

// Chunk of data from a file.
type FileChunkPacket struct {
	Header   PCHeader
	Chunk    uint32
	Checksum uint32
	Size     uint32
	Data     []byte
}

// Welcome packet with encryption vectors sent to the client upon initial connection.
type WelcomePkt struct {
	Header       BBHeader
	Copyright    [96]byte
	ServerVector [48]byte
	ClientVector [48]byte
}

// Login Packet (0x93) sent to both the login and character servers.
type LoginPkt struct {
	Header        BBHeader
	Unknown       [8]byte
	ClientVersion uint16
	Unknown2      [3]byte
	SlotNum       int8
	Phase         uint16 // differentiate login packet?
	TeamId        uint32
	Username      [16]byte
	Padding       [32]byte
	Password      [16]byte
	Unknown3      [40]byte
	HardwareInfo  [8]byte
	Security      [40]byte
}

// Represent the client's progression through the login process.
type ClientConfig struct {
	Magic        uint32 // Must be set to 0x48615467
	CharSelected uint8  // Has a character been selected?
	SlotNum      uint8  // Slot number of selected Character
	Flags        uint16
	Ports        [4]uint16
	Unused       [4]uint32
	Unused2      [2]uint32
}

// Security packet (0xE6) sent to the client to indicate the state of client login.
type SecurityPacket struct {
	Header       BBHeader
	ErrorCode    uint32
	PlayerTag    uint32
	Guildcard    uint32
	TeamId       uint32
	Config       *ClientConfig
	Capabilities uint32
}

// The address of the next server; in this case, the character server.
type RedirectPacket struct {
	Header  BBHeader
	IPAddr  [4]uint8
	Port    uint16
	Padding uint16
}

// Based on the key config structure from sylverant and newserv. KeyConfig
// and JoystickConfig are saved in the database.
type KeyTeamConfig struct {
	Unknown            [0x114]uint8
	KeyConfig          [0x16C]uint8
	JoystickConfig     [0x38]uint8
	Guildcard          uint32
	TeamId             uint32
	TeamInfo           [2]uint32
	TeamPrivilegeLevel uint16
	Reserved           uint16
	Teamname           [0x10]uint16
	TeamFlag           [0x0800]uint8
	TeamRewards        [2]uint32
}

// Option packet containing keyboard and joystick config, team options, etc.
type OptionsPacket struct {
	Header          BBHeader
	PlayerKeyConfig KeyTeamConfig
}

type CharSelectionPacket struct {
	Header    BBHeader
	Slot      uint32
	Selecting uint32
}

// Acknowledge a character selection from the client or indicate an error.
type CharAckPacket struct {
	Header BBHeader
	Slot   uint32
	Flag   uint32
}

// Sent in response to 0x01E8 to acknowledge a checksum (really it's just ignored).
type ChecksumAckPacket struct {
	Header BBHeader
	Ack    uint32
}

// Chunk header with info about the guildcard data we're about to send.
type GuildcardHeaderPacket struct {
	Header   BBHeader
	Unknown  uint32
	Length   uint16
	Padding  uint16
	Checksum uint32
}

// Received from the client to request a guildcard data chunk.
type GuildcardChunkReqPacket struct {
	Header         BBHeader
	Unknown        uint32
	ChunkRequested uint32
	Continue       uint32
}

type GuildcardChunkPacket struct {
	Header  BBHeader
	Unknown uint32
	Chunk   uint32
	Data    []uint8
}

// Parameter header containing details about the param files we're about to send.
type ParameterHeaderPacket struct {
	Header  BBHeader
	Entries []byte
}

type ParameterChunkPacket struct {
	Header BBHeader
	Chunk  uint32
	Data   []byte
}

// Used by the client to indicate whether a character should be recreated or updated.
type SetFlagPacket struct {
	Header BBHeader
	Flag   uint32
}

// Sent to the client for the selection menu and received for updating a character.
type CharacterSummaryPacket struct {
	Header    BBHeader
	Slot      uint32
	Character CharacterSummary
}

// Message in a large text box, usually sent right before a disconnect.
type LoginClientMessagePacket struct {
	Header   BBHeader
	Language uint32
	Message  []byte
}

// Indicate the server's current time.
type TimestampPacket struct {
	Header    BBHeader
	Timestamp [28]byte
}

// The list of menu items to display to the client.
type ShipListPacket struct {
	Header     BBHeader
	Padding    uint16
	Unknown    uint16 // set to 0xFFFFFFF4
	Unknown2   uint32 // set to 0x02
	Unknown3   uint16 // set to 0x04
	ServerName [36]byte
	//ShipEntries []character.ShipMenuEntry
}

// Scroll message the client should display on the ship select screen.
type ScrollMessagePacket struct {
	Header  BBHeader
	Padding [2]uint32
	Message []byte
}

// Client's selection from the ship or block selection menu.
type MenuSelectionPacket struct {
	Header  BBHeader
	Unknown uint16
	MenuId  uint16
	ItemId  uint32
}

// List containing the available blocks on a ship.
type BlockListPacket struct {
	Header   BBHeader
	Padding  [10]byte
	ShipName [32]byte
	Unknown  uint32
	Blocks   []Block
}

// Info about the available block servers.
type Block struct {
	Unknown   uint16
	BlockId   uint32
	Padding   uint16
	BlockName [36]byte
}

// Available lobbies on a block.
type LobbyListPacket struct {
	Header  BBHeader
	Lobbies []struct {
		MenuId  uint32
		LobbyId uint32
		Padding uint32
	}
}
