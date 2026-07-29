package commands

const BlockListType = 0x07

type LobbyListEntry struct {
	MenuID  uint32 // Always 0x01 0x00 0x1A 0x00
	LobbyID uint32
	Padding uint32
}

// LobbyList is the list of available lobbies in a block for use in the teleporter.
const LobbyMenuType = 0x83

type LobbyList struct {
	Header  BBHeader
	Lobbies []LobbyListEntry
}

type ItemData struct {
	Data    [12]uint8
	ItemID  uint32
	MagData uint32
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

type PlayerInventory struct {
	NumItems        uint8
	HPFromMaterials uint8
	TPFromMaterials uint8
	Language        uint8
	Items           [30]PlayerInventoryItem
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
	AttackRange    uint32 // float32
	KnockbackRange uint32 // float32
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
	ProportionX       uint32 // float32
	ProportionY       uint32 // float32
}

type PlayerDisplayData struct {
	Stats      PlayerStats
	Visual     PlayerVisual
	DispName   [16]uint8  // UTF-16
	Config     [232]uint8 // Player key/action-bar config
	TechLevels [20]uint8  // Technique levels v1
}

type BankItem struct {
	Data      [12]uint8
	ItemID    uint32
	MagData   [4]uint8
	BankCount uint32
}

// GuildCard is the guild card data embedded in the character file.
type GuildCard struct {
	GuildCardNumber uint32
	Name            [48]uint8  // UTF-16, 0x18 chars
	TeamName        [32]uint8  // UTF-16, 0x10 chars
	Description     [176]uint8 // UTF-16, 0x58 chars; the player's guild card message
	Present         uint8
	Language        uint8
	SectionID       uint8
	Class           uint8
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
	PlayTime          uint32
	OptionFlags       uint32
	SaveCount         uint32

	// Quest flags.
	QuestFlags [512]uint8

	NumDeaths uint32

	// Bank.
	BankNumItems uint32
	BankMeseta   uint32
	Bank         [200]BankItem

	// Guild card (GuildCardBB, 0x0108 bytes).
	GuildCard GuildCard

	// Remaining character file sections (largely opaque).
	GuildCardUnknown     uint32
	SymbolChats          [0x4E0]uint8 // x12, 0x68 bytes each
	Shortcuts            [0xA40]uint8 // x16, 0xA4 bytes each
	AutoReply            [0x158]uint8 // UTF-16
	InfoBoard            [0x158]uint8 // UTF-16
	BattleRecords        [0x18]uint8
	Unknown4             [4]uint8
	ChallengeRecords     [0x140]uint8
	TechMenuShortcuts    [20]uint16
	ChoiceSearch         [0x18]uint8
	Unknown5             [16]uint8
	QuestCounters        [16]uint32
	OfflineBattleRecords [0x18]uint8
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

const JoinLobbyType = 0x67

type JoinLobby struct {
	Header BBHeader

	ClientID    uint8
	LeaderID    uint8
	DisableUDP  uint8 // Always 1.
	LobbyNumber uint8
	BlockNumber uint8
	Unused      uint8 // Dreamcast battle mode, according to newserv. Not relevant to BB.
	Event       uint8
	Unused2     [5]uint8 // 1 byte for something to do with voice chat and 4 bytes for some random seed.

	// Player entries.
	Entries [12]LobbyEntry
}
