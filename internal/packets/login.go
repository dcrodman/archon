/*
* Packet constants and structures. All functions return 0 on success,
* negative int on db error, and a positive int for any other errors.
 */
package packets

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

// Welcome packet with encryption vectors sent to the client upon initial connection.
type Welcome struct {
	Header       BBHeader
	Copyright    [96]byte
	ServerVector [48]byte
	ClientVector [48]byte
}

// LoginPhase is an identifier set by the client to distinguish the "phases" it passes
// though with the Character server. The client disconnects and then reconnects between
// each phase.
type LoginPhase uint8

const (
	// Initialize represents the first connection with the Character server. The
	// client expects to authenticate, download the parameter files, and get the
	// previews of the account's characters.
	Initialize LoginPhase = iota
	// CharacterSelect is the second connection with the Character server and
	// all the client seems to do is to set a flag indicating that the user is
	// choosing a character.
	CharacterSelect
	// CharacterCreate is an optional connection with the Character server and indicates
	// that the user has either created a new character or recreated an existing one.
	CharacterCreate
	// CharacterUpdate is another optional connection with the Character server and
	// only appears when the user selects the Dressing Room during character selection.
	CharacterUpdate
	// ShipSelection is the final connection with the Character server. The client expects
	// to receive the ship list and the IP address of the selected Ship server.
	ShipSelection
)

// Login Packet (0x93) sent to both the login and character servers.
type Login struct {
	Header        BBHeader
	Unknown       [8]byte
	ClientVersion uint16
	Unknown2      uint32
	Phase         LoginPhase
	Unknown4      uint8 // It's not clear yet if this field is part of/related to the Phase field but it can take either 0 or e value on different clients
	TeamID        uint32
	Username      [16]byte
	Padding       [32]byte
	Password      [16]byte
	Unknown3      [40]byte
	HardwareInfo  [8]byte
	Security      [48]byte
	Padding2      uint32
}

type ClientConfig struct {
	// The rest of this holds various portions of client state to represent
	// the client's progression through the login process.
	Magic        uint32 // Must be set to 0x48615467
	CharSelected uint8  // Has a character been selected?
	SlotNum      uint8  // Slot number of selected Character
	Flags        uint16
	Ports        [4]uint16
	Unused       [4]uint32
	Unused2      [2]uint32
}

// Security packet (0xE6) sent to the client to indicate the state of client login.
type Security struct {
	Header       BBHeader
	ErrorCode    uint32
	PlayerTag    uint32
	Guildcard    uint32
	TeamID       uint32
	Config       ClientConfig
	Capabilities uint32
}

// The address of the next server; in this case, the character server.
type Redirect struct {
	Header  BBHeader
	IPAddr  [4]uint8
	Port    uint16
	Padding uint16
}

// Options packet containing keyboard and joystick config, team options, etc.
type Options struct {
	Header BBHeader
	// Based on the key config structure from sylverant and newserv. KeyConfig
	// and JoystickConfig are saved in the database.
	//
	// Note: This packet is shortened by dropping 4 bytes from TeamFlag in order
	// to align it with tethealla. Sylverant and Newserv do not do this and this
	// may not actually be right.
	PlayerKeyConfig struct {
		Unknown            [0x114]uint8
		KeyConfig          [0x16C]uint8
		JoystickConfig     [0x38]uint8
		Guildcard          uint32
		TeamID             uint32
		TeamInfo           [2]uint32
		TeamPrivilegeLevel uint16
		Reserved           uint16
		Teamname           [0x10]uint16
		TeamFlag           [0x7FC]uint8
		TeamRewards        [2]uint32
	}
}

type CharacterSelection struct {
	Header    BBHeader
	Slot      uint32
	Selecting uint32
}

// Acknowledge a character selection from the client or indicate an error.
type CharacterAck struct {
	Header BBHeader
	Slot   uint32
	Flag   uint32
}

// Sent in response to 0x01E8 to acknowledge a checksum (really it's just ignored).
type ChecksumAck struct {
	Header BBHeader
	Ack    uint32
}

// Chunk header with info about the guildcard data we're about to send.
type GuildcardHeader struct {
	Header   BBHeader
	Unknown  uint32
	Length   uint16
	Padding  uint16
	Checksum uint32
}

// Received from the client to request a guildcard data chunk.
type GuildcardChunkRequest struct {
	Header         BBHeader
	Unknown        uint32
	ChunkRequested uint32
	Continue       uint32
}

type GuildcardChunk struct {
	Header  BBHeader
	Unknown uint32
	Chunk   uint32
	Data    []uint8
}

// Parameter header containing details about the param files we're about to send.
type ParameterHeader struct {
	Header  BBHeader
	Entries []byte
}

type ParameterChunk struct {
	Header BBHeader
	Chunk  uint32
	Data   []byte
}

// Used by the client to indicate whether a character should be recreated or updated.
type SetFlag struct {
	Header BBHeader
	Flag   uint32
}

// CharacterSummary is the common intermediate representation of a Character as it gets
// passed around various servers and/or stored.
type CharacterPreview struct {
	Experience     uint32
	Level          uint32
	GuildcardStr   [16]byte
	Unknown        [2]uint32
	NameColor      uint32
	Model          byte
	Padding        [15]byte
	NameColorChksm uint32
	SectionID      byte
	Class          byte
	V2Flags        byte
	Version        byte
	V1Flags        uint32
	Costume        uint16
	Skin           uint16
	Face           uint16
	Head           uint16
	Hair           uint16
	HairRed        uint16
	HairGreen      uint16
	HairBlue       uint16
	PropX          float32
	PropY          float32
	// In reality this is [16]uint16 but []uint8 is more convenient to work with.
	Name     [32]uint8
	Playtime uint32
}

// Sent to the client for the selection menu and received for updating a character.
type CharacterSummary struct {
	Header    BBHeader
	Slot      uint32
	Character CharacterPreview
}

// Message in a large text box, usually sent right before a disconnect.
type LoginClientMessage struct {
	Header   BBHeader
	Language uint32
	Message  []byte
}

// Indicate the server's current time.
type Timestamp struct {
	Header    BBHeader
	Timestamp [28]byte
}

type ShipListEntry struct {
	MenuID   uint16
	ShipID   uint32
	Padding  uint16
	ShipName [36]byte
}

// The list of menu items to display to the client.
type ShipList struct {
	Header      BBHeader
	Padding     uint16
	Unknown     uint16 // Always 0x20
	Unknown2    uint32 // Always 0xFFFFFFF4
	Unknown3    uint16 // Always 0x04
	ServerName  [32]byte
	Padding2    uint32
	ShipEntries []ShipListEntry
}

// Scroll message the client should display on the ship select screen.
type ScrollMessagePacket struct {
	Header  BBHeader
	Padding [2]uint32
	Message []byte
}

// MenuSelection is a client packet indicating a player's selection from
// one of the various menus, such as the ship or block list.
type MenuSelection struct {
	Header  BBHeader
	Unknown uint16
	MenuID  uint16
	ItemID  uint32
}

// List containing the available blocks on a ship.
type BlockList struct {
	Header   BBHeader
	Padding  [10]byte
	ShipName [32]byte
	Unknown  uint32
	Blocks   []Block
}

// Info about the available block servers.
type Block struct {
	Unknown   uint16
	BlockID   uint32
	Padding   uint16
	BlockName [36]byte
}
