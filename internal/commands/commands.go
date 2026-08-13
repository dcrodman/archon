package commands

// Definitions for the game commands exchanged between the client and server.
// Most types have some sort of comment alluding to their nature, but they are
// intentionally succinct. For (much) more information and a far better reference,
// refer to newserv: https://github.com/fuzziqersoftware/newserv/blob/master/src/CommandFormats.hh

// Welcome command with encryption vectors sent to the client upon initial connection.
const LoginWelcomeType = 0x03

type Welcome struct {
	Header       BBHeader
	Copyright    [96]byte
	ServerVector [48]byte
	ClientVector [48]byte
}

const DisconnectType = 0x05

// List containing the available blocks or ships in a menu.
const ShipMenuType = 0x07

type ShipMenu struct {
	Header  BBHeader
	Entries []ShipMenuEntry
}

type ShipMenuEntry struct {
	MenuID     uint32
	ItemID     uint32
	Difficulty uint8
	NumPlayers uint8
	Name       [32]uint8
	Episode    uint8
	Flags      uint8
}

// Client sends this to indicate a selection from the ship or block menu.
const MenuSelectionType = 0x10

// MenuSelection is a client command indicating a player's selection from
// one of the various menus, such as the ship or block list.
type MenuSelection struct {
	Header BBHeader
	MenuID uint32
	ItemID uint32
}

// Tells the client to connect to the address in the command. Used for proceeding through the patch
// and character selection steps as well as for joining or changing ships/blocks.
const RedirectType = 0x19

type Redirect struct {
	Header  BBHeader
	IPAddr  [4]uint8
	Port    uint16
	Padding uint16
}

// Message in a large text box, usually sent right before a disconnect.
const ClientMessageType = 0x1A

type ClientMessage struct {
	Header   BBHeader
	Language uint32
	Message  []byte
}

// Sent to a player when joining a lobby.
const JoinLobbyType = 0x67

type JoinLobby struct {
	Header BBHeader

	ClientID         uint8
	LeaderID         uint8
	DisableUDP       uint8 // Always 1.
	LobbyNumber      uint8
	BlockNumber      uint8
	EnableBattleMode uint8 // Dreamcast battle mode, according to newserv. Alway 1 in BB.
	Event            uint8
	EnableVoiceChat  uint8 // No idea, always 1.
	RandomSeed       uint32

	Unused [24]uint8 // Voice chat stuff that might be specific to xbox?

	// Player entries.
	Entries [12]LobbyEntry
}

type LobbyEntry struct {
	PlayerTag   uint32
	Guildcard   uint32
	TMGuildcard uint32
	TeamID      uint32
	Unknown     [12]uint8
	ClientID    uint32
	Name        [32]uint8 // UTF-18
	// Per newserv, Should be set to 1 to hide the "Press F1 for help".
	HideHelpPrompt uint32

	Inventory   PlayerInventory
	DisplayData PlayerDisplayData
}

// LobbyList is the list of available lobbies in a block for use in the teleporter.
const LobbyMenuType = 0x83

type LobbyMenu struct {
	Header  BBHeader
	Lobbies []LobbyListEntry
}

type LobbyListEntry struct {
	MenuID  uint32 // Always 0x01 0x00 0x1A 0x00
	LobbyID uint32
	Padding uint32
}

// Client sends this with credential information and metadata during the login process.
const LoginType = 0x93

// Login command sent to both the login and character servers.
type Login struct {
	Header        BBHeader
	Unknown       [8]uint8
	ClientVersion uint16
	Slot          uint32
	Phase         LoginPhase
	Unknown4      uint8 // It's not clear yet if this field is part of/related to the Phase field but it can take either 0 or e value on different clients
	TeamID        uint32
	Username      [16]uint8
	Padding       [32]uint8
	Password      [16]uint8
	Unknown3      [40]uint8
	HardwareInfo  [8]uint8
	Security      [40]uint8
	Padding2      uint32
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

// TODO: ???
const ShipListType = 0xA0

// Sends the client the current server timestamp.
const TimestampType = 0xB1

// Indicate the server's current time.
type Timestamp struct {
	Header    BBHeader
	Timestamp [28]byte
}

// Client sends this to request the keyboard configuration and player options.
const OptionsRequestType = 0xE0

// Options command containing keyboard and joystick config, team options, etc.
// Response to E0.
const OptionsType = 0xE2

type Options struct {
	Header BBHeader
	// Based on the key config structure from sylverant and newserv. KeyConfig
	// and JoystickConfig are saved in the database.
	//
	// Note: This command is shortened by dropping 4 bytes from TeamFlag in order
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

// Client sends this indicating a selection from the character menu.
const CharacterSelectionType = 0xE3

type CharacterSelection struct {
	Header    BBHeader
	Slot      uint32
	Selecting uint32
}

// Acknowledge a character selection from the client or indicate an error.
const CharacterSelectionAckType = 0xE4

type CharacterSelectionAck struct {
	Header BBHeader
	Slot   uint32
	Flag   uint32
}

const CharacterSummaryType = 0xE5

// Sent to the client for the selection menu and received for updating a character.
type CharacterSummary struct {
	Header    BBHeader
	Slot      uint32
	Character CharacterPreview
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

// Security command sent to the client to indicate the state of client login.
const SecurityType = 0xE6

type Security struct {
	Header       BBHeader
	ErrorCode    uint32
	PlayerTag    uint32
	Guildcard    uint32
	TeamID       uint32
	Config       ClientConfig
	Capabilities uint32
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

// SyncCharacter is the full dataset for one character.
const SyncCharacterType = 0xE7

type SyncCharacter struct {
	Header BBHeader

	Inventory   PlayerInventory
	DisplayData PlayerDisplayData

	// Character file metadata.
	ValidationFlags   uint32
	CreationTimestamp uint32
	Signature         uint32 // Always 0xC87ED5B1
	PlayTimeSeconds   uint32
	OptionFlags       uint32 // Always 0x00040058
	SaveCount         uint32

	// TODO: This definition is incomplete relative to newserv.
	QuestFlags [512]uint8
	NumDeaths  uint32

	Bank      Bank
	GuildCard GuildCard

	// Remaining character file sections (largely opaque).
	GuildCardUnknown uint32

	SymbolChats          [1248]uint8 // 12 * 0x68 bytes each
	Shortcuts            [2624]uint8 // 16 * 0xA4 bytes each
	AutoReply            [344]uint8  // UTF-16
	InfoBoard            [344]uint8  // UTF-16
	BattleRecords        [24]uint8
	Unknown4             [4]uint8
	ChallengeRecords     [320]uint8
	TechMenuShortcuts    [20]uint16
	ChoiceSearch         [24]uint8
	Unknown5             [16]uint8
	QuestCounters        [16]uint32
	OfflineBattleRecords [24]uint8
	Unknown6             [4]uint8

	SystemChecksum          uint32
	MusicVolume             int16
	SoundVolume             int8
	SystemLanguage          uint8
	ServerTimeDeltaFrames   int32
	UDPBehavior             uint16
	SurroundSoundEnabled    uint16
	EventFlags              [256]uint8
	SystemCreationTimestamp uint32

	// Key and joystick config.
	KeyConfig      [364]uint8
	JoystickConfig [56]uint8

	TeamMasterGuildCard uint32
	TeamID              uint32
	TeamUnknownA5       uint32
	TeamUnknownA6       uint32
	TeamPrivilegeLevel  uint8
	TeamMemberCount     uint8
	TeamUnknownA8       uint8
	TeamUnknownA9       uint8
	TeamName            [32]uint8 // UTF-16

	// Team flag image and reward flags.
	TeamFlagData    [2048]uint8
	TeamRewardFlags uint32
}

type PlayerInventory struct {
	NumItems        uint8
	HPFromMaterials uint8
	TPFromMaterials uint8
	Language        uint8
	Items           [30]PlayerInventoryItem
}

type PlayerInventoryItem struct {
	Present uint8
	// Newserv somehow unearthed that these four uint8s are used for some tricky
	// multipurpose backwards compatibility between games, which appear limited
	// to PSO V2 only and therefore we ignore them.
	Unknown [3]uint8
	Flags   uint32 // 0x08 is equipped
	Item    ItemData
}

type ItemData struct {
	Data    [12]uint8
	ItemID  uint32
	MagData uint32
}

type PlayerDisplayData struct {
	Stats      PlayerStats
	Visual     PlayerVisual
	DispName   [32]uint8  // UTF-16
	Config     [232]uint8 // Player key/action-bar config
	TechLevels [20]uint8  // Technique levels v1
}

type PlayerStats struct {
	ATP            uint16
	MST            uint16
	EVP            uint16
	HP             uint16
	DFP            uint16
	ATA            uint16
	LCK            uint16
	ESP            uint16
	AttackRange    float32
	KnockbackRange float32
	Level          uint32
	Experience     uint32
	Meseta         uint32
}

type PlayerVisual struct {
	Name              [16]uint8 // ASCII
	Unknown2          [8]uint8
	NameColor         uint32
	SkinID            uint8 // extra_model; NPC skin/appearance override
	Unknown3          [15]uint8
	NameColorChecksum uint32
	SectionID         uint8
	Class             uint8
	SkinFlag          uint8
	Version           uint8
	ClassFlags        uint32
	Costume           uint16
	Skin              uint16
	Face              uint16
	Head              uint16
	Hair              uint16
	HairColorRed      uint16
	HairColorGreen    uint16
	HairColorBlue     uint16
	ProportionX       float32
	ProportionY       float32
}

type Bank struct {
	NumItems uint32
	Meseta   uint32
	Bank     [200]BankItem
}

type BankItem struct {
	Data     [12]uint8
	ItemID   uint32
	MagData  [4]uint8
	Quantity uint16
	Present  uint16
}

// GuildCard is the guild card data embedded in the character file.
type GuildCard struct {
	GuildCardNumber uint32
	Name            [48]uint8  // UTF-16, 0x18 chars
	TeamName        [32]uint8  // UTF-16, 0x10 chars
	Description     [176]uint8 // UTF-16, 0x58 chars; the player's guild card messagess
	Present         uint8
	Language        uint8
	SectionID       uint8
	Class           uint8
}

// Client sends this to indicate whether a character should be recreated or updated.
// We ignore this in favor of using the Phase from the login packet.
const SetFlagType = 0xEC

const ScrollMessageType = 0xEE

// Scroll message the client should display on the ship select screen.
type ScrollMessage struct {
	Header  BBHeader
	Padding [2]uint32
	Message []byte
}

// Sent in response to 0x01E8 to acknowledge a checksum (really it's just ignored).
const ChecksumAckType = 0x02E8

type ChecksumAck struct {
	Header BBHeader
	Ack    uint32
}

// Client request for the guild card file checksum, which all servers (including Archon) ignore.
const ChecksumType = 0x01E8

// Client requesting guildcard data.
const GuildcardRequestType = 0x03E8

// Chunk header with info about the guildcard data we're about to send.
const GuildcardHeaderType = 0x01DC

type GuildcardHeader struct {
	Header   BBHeader
	Unknown  uint32
	Length   uint16
	Padding  uint16
	Checksum uint32
}

// Chunk of guildcard data sent by the server.
const GuildcardChunkType = 0x02DC

type GuildcardChunk struct {
	Header  BBHeader
	Unknown uint32
	Chunk   uint32
	Data    []uint8
}

// Client request for a chunk of guildcard data.
const GuildcardChunkReqType = 0x03DC

type GuildcardChunkRequest struct {
	Header         BBHeader
	Unknown        uint32
	ChunkRequested uint32
	Continue       uint32
}

// Parameter header containing details about the param files we're about to send.
const ParameterHeaderType = 0x01EB

type ParameterHeader struct {
	Header  BBHeader
	Entries []byte
}

// Chunk of parameter file data that the server is sending.
const ParameterChunkType = 0x02EB

type ParameterChunk struct {
	Header BBHeader
	Chunk  uint32
	Data   []byte
}

// Client request for a chunk of parameter data.
const ParameterChunkReqType = 0x03EB

// Client request for the parameter data header (01EB).
const ParameterHeaderReqType = 0x04EB
